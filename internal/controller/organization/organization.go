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

package organization

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/organization/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"
)

const (
	errNotOrganization     = "managed resource is not an Organization custom resource"
	errGetOrganization     = "failed to get organization"
	errCreateOrganization  = "failed to create organization"
	errUpdateOrganization  = "failed to update organization"
	errDeleteOrganization  = "failed to delete organization"
	errGetProviderConfig   = "failed to get provider config"
	errInvalidExternalName = "invalid external-name, expected organization name"
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
	cr, ok := mg.(*v2.Organization)
	if !ok {
		return nil, errors.New(errNotOrganization)
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

	return &externalClient{client: conn}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it matches the managed resource's desired state.
type externalClient struct {
	client clients.Client
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "organization.observe",
		tracing.SpanAttrs("organization", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganization)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	org, err := e.client.GetOrganization(ctx, externalName)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetOrganization)
	}

	repoAdminChangeTeamAccess := org.RepoAdminChangeTeamAccess
	publicRepos := org.NumRepos
	privateRepos := org.NumPrivateRepos
	members := org.NumMembers
	teams := org.NumTeams

	cr.Status.AtProvider = v2.OrganizationObservation{
		ID:                        &org.ID,
		AvatarURL:                 &org.AvatarURL,
		Email:                     &org.Email,
		RepoAdminChangeTeamAccess: &repoAdminChangeTeamAccess,
		PublicRepos:               &publicRepos,
		PrivateRepos:              &privateRepos,
		Members:                   &members,
		Teams:                     &teams,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: isOrganizationUpToDate(cr, org)}, nil
}

func isOrganizationUpToDate(cr *v2.Organization, org *clients.Organization) bool {
	if cr.Spec.ForProvider.Username != org.Username {
		return false
	}
	if cr.Spec.ForProvider.Name != nil && *cr.Spec.ForProvider.Name != org.FullName {
		return false
	}
	if cr.Spec.ForProvider.Description != nil && *cr.Spec.ForProvider.Description != org.Description {
		return false
	}
	if cr.Spec.ForProvider.Location != nil && *cr.Spec.ForProvider.Location != org.Location {
		return false
	}
	if cr.Spec.ForProvider.Website != nil && *cr.Spec.ForProvider.Website != org.Website {
		return false
	}
	if cr.Spec.ForProvider.Visibility != nil && *cr.Spec.ForProvider.Visibility != org.Visibility {
		return false
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "organization.create",
		tracing.SpanAttrs("organization", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganization)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		externalName = cr.Spec.ForProvider.Username
		meta.SetExternalName(cr, externalName)
	}

	var fullName, description, website, location, visibility string
	if cr.Spec.ForProvider.Name != nil {
		fullName = *cr.Spec.ForProvider.Name
	}
	if cr.Spec.ForProvider.Description != nil {
		description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Website != nil {
		website = *cr.Spec.ForProvider.Website
	}
	if cr.Spec.ForProvider.Location != nil {
		location = *cr.Spec.ForProvider.Location
	}
	if cr.Spec.ForProvider.Visibility != nil {
		visibility = *cr.Spec.ForProvider.Visibility
	}

	req := &clients.CreateOrganizationRequest{
		Username:    cr.Spec.ForProvider.Username,
		FullName:    fullName,
		Description: description,
		Website:     website,
		Location:    location,
		Visibility:  visibility,
	}

	org, err := e.client.CreateOrganization(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateOrganization)
	}

	meta.SetExternalName(cr, org.Username)

	repoAdminChangeTeamAccess := org.RepoAdminChangeTeamAccess
	publicRepos := org.NumRepos
	privateRepos := org.NumPrivateRepos
	members := org.NumMembers
	teams := org.NumTeams

	cr.Status.AtProvider = v2.OrganizationObservation{
		ID:                        &org.ID,
		AvatarURL:                 &org.AvatarURL,
		Email:                     &org.Email,
		RepoAdminChangeTeamAccess: &repoAdminChangeTeamAccess,
		PublicRepos:               &publicRepos,
		PrivateRepos:              &privateRepos,
		Members:                   &members,
		Teams:                     &teams,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "organization.update",
		tracing.SpanAttrs("organization", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganization)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.New(errInvalidExternalName)
	}

	req := &clients.UpdateOrganizationRequest{
		FullName:    cr.Spec.ForProvider.Name,
		Description: cr.Spec.ForProvider.Description,
		Website:     cr.Spec.ForProvider.Website,
		Location:    cr.Spec.ForProvider.Location,
		Visibility:  cr.Spec.ForProvider.Visibility,
	}

	org, err := e.client.UpdateOrganization(ctx, externalName, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganization)
	}

	repoAdminChangeTeamAccess := org.RepoAdminChangeTeamAccess
	publicRepos := org.NumRepos
	privateRepos := org.NumPrivateRepos
	members := org.NumMembers
	teams := org.NumTeams

	cr.Status.AtProvider = v2.OrganizationObservation{
		ID:                        &org.ID,
		AvatarURL:                 &org.AvatarURL,
		Email:                     &org.Email,
		RepoAdminChangeTeamAccess: &repoAdminChangeTeamAccess,
		PublicRepos:               &publicRepos,
		PrivateRepos:              &privateRepos,
		Members:                   &members,
		Teams:                     &teams,
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "organization.delete",
		tracing.SpanAttrs("organization", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganization)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-deleted external resource.
	_, err := e.client.GetOrganization(ctx, externalName)
	if err == nil {
		if err := e.client.DeleteOrganization(ctx, externalName); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteOrganization)
		}
		return managed.ExternalDelete{}, nil
	}
	if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteOrganization)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

// Setup adds a controller that reconciles Organization managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.OrganizationKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.OrganizationGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Organization{}).
		Complete(r)
}
