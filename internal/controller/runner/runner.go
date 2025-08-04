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

package runner

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/runner/v1alpha1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRunner    = "managed resource is not a Runner custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create new Service"
	errCreateRunner = "cannot create runner"
	errUpdateRunner = "cannot update runner"
	errDeleteRunner = "cannot delete runner"
	errGetRunner    = "cannot get runner"
)

// Setup adds a controller that reconciles Runner managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.RunnerKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RunnerGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.Runner{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Runner)
	if !ok {
		return nil, errors.New(errNotRunner)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	client, err := giteaclients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: client}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client giteaclients.Client
}

func (c *external) Disconnect(ctx context.Context) error {
	// No persistent connection to disconnect
	return nil
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Runner)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRunner)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Parse external name format: scope:scopeValue:runnerID
	scope, scopeValue, runnerIDStr, err := parseExternalName(externalName)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	runnerID, err := strconv.ParseInt(runnerIDStr, 10, 64)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "invalid runner ID")
	}

	runner, err := c.client.GetRunner(ctx, scope, scopeValue, runnerID)
	if err != nil {
		// If runner doesn't exist, return that it needs to be created
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.RunnerObservation{
		ID:         &runner.ID,
		Name:       &runner.Name,
		Status:     &runner.Status,
		Labels:     runner.Labels,
		Version:    &runner.Version,
		UUID:       &runner.UUID,
		LastOnline: &runner.LastOnline,
		CreatedAt:  &runner.CreatedAt,
		UpdatedAt:  &runner.UpdatedAt,
		Scope:      &runner.Scope,
		ScopeValue: &runner.ScopeValue,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, runner),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Runner)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRunner)
	}

	cr.SetConditions(xpv1.Creating())

	req := &giteaclients.CreateRunnerRequest{
		Name:   cr.Spec.ForProvider.Name,
		Labels: cr.Spec.ForProvider.Labels,
	}

	if cr.Spec.ForProvider.Description != nil {
		req.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.RunnerGroupID != nil {
		req.RunnerGroupID = cr.Spec.ForProvider.RunnerGroupID
	}

	scope := cr.Spec.ForProvider.Scope
	scopeValue := ""
	if cr.Spec.ForProvider.ScopeValue != nil {
		scopeValue = *cr.Spec.ForProvider.ScopeValue
	}

	runner, err := c.client.CreateRunner(ctx, scope, scopeValue, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRunner)
	}

	// Set external name for the created runner
	externalName := fmt.Sprintf("%s:%s:%d", scope, scopeValue, runner.ID)
	meta.SetExternalName(cr, externalName)

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Runner)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRunner)
	}

	scope, scopeValue, runnerIDStr, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	runnerID, err := strconv.ParseInt(runnerIDStr, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "invalid runner ID")
	}

	req := &giteaclients.UpdateRunnerRequest{
		Name:   &cr.Spec.ForProvider.Name,
		Labels: cr.Spec.ForProvider.Labels,
	}

	if cr.Spec.ForProvider.Description != nil {
		req.Description = cr.Spec.ForProvider.Description
	}

	_, err = c.client.UpdateRunner(ctx, scope, scopeValue, runnerID, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRunner)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Runner)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRunner)
	}

	cr.SetConditions(xpv1.Deleting())

	scope, scopeValue, runnerIDStr, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	runnerID, err := strconv.ParseInt(runnerIDStr, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "invalid runner ID")
	}

	err = c.client.DeleteRunner(ctx, scope, scopeValue, runnerID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRunner)
	}

	return managed.ExternalDelete{}, nil
}

// Helper functions

func parseExternalName(externalName string) (scope, scopeValue, runnerID string, err error) {
	parts := strings.Split(externalName, ":")
	if len(parts) != 3 {
		return "", "", "", errors.New("external name must be in format 'scope:scopeValue:runnerID'")
	}

	return parts[0], parts[1], parts[2], nil
}

func (c *external) isUpToDate(cr *v1alpha1.Runner, runner *giteaclients.Runner) bool {
	if cr.Spec.ForProvider.Name != runner.Name {
		return false
	}

	if len(cr.Spec.ForProvider.Labels) != len(runner.Labels) {
		return false
	}

	for i, label := range cr.Spec.ForProvider.Labels {
		if i >= len(runner.Labels) || label != runner.Labels[i] {
			return false
		}
	}

	if cr.Spec.ForProvider.Description != nil && *cr.Spec.ForProvider.Description != runner.Description {
		return false
	}

	return true
}
