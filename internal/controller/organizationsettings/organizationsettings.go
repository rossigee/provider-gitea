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

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/organizationsettings/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"strings"
	"time"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotOrganizationSettings   = "managed resource is not an OrganizationSettings custom resource"
	errGetOrganizationSettings   = "failed to get organization settings"
	errApplyOrganizationSettings = "failed to apply organization settings"
	errGetProviderConfig         = "failed to get provider config"
)

// A connector is expected to produce an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect returns an ExternalClient by:
// 1. Getting the provider config
// 2. Creating a Gitea API client
// 3. Returning an external client wrapping the API client
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return nil, errors.New(errNotOrganizationSettings)
	}

	// Get provider config reference from spec
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

	return &externalClient{client: conn}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it matches the managed resource's desired state.
type externalClient struct {
	client clients.Client
}

// buildUpdateRequest maps desired spec fields to an update request.
func buildUpdateRequest(fp v2.OrganizationSettingsParameters) *clients.UpdateOrganizationSettingsRequest {
	return &clients.UpdateOrganizationSettingsRequest{
		DefaultRepoPermission:    fp.DefaultRepoPermission,
		MembersCanCreateRepos:    fp.MembersCanCreateRepos,
		MembersCanCreatePrivate:  fp.MembersCanCreatePrivate,
		MembersCanCreateInternal: fp.MembersCanCreateInternal,
		MembersCanDeleteRepos:    fp.MembersCanDeleteRepos,
		MembersCanFork:           fp.MembersCanFork,
		MembersCanCreatePages:    fp.MembersCanCreatePages,
		DefaultRepoVisibility:    fp.DefaultRepoVisibility,
		RequireSignedCommits:     fp.RequireSignedCommits,
		EnableDependencyGraph:    fp.EnableDependencyGraph,
		AllowGitHooks:            fp.AllowGitHooks,
		AllowCustomGitHooks:      fp.AllowCustomGitHooks,
	}
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "organizationsettings.observe",
		tracing.SpanAttrs("organizationsettings", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationSettings)
	}

	org := cr.Spec.ForProvider.Organization
	if meta.GetExternalName(cr) == "" {
		meta.SetExternalName(cr, org)
	}

	settings, err := e.client.GetOrganizationSettings(ctx, org)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetOrganizationSettings)
	}

	// Update observed state
	now := time.Now().UTC().Format(time.RFC3339)
	cr.Status.AtProvider = v2.OrganizationSettingsObservation{
		LastUpdated: &now,
	}

	uo := isOrganizationSettingsUpToDate(cr, settings)

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func isOrganizationSettingsUpToDate(cr *v2.OrganizationSettings, settings *clients.OrganizationSettings) bool {
	fp := cr.Spec.ForProvider
	if fp.DefaultRepoPermission != nil && *fp.DefaultRepoPermission != settings.DefaultRepoPermission {
		return false
	}
	if fp.MembersCanCreateRepos != nil && *fp.MembersCanCreateRepos != settings.MembersCanCreateRepos {
		return false
	}
	if fp.MembersCanCreatePrivate != nil && *fp.MembersCanCreatePrivate != settings.MembersCanCreatePrivate {
		return false
	}
	if fp.MembersCanCreateInternal != nil && *fp.MembersCanCreateInternal != settings.MembersCanCreateInternal {
		return false
	}
	if fp.MembersCanDeleteRepos != nil && *fp.MembersCanDeleteRepos != settings.MembersCanDeleteRepos {
		return false
	}
	if fp.MembersCanFork != nil && *fp.MembersCanFork != settings.MembersCanFork {
		return false
	}
	if fp.MembersCanCreatePages != nil && *fp.MembersCanCreatePages != settings.MembersCanCreatePages {
		return false
	}
	if fp.DefaultRepoVisibility != nil && *fp.DefaultRepoVisibility != settings.DefaultRepoVisibility {
		return false
	}
	if fp.RequireSignedCommits != nil && *fp.RequireSignedCommits != settings.RequireSignedCommits {
		return false
	}
	if fp.EnableDependencyGraph != nil && *fp.EnableDependencyGraph != settings.EnableDependencyGraph {
		return false
	}
	if fp.AllowGitHooks != nil && *fp.AllowGitHooks != settings.AllowGitHooks {
		return false
	}
	if fp.AllowCustomGitHooks != nil && *fp.AllowCustomGitHooks != settings.AllowCustomGitHooks {
		return false
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "organizationsettings.create",
		tracing.SpanAttrs("organizationsettings", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationSettings)
	}

	// Settings are a singleton per organization: create applies them.
	if _, err := e.client.UpdateOrganizationSettings(ctx, cr.Spec.ForProvider.Organization, buildUpdateRequest(cr.Spec.ForProvider)); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errApplyOrganizationSettings)
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.Organization)

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "organizationsettings.update",
		tracing.SpanAttrs("organizationsettings", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationSettings)
	}

	if _, err := e.client.UpdateOrganizationSettings(ctx, cr.Spec.ForProvider.Organization, buildUpdateRequest(cr.Spec.ForProvider)); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errApplyOrganizationSettings)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	// Settings are a singleton per organization with no delete endpoint;
	// deleting the CR only removes local state.
	if _, ok := mg.(*v2.OrganizationSettings); !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganizationSettings)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.OrganizationSettingsKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.OrganizationSettingsGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.OrganizationSettings{}).
		Complete(r)
}
