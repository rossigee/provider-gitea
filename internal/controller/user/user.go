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

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/rossigee/provider-gitea/apis/user/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"
)

const (
	errNotUser            = "managed resource is not a User custom resource"
	errGetUser            = "failed to get user"
	errCreateUser         = "failed to create user"
	errUpdateUser         = "failed to update user"
	errDeleteUser         = "failed to delete user"
	errGetProviderConfig  = "failed to get provider config"
	errInvalidExternalName = "invalid external-name, expected username"
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
	_, span := tracing.StartSpan(ctx, "user.observe",
		tracing.SpanAttrs("user", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	user, err := e.client.GetUser(ctx, externalName)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetUser)
	}

	isAdmin := user.IsAdmin
	cr.Status.AtProvider = v2.UserObservation{
		ID:        &user.ID,
		AvatarURL: &user.AvatarURL,
		IsAdmin:   &isAdmin,
		LastLogin: &user.LastLogin,
		Created:   &user.Created,
		Language:  &user.Language,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUserUpToDate(cr, user),
	}, nil
}

func isUserUpToDate(cr *v2.User, user *clients.User) bool {
	if cr.Spec.ForProvider.Email != "" && cr.Spec.ForProvider.Email != user.Email {
		return false
	}
	if cr.Spec.ForProvider.FullName != nil && *cr.Spec.ForProvider.FullName != user.FullName {
		return false
	}
	if cr.Spec.ForProvider.Website != nil && *cr.Spec.ForProvider.Website != user.Website {
		return false
	}
	if cr.Spec.ForProvider.Location != nil && *cr.Spec.ForProvider.Location != user.Location {
		return false
	}
	if cr.Spec.ForProvider.Description != nil && *cr.Spec.ForProvider.Description != user.Description {
		return false
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "user.create",
		tracing.SpanAttrs("user", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		externalName = cr.Spec.ForProvider.Username
		meta.SetExternalName(cr, externalName)
	}

	req := &clients.CreateUserRequest{
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
	if cr.Spec.ForProvider.MustChangePassword != nil {
		req.MustChangePassword = *cr.Spec.ForProvider.MustChangePassword
	}
	if cr.Spec.ForProvider.Restricted != nil {
		req.Restricted = *cr.Spec.ForProvider.Restricted
	}
	if cr.Spec.ForProvider.Visibility != nil {
		req.Visibility = *cr.Spec.ForProvider.Visibility
	}
	if cr.Spec.ForProvider.SourceID != nil {
		req.SourceID = *cr.Spec.ForProvider.SourceID
	}

	user, err := e.client.CreateUser(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUser)
	}

	meta.SetExternalName(cr, user.Username)

	isAdmin := user.IsAdmin
	cr.Status.AtProvider = v2.UserObservation{
		ID:        &user.ID,
		AvatarURL: &user.AvatarURL,
		IsAdmin:   &isAdmin,
		Created:   &user.Created,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "user.update",
		tracing.SpanAttrs("user", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.New(errInvalidExternalName)
	}

	req := &clients.UpdateUserRequest{
		Email:        &cr.Spec.ForProvider.Email,
		FullName:     cr.Spec.ForProvider.FullName,
		LoginName:    cr.Spec.ForProvider.LoginName,
		Website:      cr.Spec.ForProvider.Website,
		Location:     cr.Spec.ForProvider.Location,
		Description:  cr.Spec.ForProvider.Description,
		Visibility:   cr.Spec.ForProvider.Visibility,
		Active:       cr.Spec.ForProvider.Active,
		Admin:        cr.Spec.ForProvider.Admin,
		ProhibitLogin: cr.Spec.ForProvider.ProhibitLogin,
		AllowGitHook: cr.Spec.ForProvider.AllowGitHook,
		AllowImportLocal: cr.Spec.ForProvider.AllowImportLocal,
		AllowCreateOrganization: cr.Spec.ForProvider.AllowCreateOrganization,
		Restricted:   cr.Spec.ForProvider.Restricted,
	}

	user, err := e.client.UpdateUser(ctx, externalName, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUser)
	}

	isAdmin := user.IsAdmin
	cr.Status.AtProvider = v2.UserObservation{
		ID:        &user.ID,
		AvatarURL: &user.AvatarURL,
		IsAdmin:   &isAdmin,
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "user.delete",
		tracing.SpanAttrs("user", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUser)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetUser(ctx, externalName); err == nil {
		if err := e.client.DeleteUser(ctx, externalName); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteUser)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteUser)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.UserKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.UserGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.User{}).
		Complete(r)
}
