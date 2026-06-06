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

package user

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

	userv2 "github.com/rossigee/provider-gitea/apis/user/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotUser            = "managed resource is not a User"
	errGetProviderConfig  = "cannot get ProviderConfig"
	errProviderNotReady   = "provider is not ready"
	errGetUser            = "cannot get user"
	errCreateUser         = "cannot create user"
	errUpdateUser         = "cannot update user"
	errDeleteUser         = "cannot delete user"
	controllerName        = "users.user.gitea.m.crossplane.io"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(userv2.UserGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", "User")),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(controllerName))),
		managed.WithPollInterval(o.PollInterval),
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&userv2.User{}).
		Complete(r)
}

type connector struct{ kube client.Client }
type external struct{ client giteaclients.Client }

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*userv2.User)
	if !ok {
		return nil, errors.New(errNotUser)
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
	cr, ok := mg.(*userv2.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}
	user, err := e.client.GetUser(ctx, cr.Spec.ForProvider.Username)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetUser)
	}
	cr.Status.AtProvider = userv2.UserObservation{
		ID:        &user.ID,
		AvatarURL: &user.AvatarURL,
		IsAdmin:   &user.IsAdmin,
		LastLogin: &user.LastLogin,
		Created:   &user.Created,
		Language:  &user.Language,
	}
	cr.Status.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: userUpToDate(&cr.Spec.ForProvider, user),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*userv2.User)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}
	req := &giteaclients.CreateUserRequest{
		Username: cr.Spec.ForProvider.Username,
		Email:    cr.Spec.ForProvider.Email,
		Password: cr.Spec.ForProvider.Password,
	}
	if cr.Spec.ForProvider.FullName != nil {
		req.FullName = *cr.Spec.ForProvider.FullName
	}
	if cr.Spec.ForProvider.LoginName != nil {
		req.LoginName = *cr.Spec.ForProvider.LoginName
	}
	if cr.Spec.ForProvider.SendNotify != nil {
		req.SendNotify = *cr.Spec.ForProvider.SendNotify
	}
	if cr.Spec.ForProvider.SourceID != nil {
		req.SourceID = *cr.Spec.ForProvider.SourceID
	}
	if cr.Spec.ForProvider.MustChangePassword != nil {
		req.MustChangePassword = *cr.Spec.ForProvider.MustChangePassword
	}
	if cr.Spec.ForProvider.Restricted != nil {
		req.Restricted = *cr.Spec.ForProvider.Restricted
	}
	if cr.Spec.ForProvider.Visibility != nil {
		req.Visibility = *cr.Spec.ForProvider.Visibility
	}
	_, err := e.client.CreateUser(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUser)
	}
	cr.Status.SetConditions(xpv1.Creating())
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*userv2.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}
	req := &giteaclients.UpdateUserRequest{}
	if cr.Spec.ForProvider.Email != "" {
		email := cr.Spec.ForProvider.Email
		req.Email = &email
	}
	if cr.Spec.ForProvider.FullName != nil {
		req.FullName = cr.Spec.ForProvider.FullName
	}
	if _, err := e.client.UpdateUser(ctx, cr.Spec.ForProvider.Username, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUser)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*userv2.User)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUser)
	}
	err := e.client.DeleteUser(ctx, cr.Spec.ForProvider.Username)
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteUser)
	}
	cr.Status.SetConditions(xpv1.Deleting())
	return managed.ExternalDelete{}, nil
}

func userUpToDate(desired *userv2.UserParameters, actual *giteaclients.User) bool {
	if desired.Email != "" && desired.Email != actual.Email {
		return false
	}
	if desired.FullName != nil && *desired.FullName != actual.FullName {
		return false
	}
	return true
}
