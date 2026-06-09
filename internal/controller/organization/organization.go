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

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/organization/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotOrganization         = "managed resource is not a Organization custom resource"
	errGetOrganization         = "failed to get organization"
	errCreateOrganization      = "failed to create organization"
	errUpdateOrganization      = "failed to update organization"
	errDeleteOrganization      = "failed to delete organization"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

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

type externalClient struct {
	client clients.Client
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganization)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	org, err := e.client.GetOrganization(ctx, externalID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetOrganization)
	}

	cr.Status.AtProvider = v2.OrganizationObservation{
		ID:        &org.ID,
		AvatarURL: &org.AvatarURL,
		Email:     &org.Email,
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganization)
	}

	createReq := &clients.CreateOrganizationRequest{
		Username: cr.Spec.ForProvider.Username,
	}

	if cr.Spec.ForProvider.Name != nil {
		createReq.Name = *cr.Spec.ForProvider.Name
	}
	if cr.Spec.ForProvider.FullName != nil {
		createReq.FullName = *cr.Spec.ForProvider.FullName
	}
	if cr.Spec.ForProvider.Description != nil {
		createReq.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Website != nil {
		createReq.Website = *cr.Spec.ForProvider.Website
	}
	if cr.Spec.ForProvider.Location != nil {
		createReq.Location = *cr.Spec.ForProvider.Location
	}
	if cr.Spec.ForProvider.Visibility != nil {
		createReq.Visibility = *cr.Spec.ForProvider.Visibility
	}

	org, err := e.client.CreateOrganization(ctx, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateOrganization)
	}

	meta.SetExternalName(cr, org.Username)
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganization)
	}

	externalID := meta.GetExternalName(cr)

	updateReq := &clients.UpdateOrganizationRequest{}

	if cr.Spec.ForProvider.Name != nil {
		updateReq.Name = cr.Spec.ForProvider.Name
	}
	if cr.Spec.ForProvider.FullName != nil {
		updateReq.FullName = cr.Spec.ForProvider.FullName
	}
	if cr.Spec.ForProvider.Description != nil {
		updateReq.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Website != nil {
		updateReq.Website = cr.Spec.ForProvider.Website
	}
	if cr.Spec.ForProvider.Location != nil {
		updateReq.Location = cr.Spec.ForProvider.Location
	}
	if cr.Spec.ForProvider.Visibility != nil {
		updateReq.Visibility = cr.Spec.ForProvider.Visibility
	}

	_, err := e.client.UpdateOrganization(ctx, externalID, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganization)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Organization)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganization)
	}

	externalID := meta.GetExternalName(cr)

	err := e.client.DeleteOrganization(ctx, externalID)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteOrganization)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.OrganizationKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.OrganizationGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Organization{}).
		Complete(r)
}
