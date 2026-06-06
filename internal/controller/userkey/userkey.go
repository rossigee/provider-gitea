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

	userkeyv2 "github.com/rossigee/provider-gitea/apis/userkey/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotUserKey          = "managed resource is not a UserKey"
	errGetProviderConfig   = "cannot get ProviderConfig"
	errProviderNotReady    = "provider is not ready"
	errGetUserKey          = "cannot get user key"
	errCreateUserKey       = "cannot create user key"
	errDeleteUserKey       = "cannot delete user key"
	errParseUserKeyID      = "cannot parse user key ID"
	errUpdateNotSupported  = "user keys are immutable; changes require deletion and recreation"
	controllerName         = "userkeys.userkey.gitea.m.crossplane.io"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(userkeyv2.UserKeyGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", "UserKey")),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(controllerName))),
		managed.WithPollInterval(o.PollInterval),
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&userkeyv2.UserKey{}).
		Complete(r)
}

type connector struct{ kube client.Client }
type external struct{ client giteaclients.Client }

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*userkeyv2.UserKey)
	if !ok {
		return nil, errors.New(errNotUserKey)
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
	cr, ok := mg.(*userkeyv2.UserKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUserKey)
	}
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	keyID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errParseUserKeyID)
	}
	key, err := e.client.GetUserKey(ctx, cr.Spec.ForProvider.Username, keyID)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetUserKey)
	}
	cr.Status.AtProvider = userkeyv2.UserKeyObservation{
		ID:          &key.ID,
		Title:       &key.Title,
		Fingerprint: &key.Fingerprint,
		URL:         &key.URL,
	}
	cr.Status.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: userKeyUpToDate(&cr.Spec.ForProvider, key),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*userkeyv2.UserKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUserKey)
	}
	req := &giteaclients.CreateUserKeyRequest{
		Title: cr.Spec.ForProvider.Title,
		Key:   cr.Spec.ForProvider.Key,
	}
	key, err := e.client.CreateUserKey(ctx, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUserKey)
	}
	meta.SetExternalName(cr, strconv.FormatInt(key.ID, 10))
	cr.Status.SetConditions(xpv1.Creating())
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(_ context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mg.(*userkeyv2.UserKey)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUserKey)
	}
	return managed.ExternalUpdate{}, errors.New(errUpdateNotSupported)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*userkeyv2.UserKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUserKey)
	}
	externalName := meta.GetExternalName(cr)
	keyID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errParseUserKeyID)
	}
	err = e.client.DeleteUserKey(ctx, cr.Spec.ForProvider.Username, keyID)
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteUserKey)
	}
	cr.Status.SetConditions(xpv1.Deleting())
	return managed.ExternalDelete{}, nil
}

func userKeyUpToDate(desired *userkeyv2.UserKeyParameters, actual *giteaclients.UserKey) bool {
	if desired.Title != actual.Title {
		return false
	}
	if desired.Key != actual.Key {
		return false
	}
	return true
}
