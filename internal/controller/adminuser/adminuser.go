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

// Package adminuser implements the Crossplane managed-resource reconciler for
// the Gitea AdminUser resource. It follows the canonical pattern of the
// repository controller (see internal/controller/repository/repository.go and
// crossplane-provider-template dev/docs/09-lessons-learned.md). The create
// password is sourced from the referenced Kubernetes Secret.
package adminuser

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/adminuser/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotAdminUser      = "managed resource is not an AdminUser custom resource"
	errGetAdminUser      = "failed to get admin user"
	errCreateAdminUser   = "failed to create admin user"
	errUpdateAdminUser   = "failed to update admin user"
	errDeleteAdminUser   = "failed to delete admin user"
	errGetProviderConfig = "failed to get provider config"
	errGetPassword       = "failed to resolve password secret"
	errExternalName      = "invalid external-name, expected the username"
)

// Setup adds a controller that reconciles AdminUser managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.AdminUserKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.AdminUserGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.AdminUser{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.AdminUser)
	if !ok {
		return nil, errors.New(errNotAdminUser)
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

	// The kube client is retained so Create can resolve the password Secret.
	return &external{client: conn, kube: c.kube}, nil
}

// external observes/creates/updates/deletes the backend admin user.
type external struct {
	client clients.Client
	kube   client.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.AdminUser)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAdminUser)
	}

	// Identity is the external-name annotation (the username), authoritative for
	// Observe/Update/Delete (lesson #14). Empty -> not created; no GET.
	username := meta.GetExternalName(cr)
	if username == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	user, err := e.client.GetAdminUser(ctx, username)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetAdminUser)
	}

	cr.Status.AtProvider = v2.AdminUserObservation{
		ID:              &user.ID,
		Username:        &user.Username,
		Email:           &user.Email,
		FullName:        &user.FullName,
		AvatarURL:       &user.AvatarURL,
		IsAdmin:         &user.IsAdmin,
		IsActive:        &user.IsActive,
		IsRestricted:    &user.IsRestricted,
		ProhibitLogin:   &user.ProhibitLogin,
		Visibility:      &user.Visibility,
		CreatedAt:       &user.CreatedAt,
		LastLogin:       &user.LastLogin,
		Language:        &user.Language,
		MaxRepoCreation: &user.MaxRepoCreation,
		Website:         &user.Website,
		Location:        &user.Location,
		Description:     &user.Description,
	}

	upToDate := adminUserUpToDate(cr, user)

	// crossplane-runtime v2 no longer auto-sets Available(); set it on the exists
	// path (lesson #2/#6). Drift is carried by ResourceUpToDate, not Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// adminUserUpToDate compares only the mutable fields this provider pushes on
// Update; a nil desired pointer means "do not manage", so it never drifts.
func adminUserUpToDate(cr *v2.AdminUser, observed *clients.AdminUser) bool {
	p := cr.Spec.ForProvider
	if p.Email != "" && p.Email != observed.Email {
		return false
	}
	if p.FullName != nil && *p.FullName != observed.FullName {
		return false
	}
	if p.IsAdmin != nil && *p.IsAdmin != observed.IsAdmin {
		return false
	}
	if p.IsActive != nil && *p.IsActive != observed.IsActive {
		return false
	}
	if p.IsRestricted != nil && *p.IsRestricted != observed.IsRestricted {
		return false
	}
	if p.ProhibitLogin != nil && *p.ProhibitLogin != observed.ProhibitLogin {
		return false
	}
	if p.Visibility != nil && *p.Visibility != observed.Visibility {
		return false
	}
	if p.MaxRepoCreation != nil && *p.MaxRepoCreation != observed.MaxRepoCreation {
		return false
	}
	if p.Website != nil && *p.Website != observed.Website {
		return false
	}
	if p.Location != nil && *p.Location != observed.Location {
		return false
	}
	if p.Description != nil && *p.Description != observed.Description {
		return false
	}
	return true
}

