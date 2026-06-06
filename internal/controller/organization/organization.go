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

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	organizationv2 "github.com/rossigee/provider-gitea/apis/organization/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotOrganization        = "managed resource is not an Organization"
	errGetProviderConfig      = "cannot get ProviderConfig"
	errProviderNotReady       = "provider is not ready"
	errGetOrganization        = "cannot get organization"
	errCreateOrganization     = "cannot create organization"
	errUpdateOrganization     = "cannot update organization"
	errDeleteOrganization     = "cannot delete organization"
	controllerName            = "organizations.organization.gitea.m.crossplane.io"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(organizationv2.OrganizationGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", "Organization")),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(controllerName))),
		managed.WithPollInterval(o.PollInterval),
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&organizationv2.Organization{}).
		Complete(r)
}

type connector struct{ kube client.Client }
type external struct{ client giteaclients.Client }

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*organizationv2.Organization)
	if !ok {
		return nil, errors.New(errNotOrganization)
	}
	pcRef := cr.Spec.ProviderConfigReference
	if pcRef == nil {
		return nil, errors.New(errGetProviderConfig + ": providerConfigRef is required")
	}
	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: pcRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}
	if pc.Status.GetCondition(xpv1.TypeReady).Status != corev1.ConditionTrue {
		return nil, errors.New(errProviderNotReady)
	}
	cl, err := giteaclients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Gitea client")
	}
	return &external{client: cl}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*organizationv2.Organization)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganization)
	}
	org, err := e.client.GetOrganization(ctx, cr.Spec.ForProvider.Username)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetOrganization)
	}
	cr.Status.AtProvider = organizationv2.OrganizationObservation{
		ID:        &org.ID,
		AvatarURL: &org.AvatarURL,
		Email:     &org.Email,
	}
	cr.Status.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: organizationUpToDate(&cr.Spec.ForProvider, org),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*organizationv2.Organization)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganization)
	}
	req := &giteaclients.CreateOrganizationRequest{
		Username: cr.Spec.ForProvider.Username,
	}
	if cr.Spec.ForProvider.Name != nil {
		req.FullName = *cr.Spec.ForProvider.Name
	}
	if cr.Spec.ForProvider.Description != nil {
		req.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Website != nil {
		req.Website = *cr.Spec.ForProvider.Website
	}
	if cr.Spec.ForProvider.Location != nil {
		req.Location = *cr.Spec.ForProvider.Location
	}
	if cr.Spec.ForProvider.Visibility != nil {
		req.Visibility = *cr.Spec.ForProvider.Visibility
	}
	_, err := e.client.CreateOrganization(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateOrganization)
	}
	cr.Status.SetConditions(xpv1.Creating())
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*organizationv2.Organization)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganization)
	}
	req := &giteaclients.UpdateOrganizationRequest{}
	if cr.Spec.ForProvider.Name != nil {
		req.FullName = cr.Spec.ForProvider.Name
	}
	if cr.Spec.ForProvider.Description != nil {
		req.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Website != nil {
		req.Website = cr.Spec.ForProvider.Website
	}
	if cr.Spec.ForProvider.Location != nil {
		req.Location = cr.Spec.ForProvider.Location
	}
	if cr.Spec.ForProvider.Visibility != nil {
		req.Visibility = cr.Spec.ForProvider.Visibility
	}
	if _, err := e.client.UpdateOrganization(ctx, cr.Spec.ForProvider.Username, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganization)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*organizationv2.Organization)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganization)
	}
	err := e.client.DeleteOrganization(ctx, cr.Spec.ForProvider.Username)
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteOrganization)
	}
	cr.Status.SetConditions(xpv1.Deleting())
	return managed.ExternalDelete{}, nil
}

func organizationUpToDate(desired *organizationv2.OrganizationParameters, actual *giteaclients.Organization) bool {
	if desired.Name != nil && *desired.Name != actual.FullName {
		return false
	}
	if desired.Description != nil && *desired.Description != actual.Description {
		return false
	}
	if desired.Website != nil && *desired.Website != actual.Website {
		return false
	}
	if desired.Location != nil && *desired.Location != actual.Location {
		return false
	}
	if desired.Visibility != nil && *desired.Visibility != actual.Visibility {
		return false
	}
	return true
}
