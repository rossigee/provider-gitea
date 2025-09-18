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

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/user/v1alpha1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotUser      = "managed resource is not a User custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create new Service"
	errCreateUser   = "cannot create user"
	errUpdateUser   = "cannot update user"
	errDeleteUser   = "cannot delete user"
	errGetUser      = "cannot get user"
)

// Setup adds a controller that reconciles User managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.UserKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.UserGroupVersionKind),
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
		For(&v1alpha1.User{}).
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
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return nil, errors.New(errNotUser)
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

	return &external{client: client}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client giteaclients.Client
}

func (c *external) Disconnect(ctx context.Context) error {
	// No persistent connection to disconnect
	return nil
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	// Use the username as external identifier
	username := cr.Spec.ForProvider.Username
	if username == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	user, err := c.client.GetUser(ctx, username)
	if err != nil {
		// If user doesn't exist, that's not an error
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetUser)
	}

	// Update status with observed values
	cr.Status.AtProvider.ID = &user.ID
	cr.Status.AtProvider.AvatarURL = &user.AvatarURL
	cr.Status.AtProvider.IsAdmin = &user.IsAdmin
	if user.LastLogin != "" {
		cr.Status.AtProvider.LastLogin = &user.LastLogin
	}
	if user.Created != "" {
		cr.Status.AtProvider.Created = &user.Created
	}

	cr.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  c.isUpToDate(cr, user),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}

	// Convert API spec to client request - TODO: implement proper conversion
	req := &giteaclients.CreateUserRequest{
		Username: cr.Spec.ForProvider.Username,
		Email:    cr.Spec.ForProvider.Email,
		Password: cr.Spec.ForProvider.Password,
	}

	user, err := c.client.CreateUser(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUser)
	}

	// Set external name annotation to username
	meta.SetExternalName(cr, user.Username)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	// Convert API spec to client request - TODO: implement proper conversion
	req := &giteaclients.UpdateUserRequest{
		Email: &cr.Spec.ForProvider.Email,
	}

	_, err := c.client.UpdateUser(ctx, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUser)
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUser)
	}

	err := c.client.DeleteUser(ctx, cr.Spec.ForProvider.Username)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteUser)
	}

	return managed.ExternalDelete{}, nil
}

// isUpToDate checks if the observed user matches the desired state
func (c *external) isUpToDate(cr *v1alpha1.User, user *giteaclients.User) bool {
	// Check if email matches
	if cr.Spec.ForProvider.Email != "" && cr.Spec.ForProvider.Email != user.Email {
		return false
	}

	// Check if full name matches
	if cr.Spec.ForProvider.FullName != nil && *cr.Spec.ForProvider.FullName != user.FullName {
		return false
	}

	// Check if website matches
	if cr.Spec.ForProvider.Website != nil && *cr.Spec.ForProvider.Website != user.Website {
		return false
	}

	// Check if location matches
	if cr.Spec.ForProvider.Location != nil && *cr.Spec.ForProvider.Location != user.Location {
		return false
	}

	// Add more field comparisons as needed
	return true
}
