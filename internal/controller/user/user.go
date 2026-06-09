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
	"strings"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/user/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotUser         = "managed resource is not a User custom resource"
	errGetUser         = "failed to get user"
	errCreateUser      = "failed to create user"
	errUpdateUser      = "failed to update user"
	errDeleteUser      = "failed to delete user"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.User)
	if !ok {
		return nil, errors.New(errNotUser)
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
	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	user, err := e.client.GetUser(ctx, externalID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetUser)
	}

	cr.Status.AtProvider = v2.UserObservation{
		ID:        &user.ID,
		AvatarURL: &user.AvatarURL,
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}

	createReq := &clients.CreateUserRequest{
		Username: cr.Spec.ForProvider.Username,
		Email:    cr.Spec.ForProvider.Email,
		Password: cr.Spec.ForProvider.Password,
	}

	if cr.Spec.ForProvider.FullName != nil {
		createReq.FullName = *cr.Spec.ForProvider.FullName
	}
	if cr.Spec.ForProvider.LoginName != nil {
		createReq.LoginName = *cr.Spec.ForProvider.LoginName
	}
	if cr.Spec.ForProvider.SendNotify != nil {
		createReq.SendNotify = *cr.Spec.ForProvider.SendNotify
	}
	if cr.Spec.ForProvider.SourceID != nil {
		createReq.SourceID = *cr.Spec.ForProvider.SourceID
	}
	if cr.Spec.ForProvider.MustChangePassword != nil {
		createReq.MustChangePassword = *cr.Spec.ForProvider.MustChangePassword
	}
	if cr.Spec.ForProvider.Restricted != nil {
		createReq.Restricted = *cr.Spec.ForProvider.Restricted
	}
	if cr.Spec.ForProvider.Visibility != nil {
		createReq.Visibility = *cr.Spec.ForProvider.Visibility
	}

	user, err := e.client.CreateUser(ctx, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUser)
	}

	meta.SetExternalName(cr, user.Username)
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	externalID := meta.GetExternalName(cr)

	updateReq := &clients.UpdateUserRequest{}

	if cr.Spec.ForProvider.Email != "" {
		email := cr.Spec.ForProvider.Email
		updateReq.Email = &email
	}

	_, err := e.client.UpdateUser(ctx, externalID, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUser)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUser)
	}

	externalID := meta.GetExternalName(cr)

	err := e.client.DeleteUser(ctx, externalID)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteUser)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.UserKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.UserGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.User{}).
		Complete(r)
}
