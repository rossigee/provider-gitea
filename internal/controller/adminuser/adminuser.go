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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/adminuser/v1alpha1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotAdminUser    = "managed resource is not an AdminUser custom resource"
	errTrackPCUsage    = "cannot track ProviderConfig usage"
	errGetPC           = "cannot get ProviderConfig"
	errGetCreds        = "cannot get credentials"
	errNewClient       = "cannot create new Service"
	errCreateAdminUser = "cannot create admin user"
	errUpdateAdminUser = "cannot update admin user"
	errDeleteAdminUser = "cannot delete admin user"
	errGetAdminUser    = "cannot get admin user"
	errGetPassword     = "cannot get password from secret"
)

// Setup adds a controller that reconciles AdminUser managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.AdminUserKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.AdminUserGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.TrackerFn(func(ctx context.Context, mg resource.Managed) error { return nil }),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.AdminUser{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.AdminUser)
	if !ok {
		return nil, errors.New(errNotAdminUser)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	client, err := giteaclients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: client, kube: c.kube}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client giteaclients.Client
	kube   client.Client
}

func (c *external) Disconnect(ctx context.Context) error {
	// No persistent connection to disconnect
	return nil
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.AdminUser)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAdminUser)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	user, err := c.client.GetAdminUser(ctx, externalName)
	if err != nil {
		// If user doesn't exist, return that it needs to be created
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.AdminUserObservation{
		ID:              &user.ID,
		Username:        &user.Username,
		Email:           &user.Email,
		FullName:        &user.FullName,
		AvatarURL:       &user.AvatarURL,
		IsAdmin:         &user.IsAdmin,
		IsActive:        &user.IsActive,
		IsRestricted:    &user.IsRestricted,
		MaxRepoCreation: &user.MaxRepoCreation,
		CreatedAt:       &user.CreatedAt,
		LastLogin:       &user.LastLogin,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, user),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.AdminUser)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAdminUser)
	}

	cr.SetConditions(xpv1.Creating())

	// Get password from secret
	password, err := c.getPasswordFromSecret(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetPassword)
	}

	req := &giteaclients.CreateAdminUserRequest{
		Username: cr.Spec.ForProvider.Username,
		Email:    cr.Spec.ForProvider.Email,
		Password: password,
		IsAdmin:  cr.Spec.ForProvider.IsAdmin != nil && *cr.Spec.ForProvider.IsAdmin,
	}

	if cr.Spec.ForProvider.FullName != nil {
		req.FullName = *cr.Spec.ForProvider.FullName
	}

	if cr.Spec.ForProvider.IsActive != nil {
		req.IsActive = *cr.Spec.ForProvider.IsActive
	}
	if cr.Spec.ForProvider.IsRestricted != nil {
		req.IsRestricted = *cr.Spec.ForProvider.IsRestricted
	}
	if cr.Spec.ForProvider.ProhibitLogin != nil {
		req.ProhibitLogin = *cr.Spec.ForProvider.ProhibitLogin
	}
	if cr.Spec.ForProvider.MustChangePassword != nil {
		req.MustChangePassword = *cr.Spec.ForProvider.MustChangePassword
	}
	if cr.Spec.ForProvider.SendNotify != nil {
		req.SendNotify = *cr.Spec.ForProvider.SendNotify
	}
	if cr.Spec.ForProvider.MaxRepoCreation != nil {
		req.MaxRepoCreation = *cr.Spec.ForProvider.MaxRepoCreation
	}
	if cr.Spec.ForProvider.Visibility != nil {
		req.Visibility = *cr.Spec.ForProvider.Visibility
	}

	user, err := c.client.CreateAdminUser(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAdminUser)
	}

	// Set external name for the created user
	meta.SetExternalName(cr, user.Username)

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.AdminUser)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAdminUser)
	}

	username := meta.GetExternalName(cr)

	// Get password from secret if provided - password updates not supported in Update
	// Password can only be set during creation

	req := &giteaclients.UpdateAdminUserRequest{
		Email:    &cr.Spec.ForProvider.Email,
		FullName: cr.Spec.ForProvider.FullName,
		IsAdmin:  cr.Spec.ForProvider.IsAdmin,
	}

	if cr.Spec.ForProvider.IsActive != nil {
		req.IsActive = cr.Spec.ForProvider.IsActive
	}
	if cr.Spec.ForProvider.IsRestricted != nil {
		req.IsRestricted = cr.Spec.ForProvider.IsRestricted
	}
	if cr.Spec.ForProvider.ProhibitLogin != nil {
		req.ProhibitLogin = cr.Spec.ForProvider.ProhibitLogin
	}
	if cr.Spec.ForProvider.MaxRepoCreation != nil {
		req.MaxRepoCreation = cr.Spec.ForProvider.MaxRepoCreation
	}
	if cr.Spec.ForProvider.Visibility != nil {
		req.Visibility = cr.Spec.ForProvider.Visibility
	}

	_, err := c.client.UpdateAdminUser(ctx, username, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAdminUser)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.AdminUser)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAdminUser)
	}

	cr.SetConditions(xpv1.Deleting())

	username := meta.GetExternalName(cr)

	err := c.client.DeleteAdminUser(ctx, username)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAdminUser)
	}

	return managed.ExternalDelete{}, nil
}

// Helper functions

func (c *external) getPasswordFromSecret(ctx context.Context, cr *v1alpha1.AdminUser) (string, error) {
	if cr.Spec.ForProvider.PasswordSecretRef.Name == "" {
		return "", errors.New("password secret reference is required")
	}

	secret := &corev1.Secret{}
	err := c.kube.Get(ctx, types.NamespacedName{
		Name:      cr.Spec.ForProvider.PasswordSecretRef.Name,
		Namespace: cr.Spec.ForProvider.PasswordSecretRef.Namespace,
	}, secret)
	if err != nil {
		return "", err
	}

	password, ok := secret.Data[cr.Spec.ForProvider.PasswordSecretRef.Key]
	if !ok {
		return "", errors.New("password key not found in secret")
	}

	return string(password), nil
}

func (c *external) isUpToDate(cr *v1alpha1.AdminUser, user *giteaclients.AdminUser) bool {
	if cr.Spec.ForProvider.Email != user.Email {
		return false
	}

	if cr.Spec.ForProvider.FullName != nil && *cr.Spec.ForProvider.FullName != user.FullName {
		return false
	}

	if cr.Spec.ForProvider.IsAdmin != nil && *cr.Spec.ForProvider.IsAdmin != user.IsAdmin {
		return false
	}

	if cr.Spec.ForProvider.IsActive != nil && *cr.Spec.ForProvider.IsActive != user.IsActive {
		return false
	}

	if cr.Spec.ForProvider.IsRestricted != nil && *cr.Spec.ForProvider.IsRestricted != user.IsRestricted {
		return false
	}

	if cr.Spec.ForProvider.MaxRepoCreation != nil && *cr.Spec.ForProvider.MaxRepoCreation != user.MaxRepoCreation {
		return false
	}

	return true
}
