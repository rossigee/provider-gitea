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

package organizationsettings

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/organizationsettings/v1alpha1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotOrganizationSettings = "managed resource is not an OrganizationSettings custom resource"
	errTrackPCUsage            = "cannot track ProviderConfig usage"
	errGetPC                   = "cannot get ProviderConfig"
	errGetCreds                = "cannot get credentials"
	errNewClient               = "cannot create new Service"
	errGetOrgSettings          = "cannot get organization settings"
	errUpdateOrgSettings       = "cannot update organization settings"
)

// Setup adds a controller that reconciles OrganizationSettings managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.OrganizationSettingsKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.OrganizationSettingsGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.TrackerFn(func(ctx context.Context, mg resource.Managed) error { return nil }),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.OrganizationSettings{}).
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
	cr, ok := mg.(*v1alpha1.OrganizationSettings)
	if !ok {
		return nil, errors.New(errNotOrganizationSettings)
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
	cr, ok := mg.(*v1alpha1.OrganizationSettings)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationSettings)
	}

	// External name is the organization name
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		// Use the organization from the spec as external name if not set
		meta.SetExternalName(cr, cr.Spec.ForProvider.Organization)
		externalName = cr.Spec.ForProvider.Organization
	}

	settings, err := c.client.GetOrganizationSettings(ctx, externalName)
	if err != nil {
		// If settings don't exist, they need to be "created" (first time configuration)
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.OrganizationSettingsObservation{
		AppliedSettings: &v1alpha1.AppliedOrganizationSettings{
			DefaultRepoPermission:    &settings.DefaultRepoPermission,
			MembersCanCreateRepos:    &settings.MembersCanCreateRepos,
			MembersCanCreatePrivate:  &settings.MembersCanCreatePrivate,
			MembersCanCreateInternal: &settings.MembersCanCreateInternal,
			MembersCanDeleteRepos:    &settings.MembersCanDeleteRepos,
			MembersCanFork:           &settings.MembersCanFork,
			MembersCanCreatePages:    &settings.MembersCanCreatePages,
			DefaultRepoVisibility:    &settings.DefaultRepoVisibility,
			RequireSignedCommits:     &settings.RequireSignedCommits,
			EnableDependencyGraph:    &settings.EnableDependencyGraph,
			AllowGitHooks:            &settings.AllowGitHooks,
			AllowCustomGitHooks:      &settings.AllowCustomGitHooks,
		},
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, settings),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.OrganizationSettings)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationSettings)
	}

	cr.SetConditions(xpv1.Creating())

	req := c.buildUpdateRequest(cr)

	_, err := c.client.UpdateOrganizationSettings(ctx, cr.Spec.ForProvider.Organization, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUpdateOrgSettings)
	}

	// Set external name to the organization name
	meta.SetExternalName(cr, cr.Spec.ForProvider.Organization)

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.OrganizationSettings)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationSettings)
	}

	req := c.buildUpdateRequest(cr)

	_, err := c.client.UpdateOrganizationSettings(ctx, cr.Spec.ForProvider.Organization, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrgSettings)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	// OrganizationSettings cannot be "deleted" - they can only be reset to defaults
	// This is a no-op since organization settings always exist
	return managed.ExternalDelete{}, nil
}

