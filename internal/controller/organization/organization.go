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

// Package organization implements the Crossplane managed-resource reconciler
// for the Gitea Organization resource. It follows the canonical pattern of the
// repository controller (see internal/controller/repository/repository.go and
// crossplane-provider-template dev/docs/09-lessons-learned.md).
package organization

import (
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/organization/v1beta1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotOrganization    = "managed resource is not an Organization custom resource"
	errGetOrganization    = "failed to get organization"
	errCreateOrganization = "failed to create organization"
	errUpdateOrganization = "failed to update organization"
	errDeleteOrganization = "failed to delete organization"
	errGetProviderConfig  = "failed to get provider config"
	errExternalName       = "invalid external-name, expected the organization username"
	errTrackUsage         = "cannot track ProviderConfig usage"
)

// Setup adds a controller that reconciles Organization managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.OrganizationKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithPollJitterHook(o.PollInterval / 10),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}
	// Honour spec.managementPolicies (ObserveOnly, no-delete, pause, ...) when the
	// operator runs the provider with --enable-management-policies.
	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.OrganizationGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.Organization{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube  client.Client
	usage resource.ModernTracker
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.Organization)
	if !ok {
		return nil, errors.New(errNotOrganization)
	}

	if err := c.usage.Track(ctx, cr); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
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

// external observes/creates/updates/deletes the backend organization.
type external struct {
	client clients.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganization)
	}

	// Identity is the external-name annotation (the org username), authoritative
	// for Observe/Update/Delete (lesson #14). Empty -> not created; no GET.
	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	org, err := e.client.GetOrganization(ctx, name)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetOrganization)
	}

	cr.Status.AtProvider = v2.OrganizationObservation{
		ID:                        &org.ID,
		AvatarURL:                 &org.AvatarURL,
		Email:                     &org.Email,
		RepoAdminChangeTeamAccess: &org.RepoAdminChangeTeamAccess,
	}

	upToDate := organizationUpToDate(cr, org)

	// crossplane-runtime v2 no longer auto-sets Available(); the provider must
	// set readiness on the exists path (lesson #2/#6). Drift is carried by
	// ResourceUpToDate, never by withholding Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// organizationUpToDate compares only the mutable fields this provider pushes on
// Update; a nil desired pointer means "do not manage", so it never drifts.
func organizationUpToDate(cr *v2.Organization, observed *clients.Organization) bool {
	p := cr.Spec.ForProvider
	if p.FullName != nil && *p.FullName != observed.FullName {
		return false
	}
	if p.Description != nil && *p.Description != observed.Description {
		return false
	}
	if p.Website != nil && *p.Website != observed.Website {
		return false
	}
	if p.Location != nil && *p.Location != observed.Location {
		return false
	}
	if p.Visibility != nil && *p.Visibility != observed.Visibility {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganization)
	}
	cr.SetConditions(xpv1.Creating())

	createReq := &clients.CreateOrganizationRequest{
		Username:    cr.Spec.ForProvider.Username,
		Name:        ptr.Deref(cr.Spec.ForProvider.Name, ""),
		FullName:    ptr.Deref(cr.Spec.ForProvider.FullName, ""),
		Description: ptr.Deref(cr.Spec.ForProvider.Description, ""),
		Website:     ptr.Deref(cr.Spec.ForProvider.Website, ""),
		Location:    ptr.Deref(cr.Spec.ForProvider.Location, ""),
		Visibility:  ptr.Deref(cr.Spec.ForProvider.Visibility, ""),
	}

	org, err := e.client.CreateOrganization(ctx, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateOrganization)
	}

	// Pin external-name to the authoritative username from the backend response
	// (lesson #3/#7/#14), never a spec guess.
	meta.SetExternalName(cr, org.Username)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganization)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	updateReq := &clients.UpdateOrganizationRequest{
		FullName:    cr.Spec.ForProvider.FullName,
		Description: cr.Spec.ForProvider.Description,
		Website:     cr.Spec.ForProvider.Website,
		Location:    cr.Spec.ForProvider.Location,
		Visibility:  cr.Spec.ForProvider.Visibility,
	}

	if _, err := e.client.UpdateOrganization(ctx, name, updateReq); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganization)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganization)
	}
	cr.SetConditions(xpv1.Deleting())

	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteOrganization(ctx, name)
	// Treat an already-absent organization as a successful delete (lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteOrganization)
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}
