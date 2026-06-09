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

package userkey

import (
	"context"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/userkey/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotUserKey         = "managed resource is not a UserKey custom resource"
	errGetUserKey         = "failed to get userkey"
	errCreateUserKey      = "failed to create userkey"
	errUpdateUserKey      = "failed to update userkey"
	errDeleteUserKey      = "failed to delete userkey"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.UserKey)
	if !ok {
		return nil, errors.New(errNotUserKey)
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
	cr, ok := mg.(*v2.UserKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUserKey)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	keyID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to parse key ID")
	}

	key, err := e.client.GetUserKey(ctx, cr.Spec.ForProvider.Username, keyID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetUserKey)
	}

	cr.Status.AtProvider = v2.UserKeyObservation{
		ID:    &key.ID,
		Title: &key.Title,
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.UserKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUserKey)
	}

	readOnly := false
	createReq := &clients.CreateUserKeyRequest{
		Title:   cr.Spec.ForProvider.Title,
		Key:     cr.Spec.ForProvider.Key,
		ReadOnly: &readOnly,
	}

	key, err := e.client.CreateUserKey(ctx, cr.Spec.ForProvider.Username, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUserKey)
	}

	meta.SetExternalName(cr, strconv.FormatInt(key.ID, 10))
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.UserKey)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUserKey)
	}

	externalID := meta.GetExternalName(cr)
	keyID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to parse key ID")
	}

	title := cr.Spec.ForProvider.Title
	updateReq := &clients.UpdateUserKeyRequest{
		Title: &title,
	}

	_, err = e.client.UpdateUserKey(ctx, cr.Spec.ForProvider.Username, keyID, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUserKey)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.UserKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUserKey)
	}

	externalID := meta.GetExternalName(cr)
	keyID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to parse key ID")
	}

	err = e.client.DeleteUserKey(ctx, cr.Spec.ForProvider.Username, keyID)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteUserKey)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.UserKeyKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.UserKeyGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.UserKey{}).
		Complete(r)
}
