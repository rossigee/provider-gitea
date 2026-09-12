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

package adminuser

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/adminuser/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"strings"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotAdminUser      = "managed resource is not an AdminUser custom resource"
	errGetAdminUser      = "failed to get admin user"
	errCreateAdminUser   = "failed to create admin user"
	errUpdateAdminUser   = "failed to update admin user"
	errDeleteAdminUser   = "failed to delete admin user"
	errGetProviderConfig = "failed to get provider config"
	errGetPassword       = "failed to read password from secret"
)

// A connector is expected to produce an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect returns an ExternalClient by:
// 1. Getting the provider config
// 2. Creating a Gitea API client
// 3. Returning an external client wrapping the API client
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.AdminUser)
	if !ok {
		return nil, errors.New(errNotAdminUser)
	}

	// Get provider config reference from spec
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

	return &externalClient{kube: c.kube, client: conn}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it matches the managed resource's desired state.
type externalClient struct {
	kube   client.Client
	client clients.Client
}

// readPassword returns the password from the referenced secret.
func (e *externalClient) readPassword(ctx context.Context, cr *v2.AdminUser) (string, error) {
	ref := cr.Spec.ForProvider.PasswordSecretRef
	ns := cr.GetNamespace()
	if ref.Namespace != "" {
		ns = ref.Namespace
	}
	var secret corev1.Secret
	if err := e.kube.Get(ctx, client.ObjectKey{Namespace: ns, Name: ref.Name}, &secret); err != nil {
		return "", errors.Wrap(err, errGetPassword)
	}
	pw, ok := secret.Data[ref.Key]
	if !ok {
		return "", errors.New(errGetPassword + ": key not found in secret")
	}
	return string(pw), nil
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "adminuser.observe",
		tracing.SpanAttrs("adminuser", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.AdminUser)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAdminUser)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		externalID = cr.Spec.ForProvider.Username
	}

	user, err := e.client.GetAdminUser(ctx, externalID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetAdminUser)
	}

	// Update observed state
	cr.Status.AtProvider = v2.AdminUserObservation{
		ID:        &user.ID,
		Username:  &user.Username,
		Email:     &user.Email,
		FullName:  &user.FullName,
		AvatarURL: &user.AvatarURL,
		IsAdmin:   &user.IsAdmin,
	}

	uo := isAdminUserUpToDate(cr, user)

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func isAdminUserUpToDate(cr *v2.AdminUser, user *clients.AdminUser) bool {
	fp := cr.Spec.ForProvider
	if fp.Email != user.Email {
		return false
	}
	if fp.FullName != nil && *fp.FullName != user.FullName {
		return false
	}
	if fp.IsAdmin != nil && *fp.IsAdmin != user.IsAdmin {
		return false
	}
	if fp.IsActive != nil && *fp.IsActive != user.IsActive {
		return false
	}
	if fp.Visibility != nil && *fp.Visibility != user.Visibility {
		return false
	}
	if fp.IsRestricted != nil && *fp.IsRestricted != user.IsRestricted {
		return false
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "adminuser.create",
		tracing.SpanAttrs("adminuser", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.AdminUser)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAdminUser)
	}

	password, err := e.readPassword(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	fp := cr.Spec.ForProvider
	req := &clients.CreateAdminUserRequest{
		Username: fp.Username,
		Email:    fp.Email,
		Password: password,
	}
	if fp.FullName != nil {
		req.FullName = *fp.FullName
	}
	if fp.IsAdmin != nil {
		req.IsAdmin = *fp.IsAdmin
	}
	if fp.MustChangePassword != nil {
		req.MustChangePassword = *fp.MustChangePassword
	}
	if fp.SendNotify != nil {
		req.SendNotify = *fp.SendNotify
	}
	if fp.Visibility != nil {
		req.Visibility = *fp.Visibility
	}
	if fp.IsActive != nil {
		req.IsActive = *fp.IsActive
	}
	if fp.IsRestricted != nil {
		req.IsRestricted = *fp.IsRestricted
	}

	user, err := e.client.CreateAdminUser(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAdminUser)
	}

	// Set external name to username for future lookups
	meta.SetExternalName(cr, user.Username)

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "adminuser.update",
		tracing.SpanAttrs("adminuser", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.AdminUser)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAdminUser)
	}

	username := meta.GetExternalName(cr)
	if username == "" {
		username = cr.Spec.ForProvider.Username
	}

	fp := cr.Spec.ForProvider
	req := &clients.UpdateAdminUserRequest{
		Email:        &fp.Email,
		FullName:     fp.FullName,
		IsAdmin:      fp.IsAdmin,
		Visibility:   fp.Visibility,
		IsActive:     fp.IsActive,
		IsRestricted: fp.IsRestricted,
	}

	if _, err := e.client.UpdateAdminUser(ctx, username, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAdminUser)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "adminuser.delete",
		tracing.SpanAttrs("adminuser", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.AdminUser)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAdminUser)
	}

	username := meta.GetExternalName(cr)
	if username == "" {
		username = cr.Spec.ForProvider.Username
	}

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetAdminUser(ctx, username); err == nil {
		if err := e.client.DeleteAdminUser(ctx, username); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAdminUser)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAdminUser)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.AdminUserKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.AdminUserGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.AdminUser{}).
		Complete(r)
}
