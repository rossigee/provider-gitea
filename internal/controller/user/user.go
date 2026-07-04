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

// Package user implements the Crossplane managed-resource reconciler for the
// Gitea User resource. It follows the canonical pattern of the repository
// controller (see internal/controller/repository/repository.go and
// crossplane-provider-template dev/docs/09-lessons-learned.md).
package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/user/v1beta1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotUser           = "managed resource is not a User custom resource"
	errGetUser           = "failed to get user"
	errCreateUser        = "failed to create user"
	errUpdateUser        = "failed to update user"
	errDeleteUser        = "failed to delete user"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected the username"
	errTrackUsage        = "cannot track ProviderConfig usage"
)

// Setup adds a controller that reconciles User managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.UserKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithPollJitterHook(o.PollInterval / 10),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}
	// Honour spec.managementPolicies (ObserveOnly, no-delete, pause, ...) when the
	// operator runs the provider with --enable-management-policies.
	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.UserGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.User{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube  client.Client
	usage resource.ModernTracker
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.User)
	if !ok {
		return nil, errors.New(errNotUser)
	}

	if err := c.usage.Track(ctx, cr); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
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

	return &external{client: conn, kube: c.kube}, nil
}

// external observes/creates/updates/deletes the backend user.
type external struct {
	client clients.Client
	kube   client.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	// Identity is the external-name annotation (the username), authoritative for
	// Observe/Update/Delete (lesson #14). Empty -> not created; no GET.
	username := meta.GetExternalName(cr)
	if username == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	user, err := e.client.GetUser(ctx, username)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetUser)
	}

	// Preserve the password hash persisted by a previous reconcile; rebuilding
	// AtProvider from the GET response would otherwise drop it.
	priorHash := cr.Status.AtProvider.PasswordHash

	cr.Status.AtProvider = v2.UserObservation{
		ID:           &user.ID,
		AvatarURL:    &user.AvatarURL,
		IsAdmin:      &user.IsAdmin,
		LastLogin:    &user.LastLogin,
		Created:      &user.Created,
		Language:     &user.Language,
		PasswordHash: priorHash,
	}

	// Password-rotation drift: hash the current Secret content and compare it to
	// the hash the provider last applied. An empty stored hash or a mismatch
	// means the password must be (re)pushed on Update.
	passwordDrift := false
	if cr.Spec.ForProvider.PasswordSecretRef != nil {
		password, err := clients.ResolveSecretValue(ctx, e.kube, cr.Spec.ForProvider.PasswordSecretRef)
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, errGetUser)
		}
		current := hashPassword(password)
		if priorHash == nil || *priorHash != current {
			passwordDrift = true
		}
	}

	upToDate := userUpToDate(cr, user) && !passwordDrift

	// crossplane-runtime v2 no longer auto-sets Available(); set it on the exists
	// path (lesson #2/#6). Drift is carried by ResourceUpToDate, not Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// userUpToDate compares only the mutable fields this provider pushes on Update;
