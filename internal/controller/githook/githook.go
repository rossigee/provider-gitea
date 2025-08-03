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

package githook

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

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

	"github.com/crossplane-contrib/provider-gitea/apis/githook/v1alpha1"
	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
	giteaclients "github.com/crossplane-contrib/provider-gitea/internal/clients"
)

const (
	errNotGitHook     = "managed resource is not a GitHook custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
	errNewClient      = "cannot create new Service"
	errGetGitHook     = "cannot get git hook"
	errCreateGitHook  = "cannot create git hook"
	errUpdateGitHook  = "cannot update git hook"
	errDeleteGitHook  = "cannot delete git hook"
)

// Setup adds a controller that reconciles GitHook managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.GitHookKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.GitHookGroupVersionKind),
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
		For(&v1alpha1.GitHook{}).
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
	cr, ok := mg.(*v1alpha1.GitHook)
	if !ok {
		return nil, errors.New(errNotGitHook)
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
	cr, ok := mg.(*v1alpha1.GitHook)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotGitHook)
	}

	// External name format: repository/hookType (e.g., "myorg/myrepo/pre-receive")
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		externalName = fmt.Sprintf("%s/%s", cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.HookType)
		meta.SetExternalName(cr, externalName)
	}

	hook, err := c.client.GetGitHook(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.HookType)
	if err != nil {
		// If hook doesn't exist, it needs to be created
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Calculate content hash for drift detection
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(hook.Content)))

	// Update observed state
	lastUpdated := time.Now().Format(time.RFC3339)
	cr.Status.AtProvider = v1alpha1.GitHookObservation{
		Name:        &hook.Name,
		LastUpdated: &lastUpdated,
		ContentHash: &contentHash,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, hook),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.GitHook)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotGitHook)
	}

	cr.SetConditions(xpv1.Creating())

	isActive := true
	if cr.Spec.ForProvider.IsActive != nil {
		isActive = *cr.Spec.ForProvider.IsActive
	}

	req := &giteaclients.CreateGitHookRequest{
		HookType: cr.Spec.ForProvider.HookType,
		Content:  cr.Spec.ForProvider.Content,
		IsActive: isActive,
	}

	_, err := c.client.CreateGitHook(ctx, cr.Spec.ForProvider.Repository, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateGitHook)
	}

	// Set external name to repository/hookType format
	externalName := fmt.Sprintf("%s/%s", cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.HookType)
	meta.SetExternalName(cr, externalName)

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.GitHook)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotGitHook)
	}

	isActive := true
	if cr.Spec.ForProvider.IsActive != nil {
		isActive = *cr.Spec.ForProvider.IsActive
	}

	req := &giteaclients.UpdateGitHookRequest{
		Content:  cr.Spec.ForProvider.Content,
		IsActive: isActive,
	}

	_, err := c.client.UpdateGitHook(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.HookType, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateGitHook)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.GitHook)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotGitHook)
	}

	err := c.client.DeleteGitHook(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.HookType)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteGitHook)
	}

	return managed.ExternalDelete{}, nil
}

// isUpToDate checks if the git hook is up to date with the desired state
func (c *external) isUpToDate(cr *v1alpha1.GitHook, hook *giteaclients.GitHook) bool {
	// Check content
	if cr.Spec.ForProvider.Content != hook.Content {
		return false
	}

	// Check active state
	desiredActive := true
	if cr.Spec.ForProvider.IsActive != nil {
		desiredActive = *cr.Spec.ForProvider.IsActive
	}
	if desiredActive != hook.IsActive {
		return false
	}

	return true
}