// resolvePassword reads the password from the referenced Kubernetes Secret.
func (e *external) resolvePassword(ctx context.Context, cr *v2.AdminUser) (string, error) {
	ref := cr.Spec.ForProvider.PasswordSecretRef
	var sec corev1.Secret
	if err := e.kube.Get(ctx, client.ObjectKey{Namespace: ref.Namespace, Name: ref.Name}, &sec); err != nil {
		return "", errors.Wrap(err, errGetPassword)
	}
	v, ok := sec.Data[ref.Key]
	if !ok {
		return "", errors.Errorf("%s: key %q not found in secret %s/%s", errGetPassword, ref.Key, ref.Namespace, ref.Name)
	}
	return string(v), nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.AdminUser)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAdminUser)
	}
	cr.SetConditions(xpv1.Creating())

	password, err := e.resolvePassword(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	createReq := &clients.CreateAdminUserRequest{
		Username: cr.Spec.ForProvider.Username,
		Email:    cr.Spec.ForProvider.Email,
		Password: password,
	}
	if cr.Spec.ForProvider.FullName != nil {
		createReq.FullName = *cr.Spec.ForProvider.FullName
	}
	if cr.Spec.ForProvider.IsAdmin != nil {
		createReq.IsAdmin = *cr.Spec.ForProvider.IsAdmin
	}
	if cr.Spec.ForProvider.MustChangePassword != nil {
		createReq.MustChangePassword = *cr.Spec.ForProvider.MustChangePassword
	}
	if cr.Spec.ForProvider.SendNotify != nil {
		createReq.SendNotify = *cr.Spec.ForProvider.SendNotify
	}
	if cr.Spec.ForProvider.Visibility != nil {
		createReq.Visibility = *cr.Spec.ForProvider.Visibility
	}
	if cr.Spec.ForProvider.IsActive != nil {
		createReq.IsActive = *cr.Spec.ForProvider.IsActive
	}
	if cr.Spec.ForProvider.IsRestricted != nil {
		createReq.IsRestricted = *cr.Spec.ForProvider.IsRestricted
	}
	if cr.Spec.ForProvider.MaxRepoCreation != nil {
		createReq.MaxRepoCreation = *cr.Spec.ForProvider.MaxRepoCreation
	}
	if cr.Spec.ForProvider.ProhibitLogin != nil {
		createReq.ProhibitLogin = *cr.Spec.ForProvider.ProhibitLogin
	}
	if cr.Spec.ForProvider.Website != nil {
		createReq.Website = *cr.Spec.ForProvider.Website
	}
	if cr.Spec.ForProvider.Location != nil {
		createReq.Location = *cr.Spec.ForProvider.Location
	}
	if cr.Spec.ForProvider.Description != nil {
		createReq.Description = *cr.Spec.ForProvider.Description
	}

	user, err := e.client.CreateAdminUser(ctx, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAdminUser)
	}

	// Pin external-name to the authoritative username from the backend response
	// (lesson #3/#7/#14), never a spec guess.
	meta.SetExternalName(cr, user.Username)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.AdminUser)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAdminUser)
	}

	username := meta.GetExternalName(cr)
	if username == "" {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	// Gitea's admin-user PATCH requires login_name + source_id together (422
	// otherwise). For a local user, login_name is the username and source_id 0.
	loginName := username
	var sourceID int64
	updateReq := &clients.UpdateAdminUserRequest{
		LoginName:       &loginName,
		SourceID:        &sourceID,
		FullName:        cr.Spec.ForProvider.FullName,
		IsAdmin:         cr.Spec.ForProvider.IsAdmin,
		Visibility:      cr.Spec.ForProvider.Visibility,
		IsActive:        cr.Spec.ForProvider.IsActive,
		IsRestricted:    cr.Spec.ForProvider.IsRestricted,
		MaxRepoCreation: cr.Spec.ForProvider.MaxRepoCreation,
		ProhibitLogin:   cr.Spec.ForProvider.ProhibitLogin,
		Website:         cr.Spec.ForProvider.Website,
		Location:        cr.Spec.ForProvider.Location,
		Description:     cr.Spec.ForProvider.Description,
	}
	if cr.Spec.ForProvider.Email != "" {
		updateReq.Email = &cr.Spec.ForProvider.Email
	}

	if _, err := e.client.UpdateAdminUser(ctx, username, updateReq); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAdminUser)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.AdminUser)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAdminUser)
	}
	cr.SetConditions(xpv1.Deleting())

	username := meta.GetExternalName(cr)
	if username == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteAdminUser(ctx, username)
	// Treat an already-absent user as a successful delete (lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAdminUser)
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}