// a nil desired pointer means "do not manage", so it never drifts.
func userUpToDate(cr *v2.User, observed *clients.User) bool {
	p := cr.Spec.ForProvider
	if p.Email != "" && p.Email != observed.Email {
		return false
	}
	if p.FullName != nil && *p.FullName != observed.FullName {
		return false
	}
	if p.LoginName != nil && *p.LoginName != observed.LoginName {
		return false
	}
	if p.Active != nil && *p.Active != observed.Active {
		return false
	}
	if p.Admin != nil && *p.Admin != observed.IsAdmin {
		return false
	}
	if p.ProhibitLogin != nil && *p.ProhibitLogin != observed.ProhibitLogin {
		return false
	}
	if p.Restricted != nil && *p.Restricted != observed.Restricted {
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
	// Visibility is not surfaced on the User response, so it is not compared
	// here — comparing against an unobservable field would drift forever.
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}
	cr.SetConditions(xpv1.Creating())

	// The password is required for creation and is only ever read from the
	// referenced Secret — never from the spec (secret-ref convention).
	if cr.Spec.ForProvider.PasswordSecretRef == nil {
		return managed.ExternalCreation{}, errors.New("passwordSecretRef is required to create a user")
	}
	password, err := clients.ResolveSecretValue(ctx, e.kube, cr.Spec.ForProvider.PasswordSecretRef)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUser)
	}

	createReq := &clients.CreateUserRequest{
		Username: cr.Spec.ForProvider.Username,
		Email:    cr.Spec.ForProvider.Email,
		Password: password,
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
	// Passed as pointers, not dereferenced: both are meaningful at false
	// (see CreateUserRequest's doc comment on why these two are *bool).
	createReq.MustChangePassword = cr.Spec.ForProvider.MustChangePassword
	createReq.Restricted = cr.Spec.ForProvider.Restricted
	if cr.Spec.ForProvider.Visibility != nil {
		createReq.Visibility = *cr.Spec.ForProvider.Visibility
	}

	user, err := e.client.CreateUser(ctx, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUser)
	}

	// Pin external-name to the authoritative username from the backend response
	// (lesson #3/#7/#14), never a spec guess.
	meta.SetExternalName(cr, user.Username)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	username := meta.GetExternalName(cr)
	if username == "" {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	updateReq := &clients.UpdateUserRequest{
		FullName:                cr.Spec.ForProvider.FullName,
		LoginName:               cr.Spec.ForProvider.LoginName,
		SourceID:                cr.Spec.ForProvider.SourceID,
		MustChangePassword:      cr.Spec.ForProvider.MustChangePassword,
		Active:                  cr.Spec.ForProvider.Active,
		Admin:                   cr.Spec.ForProvider.Admin,
		AllowGitHook:            cr.Spec.ForProvider.AllowGitHook,
		AllowImportLocal:        cr.Spec.ForProvider.AllowImportLocal,
		AllowCreateOrganization: cr.Spec.ForProvider.AllowCreateOrganization,
		ProhibitLogin:           cr.Spec.ForProvider.ProhibitLogin,
		Restricted:              cr.Spec.ForProvider.Restricted,
		Website:                 cr.Spec.ForProvider.Website,
		Location:                cr.Spec.ForProvider.Location,
		Description:             cr.Spec.ForProvider.Description,
		Visibility:              cr.Spec.ForProvider.Visibility,
		MaxRepoCreation:         cr.Spec.ForProvider.MaxRepoCreation,
	}
	if cr.Spec.ForProvider.Email != "" {
		updateReq.Email = &cr.Spec.ForProvider.Email
	}

	// Gitea's PATCH /admin/users/{username} always requires login_name and
	// source_id even when not changing them. Default to the username and 0
	// (local account) when the spec does not pin them.
	if updateReq.LoginName == nil || *updateReq.LoginName == "" {
		updateReq.LoginName = &username
	}
	if updateReq.SourceID == nil {
		zero := int64(0)
		updateReq.SourceID = &zero
	}

	// Rotate the password only when the referenced Secret's content differs from
	// the hash the provider last applied (recomputed here, not trusted from a
	// possibly-stale status). When it matches we leave password unset so Forgejo
	// is not re-PATCHed with the password on every drift-driven Update.
	var newHash string
	if cr.Spec.ForProvider.PasswordSecretRef != nil {
		password, err := clients.ResolveSecretValue(ctx, e.kube, cr.Spec.ForProvider.PasswordSecretRef)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUser)
		}
		current := hashPassword(password)
		stored := cr.Status.AtProvider.PasswordHash
		if stored == nil || *stored != current {
			updateReq.Password = password
			newHash = current
		}
	}

	if _, err := e.client.UpdateUser(ctx, username, updateReq); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUser)
	}

	// Persist the applied hash so the next Observe sees the rotation as settled.
	if newHash != "" {
		cr.Status.AtProvider.PasswordHash = &newHash
	}

	return managed.ExternalUpdate{}, nil
}

// hashPassword returns a stable sha256 hex digest of the password content. It is
// the canonical form persisted to status.atProvider.passwordHash and compared in
// Observe; the password itself is never stored.
func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.User)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUser)
	}
	cr.SetConditions(xpv1.Deleting())

	username := meta.GetExternalName(cr)
	if username == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteUser(ctx, username)
	// Treat an already-absent user as a successful delete (lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteUser)
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}
