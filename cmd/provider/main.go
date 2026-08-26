/*
Copyright 2024 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/alecthomas/kingpin/v2"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	branchprotectionv2 "github.com/rossigee/provider-gitea/apis/branchprotection/v2"
	organizationv2 "github.com/rossigee/provider-gitea/apis/organization/v2"
	repositoryv2 "github.com/rossigee/provider-gitea/apis/repository/v2"
	repositorykeyv2 "github.com/rossigee/provider-gitea/apis/repositorykey/v2"
	repositorysecretv2 "github.com/rossigee/provider-gitea/apis/repositorysecret/v2"
	userv2 "github.com/rossigee/provider-gitea/apis/user/v2"
	webhookv2 "github.com/rossigee/provider-gitea/apis/webhook/v2"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/rossigee/provider-gitea/apis"
	giteacontroller "github.com/rossigee/provider-gitea/internal/controller"
	"github.com/rossigee/provider-gitea/internal/tracing"
	"github.com/rossigee/provider-gitea/internal/version"
)

func main() {
	var (
		app                      = kingpin.New(filepath.Base(os.Args[0]), "Gitea Crossplane provider").DefaultEnvars()
		debug                    = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncInterval             = app.Flag("sync", "Sync interval controls how often all resources will be double checked for drift.").Short('s').Default("1h").Duration()
		pollInterval             = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for drift.").Default("1m").Duration()
		leaderElection           = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").Bool()
		maxReconcileRate         = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("100").Int()
		_                        = app.Flag("namespace", "Namespace used to set as default scope in default secret store config.").Default("crossplane-system").Envar("POD_NAMESPACE").String()
		pollStateMetricInterval  = app.Flag("poll-state-metric", "State metric recording interval").Default("5s").Duration()
		metricsBindAddress       = app.Flag("metrics-bind-address", "The address the metrics endpoint binds to.").Default(":8080").String()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("provider-gitea"))

	shutdownTracing := tracing.Init("provider-gitea")
	defer shutdownTracing(context.Background())

	// The controller-runtime runs with a no-op logger by default. Always set it
	// to ensure logs are displayed. Use higher verbosity level in production.
	ctrl.SetLogger(zl)

	log.Info("Provider starting up",
		"provider", "provider-gitea",
		"version", version.Version,
		"go-version", runtime.Version(),
		"platform", runtime.GOOS+"/"+runtime.GOARCH,
		"sync-interval", syncInterval.String(),
		"poll-interval", pollInterval.String(),
		"max-reconcile-rate", *maxReconcileRate,
		"leader-election", *leaderElection,
		"debug-mode", *debug)

	s := apimachineryruntime.NewScheme()
	kingpin.FatalIfError(scheme.AddToScheme(s), "Cannot add k8s types to scheme")
	kingpin.FatalIfError(apis.AddToScheme(s), "Cannot add Gitea APIs to scheme")

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	// Get the manager options
	options := ctrl.Options{
		Scheme:           s,
		LeaderElection:   *leaderElection,
		LeaderElectionID: "crossplane-leader-election-provider-gitea",
		Cache: cache.Options{
			SyncPeriod: syncInterval,
		},
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:              func() *time.Duration { d := 50 * time.Second; return &d }(),
		Metrics: metricserver.Options{
			BindAddress: *metricsBindAddress,
		},
	}

	mgr, err := ctrl.NewManager(cfg, options)
	kingpin.FatalIfError(err, "Cannot create controller manager")

	mrStateMetrics := statemetrics.NewMRStateMetrics()
	metrics.Registry.MustRegister(mrStateMetrics)

	mo := controller.MetricOptions{
		PollStateMetricInterval: *pollStateMetricInterval,
		MRStateMetrics:          mrStateMetrics,
	}

	featureFlags := &feature.Flags{}
	o := controller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
		Features:                featureFlags,
		MetricOptions:           &mo,
	}

	kingpin.FatalIfError(giteacontroller.Setup(mgr, o), "Cannot setup Gitea controllers")

	// Register state metrics for managed resources
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &repositoryv2.RepositoryList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Repository")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &repositorykeyv2.RepositoryKeyList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for RepositoryKey")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &repositorysecretv2.RepositorySecretList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for RepositorySecret")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &organizationv2.OrganizationList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Organization")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &userv2.UserList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for User")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &webhookv2.WebhookList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Webhook")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &branchprotectionv2.BranchProtectionList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for BranchProtection")

	kingpin.FatalIfError(mgr.AddHealthzCheck("healthz", healthz.Ping), "Cannot add health check")
	kingpin.FatalIfError(mgr.AddReadyzCheck("readyz", healthz.Ping), "Cannot add ready check")

	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
