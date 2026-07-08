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

package repositorysecret

import (
	"context"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"github.com/rossigee/provider-gitea/apis/repositorysecret/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	"sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	errNotRepositorySecret    = "managed resource is not a RepositorySecret custom resource"
	errGetRepositorySecret    = "failed to get repositorysecret"
	errCreateRepositorySecret = "failed to create repositorysecret"
	errUpdateRepositorySecret = "failed to update repositorysecret"
	errDeleteRepositorySecret = "failed to delete repositorysecret"
	errGetProviderConfig      = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.RepositorySecret)
	if !ok {
		return nil, errors.New(errNotRepositorySecret)
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
	_, span := tracing.StartSpan(ctx, "repositorysecret.observe",
		tracing.SpanAttrs("repositorysecret", mg.GetName(), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.RepositorySecret)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepositorySecret)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalObservation{}, errors.New("invalid repository format")
	}

	_, err := e.client.GetRepositorySecret(ctx, cr.Spec.ForProvider.Repository, externalID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepositorySecret)
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "repositorysecret.create",
		tracing.SpanAttrs("repositorysecret", mg.GetName(), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.RepositorySecret)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepositorySecret)
	}

	createReq := &clients.CreateRepositorySecretRequest{
		Data: "",
	}

	err := e.client.CreateRepositorySecret(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.SecretName, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepositorySecret)
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.SecretName)
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "repositorysecret.update",
		tracing.SpanAttrs("repositorysecret", mg.GetName(), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.RepositorySecret)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepositorySecret)
	}

	updateReq := &clients.UpdateRepositorySecretRequest{}

	err := e.client.UpdateRepositorySecret(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.SecretName, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepositorySecret)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "repositorysecret.delete",
		tracing.SpanAttrs("repositorysecret", mg.GetName(), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.RepositorySecret)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepositorySecret)
	}

	externalID := meta.GetExternalName(cr)
	err := e.client.DeleteRepositorySecret(ctx, cr.Spec.ForProvider.Repository, externalID)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepositorySecret)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.RepositorySecretKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RepositorySecretGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.RepositorySecret{}).
		Complete(r)
}