// buildUpdateRequest creates an update request from the CR spec
func (c *external) buildUpdateRequest(cr *v1alpha1.OrganizationSettings) *giteaclients.UpdateOrganizationSettingsRequest {
	req := &giteaclients.UpdateOrganizationSettingsRequest{}

	if cr.Spec.ForProvider.DefaultRepoPermission != nil {
		req.DefaultRepoPermission = cr.Spec.ForProvider.DefaultRepoPermission
	}
	if cr.Spec.ForProvider.MembersCanCreateRepos != nil {
		req.MembersCanCreateRepos = cr.Spec.ForProvider.MembersCanCreateRepos
	}
	if cr.Spec.ForProvider.MembersCanCreatePrivate != nil {
		req.MembersCanCreatePrivate = cr.Spec.ForProvider.MembersCanCreatePrivate
	}
	if cr.Spec.ForProvider.MembersCanCreateInternal != nil {
		req.MembersCanCreateInternal = cr.Spec.ForProvider.MembersCanCreateInternal
	}
	if cr.Spec.ForProvider.MembersCanDeleteRepos != nil {
		req.MembersCanDeleteRepos = cr.Spec.ForProvider.MembersCanDeleteRepos
	}
	if cr.Spec.ForProvider.MembersCanFork != nil {
		req.MembersCanFork = cr.Spec.ForProvider.MembersCanFork
	}
	if cr.Spec.ForProvider.MembersCanCreatePages != nil {
		req.MembersCanCreatePages = cr.Spec.ForProvider.MembersCanCreatePages
	}
	if cr.Spec.ForProvider.DefaultRepoVisibility != nil {
		req.DefaultRepoVisibility = cr.Spec.ForProvider.DefaultRepoVisibility
	}
	if cr.Spec.ForProvider.RequireSignedCommits != nil {
		req.RequireSignedCommits = cr.Spec.ForProvider.RequireSignedCommits
	}
	if cr.Spec.ForProvider.EnableDependencyGraph != nil {
		req.EnableDependencyGraph = cr.Spec.ForProvider.EnableDependencyGraph
	}
	if cr.Spec.ForProvider.AllowGitHooks != nil {
		req.AllowGitHooks = cr.Spec.ForProvider.AllowGitHooks
	}
	if cr.Spec.ForProvider.AllowCustomGitHooks != nil {
		req.AllowCustomGitHooks = cr.Spec.ForProvider.AllowCustomGitHooks
	}

	return req
}

// isUpToDate checks if the organization settings are up to date with the desired state
func (c *external) isUpToDate(cr *v1alpha1.OrganizationSettings, settings *giteaclients.OrganizationSettings) bool {
	if cr.Spec.ForProvider.DefaultRepoPermission != nil && *cr.Spec.ForProvider.DefaultRepoPermission != settings.DefaultRepoPermission {
		return false
	}
	if cr.Spec.ForProvider.MembersCanCreateRepos != nil && *cr.Spec.ForProvider.MembersCanCreateRepos != settings.MembersCanCreateRepos {
		return false
	}
	if cr.Spec.ForProvider.MembersCanCreatePrivate != nil && *cr.Spec.ForProvider.MembersCanCreatePrivate != settings.MembersCanCreatePrivate {
		return false
	}
	if cr.Spec.ForProvider.MembersCanCreateInternal != nil && *cr.Spec.ForProvider.MembersCanCreateInternal != settings.MembersCanCreateInternal {
		return false
	}
	if cr.Spec.ForProvider.MembersCanDeleteRepos != nil && *cr.Spec.ForProvider.MembersCanDeleteRepos != settings.MembersCanDeleteRepos {
		return false
	}
	if cr.Spec.ForProvider.MembersCanFork != nil && *cr.Spec.ForProvider.MembersCanFork != settings.MembersCanFork {
		return false
	}
	if cr.Spec.ForProvider.MembersCanCreatePages != nil && *cr.Spec.ForProvider.MembersCanCreatePages != settings.MembersCanCreatePages {
		return false
	}
	if cr.Spec.ForProvider.DefaultRepoVisibility != nil && *cr.Spec.ForProvider.DefaultRepoVisibility != settings.DefaultRepoVisibility {
		return false
	}
	if cr.Spec.ForProvider.RequireSignedCommits != nil && *cr.Spec.ForProvider.RequireSignedCommits != settings.RequireSignedCommits {
		return false
	}
	if cr.Spec.ForProvider.EnableDependencyGraph != nil && *cr.Spec.ForProvider.EnableDependencyGraph != settings.EnableDependencyGraph {
		return false
	}
	if cr.Spec.ForProvider.AllowGitHooks != nil && *cr.Spec.ForProvider.AllowGitHooks != settings.AllowGitHooks {
		return false
	}
	if cr.Spec.ForProvider.AllowCustomGitHooks != nil && *cr.Spec.ForProvider.AllowCustomGitHooks != settings.AllowCustomGitHooks {
		return false
	}

	return true
}
