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

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	accesstokenv2 "github.com/rossigee/provider-gitea/apis/accesstoken/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotAccessToken      = "managed resource is not an AccessToken"
	errGetProviderConfig   = "cannot get ProviderConfig"
	errProviderNotReady    = "provider is not ready"
	errGetAccessToken      = "cannot get access token"
	errCreateAccessToken   = "cannot create access token"
	errUpdateAccessToken   = "cannot update access token"
	errDeleteAccessToken   = "cannot delete access token"
	errParseAccessTokenID  = "cannot parse access token ID"
	controllerName         = "accesstokens.accesstoken.gitea.m.crossplane.io"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(accesstokenv2.AccessTokenGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", "AccessToken")),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(controllerName))),
		managed.WithPollInterval(o.PollInterval),
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&accesstokenv2.AccessToken{}).
		Complete(r)
}

type connector struct{ kube client.Client }
type external struct{ client giteaclients.Client }

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*accesstokenv2.AccessToken)
	if !ok {
		return nil, errors.New(errNotAccessToken)
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
	cr, ok := mg.(*accesstokenv2.AccessToken)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAccessToken)
	}
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	tokenID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errParseAccessTokenID)
	}
	token, err := e.client.GetAccessToken(ctx, cr.Spec.ForProvider.Username, tokenID)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetAccessToken)
	}
	cr.Status.AtProvider = accesstokenv2.AccessTokenObservation{
		ID:            &token.ID,
		Name:          &token.Name,
		Scopes:        token.Scopes,
		TokenLastEight: &token.TokenLastEight,
	}
	cr.Status.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: accessTokenUpToDate(&cr.Spec.ForProvider, token),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*accesstokenv2.AccessToken)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAccessToken)
	}
	req := &giteaclients.CreateAccessTokenRequest{
		Name:   cr.Spec.ForProvider.Name,
		Scopes: cr.Spec.ForProvider.Scopes,
	}
	token, err := e.client.CreateAccessToken(ctx, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAccessToken)
	}
	meta.SetExternalName(cr, strconv.FormatInt(token.ID, 10))
	cr.Status.SetConditions(xpv1.Creating())
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*accesstokenv2.AccessToken)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAccessToken)
	}
	externalName := meta.GetExternalName(cr)
	tokenID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errParseAccessTokenID)
	}
	req := &giteaclients.UpdateAccessTokenRequest{
		Name:   &cr.Spec.ForProvider.Name,
		Scopes: cr.Spec.ForProvider.Scopes,
	}
	_, err = e.client.UpdateAccessToken(ctx, cr.Spec.ForProvider.Username, tokenID, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAccessToken)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*accesstokenv2.AccessToken)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAccessToken)
	}
	externalName := meta.GetExternalName(cr)
	tokenID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errParseAccessTokenID)
	}
	err = e.client.DeleteAccessToken(ctx, cr.Spec.ForProvider.Username, tokenID)
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAccessToken)
	}
	cr.Status.SetConditions(xpv1.Deleting())
	return managed.ExternalDelete{}, nil
}

func accessTokenUpToDate(desired *accesstokenv2.AccessTokenParameters, actual *giteaclients.AccessToken) bool {
	if desired.Name != actual.Name {
		return false
	}
	if len(desired.Scopes) != len(actual.Scopes) {
		return false
	}
	for i := range desired.Scopes {
		if desired.Scopes[i] != actual.Scopes[i] {
			return false
		}
	}
	return true
}
