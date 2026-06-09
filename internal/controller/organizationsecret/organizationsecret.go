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

package organizationsecret

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

	"github.com/rossigee/provider-gitea/apis/organizationsecret/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotOrganizationSecret         = "managed resource is not a OrganizationSecret custom resource"
	errGetOrganizationSecret         = "failed to get organizationsecret"
	errCreateOrganizationSecret      = "failed to create organizationsecret"
	errUpdateOrganizationSecret      = "failed to update organizationsecret"
	errDeleteOrganizationSecret      = "failed to delete organizationsecret"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.OrganizationSecret)
	if !ok {
		return nil, errors.New(errNotOrganizationSecret)
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
	cr, ok := mg.(*v2.OrganizationSecret)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationSecret)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	_, err := e.client.GetOrganizationSecret(ctx, cr.Spec.ForProvider.Organization, externalID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetOrganizationSecret)
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.OrganizationSecret)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationSecret)
	}

	createReq := &clients.CreateOrganizationSecretRequest{
		Data: "",
	}

	err := e.client.CreateOrganizationSecret(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.SecretName, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateOrganizationSecret)
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.SecretName)
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.OrganizationSecret)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationSecret)
	}

	updateReq := &clients.CreateOrganizationSecretRequest{}

	err := e.client.UpdateOrganizationSecret(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.SecretName, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganizationSecret)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.OrganizationSecret)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganizationSecret)
	}

	externalID := meta.GetExternalName(cr)
	err := e.client.DeleteOrganizationSecret(ctx, cr.Spec.ForProvider.Organization, externalID)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteOrganizationSecret)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.OrganizationSecretKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.OrganizationSecretGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.OrganizationSecret{}).
		Complete(r)
}
