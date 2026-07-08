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

package repositorykey

import (
	"context"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"github.com/rossigee/provider-gitea/apis/repositorykey/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	"sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

const (
	errNotRepositoryKey    = "managed resource is not a RepositoryKey custom resource"
	errGetRepositoryKey    = "failed to get repositorykey"
	errCreateRepositoryKey = "failed to create repositorykey"
	errUpdateRepositoryKey = "failed to update repositorykey"
	errDeleteRepositoryKey = "failed to delete repositorykey"
	errGetProviderConfig   = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.RepositoryKey)
	if !ok {
		return nil, errors.New(errNotRepositoryKey)
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
	_, span := tracing.StartSpan(ctx, "repositorykey.observe",
		tracing.SpanAttrs("repositorykey", mg.GetName(), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.RepositoryKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepositoryKey)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	keyID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to parse key ID")
	}

	key, err := e.client.GetRepositoryKey(ctx, cr.Spec.ForProvider.Repository, keyID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepositoryKey)
	}

	cr.Status.AtProvider = v2.RepositoryKeyObservation{
		ID: &key.ID,
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "repositorykey.create",
		tracing.SpanAttrs("repositorykey", mg.GetName(), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.RepositoryKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepositoryKey)
	}

	createReq := &clients.CreateRepositoryKeyRequest{
		Title: cr.Spec.ForProvider.Title,
		Key:   cr.Spec.ForProvider.Key,
	}

	key, err := e.client.CreateRepositoryKey(ctx, cr.Spec.ForProvider.Repository, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepositoryKey)
	}

	meta.SetExternalName(cr, strconv.FormatInt(key.ID, 10))
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "repositorykey.update",
		tracing.SpanAttrs("repositorykey", mg.GetName(), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.RepositoryKey)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepositoryKey)
	}

	externalID := meta.GetExternalName(cr)
	keyID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to parse key ID")
	}

	title := cr.Spec.ForProvider.Title
	updateReq := &clients.UpdateRepositoryKeyRequest{
		Title: &title,
	}

	_, err = e.client.UpdateRepositoryKey(ctx, cr.Spec.ForProvider.Repository, keyID, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepositoryKey)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "repositorykey.delete",
		tracing.SpanAttrs("repositorykey", mg.GetName(), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.RepositoryKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepositoryKey)
	}

	externalID := meta.GetExternalName(cr)
	keyID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to parse key ID")
	}

	err = e.client.DeleteRepositoryKey(ctx, cr.Spec.ForProvider.Repository, keyID)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepositoryKey)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.RepositoryKeyKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RepositoryKeyGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.RepositoryKey{}).
		Complete(r)
}
