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

// Package organizationsettings implements the Crossplane managed-resource
// reconciler for the Gitea OrganizationSettings resource. It follows the
// canonical pattern documented in
// internal/controller/repository/repository.go and the lessons in
// crossplane-provider-template dev/docs/09-lessons-learned.md.
//
// OrganizationSettings is an UPDATE-ONLY resource: the backend has no Create
// and no Delete for org-wide settings (settings exist for the lifetime of the
// org). The resource is therefore modelled as:
//   - Observe: GET the settings; if found, exists:true + Available; compute
//     drift against the desired spec.
//   - Create:  apply the desired settings via Update, then pin external-name=org.
//   - Update:  apply the desired settings via Update.
//   - Delete:  no-op (settings can't be deleted).
//
// Identity is the org name. It comes from cr.Spec.ForProvider.Organization, so
// the very first Observe (empty external-name) still keys off the org and
// adopts the existing settings if present.
package organizationsettings

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/organizationsettings/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotOrganizationSettings = "managed resource is not an OrganizationSettings custom resource"
	errGetSettings             = "failed to get organization settings"
	errUpdateSettings          = "failed to update organization settings"
	errGetProviderConfig       = "failed to get provider config"
)

// Setup adds a controller that reconciles OrganizationSettings managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.OrganizationSettingsKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}
	// Honour spec.managementPolicies (ObserveOnly, no-delete, pause, ...) when the
	// operator runs the provider with --enable-management-policies.
	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.OrganizationSettingsGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.OrganizationSettings{}).
		// A non-nil rate limiter is mandatory: ratelimiter.Reconciler.When()
		// dereferences it every reconcile, so a nil limiter panics on the first
		// event (lesson #1). o.ForControllerRuntime() also carries
		// MaxConcurrentReconciles through; without WithOptions both are dropped.
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return nil, errors.New(errNotOrganizationSettings)
	}

	pcRef := cr.Spec.ProviderConfigReference
	if pcRef == nil {
		return nil, errors.New("providerConfigRef is required")
	}

	var pc v1beta1.ProviderConfig
	if err := c.kube.Get(ctx, client.ObjectKey{
		Namespace: cr.GetNamespace(),
		Name:      pcRef.Name,
	}, &pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	conn, err := clients.NewClient(ctx, &pc, c.kube)
	if err != nil {
		return nil, err
	}

	return &external{client: conn}, nil
}

// external observes/applies the backend organization settings.
type external struct {
	client clients.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationSettings)
	}

	// Identity is the org name from spec. Unlike a create-able resource, settings
	// have no create call, so on the very first Observe (empty external-name) we
	// still GET by org and adopt the existing settings if present.
	org := cr.Spec.ForProvider.Organization
	if org == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	settings, err := e.client.GetOrganizationSettings(ctx, org)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). Real failures (auth/network/5xx) must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetSettings)
	}

	upToDate := settingsUpToDate(cr, settings)

	// crossplane-runtime v2's managed reconciler no longer auto-sets
	// Available(); readiness is the provider's job (lesson #2/#6). Set Available
	// on the exists path; drift is signalled via ResourceUpToDate.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// settingsUpToDate reports whether the observed backend settings match the
// managed fields of the desired spec. Only fields this provider pushes on
// Update are compared; a nil desired pointer means "do not manage".
func settingsUpToDate(cr *v2.OrganizationSettings, observed *clients.OrganizationSettings) bool {
	p := cr.Spec.ForProvider
	if p.DefaultRepoPermission != nil && *p.DefaultRepoPermission != observed.DefaultRepoPermission {
		return false
	}
	if p.MembersCanCreateRepos != nil && *p.MembersCanCreateRepos != observed.MembersCanCreateRepos {
		return false
	}
	if p.MembersCanCreatePrivate != nil && *p.MembersCanCreatePrivate != observed.MembersCanCreatePrivate {
		return false
	}
	if p.MembersCanCreateInternal != nil && *p.MembersCanCreateInternal != observed.MembersCanCreateInternal {
		return false
	}
	if p.MembersCanDeleteRepos != nil && *p.MembersCanDeleteRepos != observed.MembersCanDeleteRepos {
		return false
	}
	if p.MembersCanFork != nil && *p.MembersCanFork != observed.MembersCanFork {
		return false
	}
	if p.MembersCanCreatePages != nil && *p.MembersCanCreatePages != observed.MembersCanCreatePages {
		return false
	}
	if p.DefaultRepoVisibility != nil && *p.DefaultRepoVisibility != observed.DefaultRepoVisibility {
		return false
	}
	if p.RequireSignedCommits != nil && *p.RequireSignedCommits != observed.RequireSignedCommits {
		return false
	}
	if p.EnableDependencyGraph != nil && *p.EnableDependencyGraph != observed.EnableDependencyGraph {
		return false
	}
	if p.AllowGitHooks != nil && *p.AllowGitHooks != observed.AllowGitHooks {
		return false
	}
	if p.AllowCustomGitHooks != nil && *p.AllowCustomGitHooks != observed.AllowCustomGitHooks {
		return false
	}
	return true
}

func buildUpdateRequest(cr *v2.OrganizationSettings) *clients.UpdateOrganizationSettingsRequest {
	p := cr.Spec.ForProvider
	return &clients.UpdateOrganizationSettingsRequest{
		DefaultRepoPermission:    p.DefaultRepoPermission,
		MembersCanCreateRepos:    p.MembersCanCreateRepos,
		MembersCanCreatePrivate:  p.MembersCanCreatePrivate,
		MembersCanCreateInternal: p.MembersCanCreateInternal,
		MembersCanDeleteRepos:    p.MembersCanDeleteRepos,
		MembersCanFork:           p.MembersCanFork,
		MembersCanCreatePages:    p.MembersCanCreatePages,
		DefaultRepoVisibility:    p.DefaultRepoVisibility,
		RequireSignedCommits:     p.RequireSignedCommits,
		EnableDependencyGraph:    p.EnableDependencyGraph,
		AllowGitHooks:            p.AllowGitHooks,
		AllowCustomGitHooks:      p.AllowCustomGitHooks,
	}
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationSettings)
	}
	cr.SetConditions(xpv1.Creating())

	// There is no backend "create settings" call — Create applies the desired
	// settings via Update and then pins the external name to the org (identity).
	org := cr.Spec.ForProvider.Organization
	if _, err := e.client.UpdateOrganizationSettings(ctx, org, buildUpdateRequest(cr)); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUpdateSettings)
	}

	meta.SetExternalName(cr, org)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationSettings)
	}

	// The org is the immutable identity from spec — use it directly. (The
	// external-name annotation defaults to metadata.name, which is NOT the org,
	// so keying Update off it 404s with "GetOrgByName <metadata.name>".)
	org := cr.Spec.ForProvider.Organization

	if _, err := e.client.UpdateOrganizationSettings(ctx, org, buildUpdateRequest(cr)); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateSettings)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(_ context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganizationSettings)
	}
	cr.SetConditions(xpv1.Deleting())

	// No-op: organization-wide settings cannot be deleted — they live for the
	// lifetime of the organization, and the backend exposes no delete endpoint.
	// Returning success lets the finalizer release (cf. idempotent delete,
	// lesson #16).
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
