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

package accesstoken

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

	"github.com/rossigee/provider-gitea/apis/accesstoken/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotAccessToken         = "managed resource is not a AccessToken custom resource"
	errGetAccessToken         = "failed to get accesstoken"
	errCreateAccessToken      = "failed to create accesstoken"
	errUpdateAccessToken      = "failed to update accesstoken"
	errDeleteAccessToken      = "failed to delete accesstoken"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return nil, errors.New(errNotAccessToken)
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
	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAccessToken)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	tokenID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to parse token ID")
	}

	token, err := e.client.GetAccessToken(ctx, cr.Spec.ForProvider.Username, tokenID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetAccessToken)
	}

	cr.Status.AtProvider = v2.AccessTokenObservation{
		ID:       &token.ID,
		Name:     &token.Name,
		Scopes:   token.Scopes,
		Username: &cr.Spec.ForProvider.Username,
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAccessToken)
	}

	createReq := &clients.CreateAccessTokenRequest{
		Name:   cr.Spec.ForProvider.Name,
		Scopes: cr.Spec.ForProvider.Scopes,
	}

	token, err := e.client.CreateAccessToken(ctx, cr.Spec.ForProvider.Username, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAccessToken)
	}

	meta.SetExternalName(cr, strconv.FormatInt(token.ID, 10))
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mg.(*v2.AccessToken)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAccessToken)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAccessToken)
	}

	externalID := meta.GetExternalName(cr)
	tokenID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to parse token ID")
	}

	err = e.client.DeleteAccessToken(ctx, cr.Spec.ForProvider.Username, tokenID)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAccessToken)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.AccessTokenKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.AccessTokenGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.AccessToken{}).
		Complete(r)
}
