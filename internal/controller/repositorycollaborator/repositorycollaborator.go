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

package repositorycollaborator

import (
	"context"
	"strings"

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

	"github.com/rossigee/provider-gitea/apis/repositorycollaborator/v1alpha1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRepositoryCollaborator = "managed resource is not a RepositoryCollaborator custom resource"
	errTrackPCUsage              = "cannot track ProviderConfig usage"
	errGetPC                     = "cannot get ProviderConfig"
	errGetCreds                  = "cannot get credentials"
	errNewClient                 = "cannot create new Service"
	errAddCollaborator           = "cannot add collaborator"
	errUpdateCollaborator        = "cannot update collaborator"
	errRemoveCollaborator        = "cannot remove collaborator"
	errGetCollaborator           = "cannot get collaborator"
	errParseRepo                 = "cannot parse repository (expected owner/repo format)"
)

// Setup adds a controller that reconciles RepositoryCollaborator managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.RepositoryCollaboratorKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RepositoryCollaboratorGroupVersionKind),
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
		For(&v1alpha1.RepositoryCollaborator{}).
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
	cr, ok := mg.(*v1alpha1.RepositoryCollaborator)
	if !ok {
		return nil, errors.New(errNotRepositoryCollaborator)
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
	cr, ok := mg.(*v1alpha1.RepositoryCollaborator)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepositoryCollaborator)
	}

	// Parse repository owner/name
	repoParts := strings.SplitN(cr.Spec.ForProvider.Repository, "/", 2)
	if len(repoParts) != 2 {
		return managed.ExternalObservation{ResourceExists: false}, errors.New(errParseRepo)
	}
	owner, repo := repoParts[0], repoParts[1]

	// External name is the username
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		// Use the username from the spec as external name if not set
		meta.SetExternalName(cr, cr.Spec.ForProvider.Username)
		externalName = cr.Spec.ForProvider.Username
	}

	collaborator, err := c.client.GetRepositoryCollaborator(ctx, owner, repo, externalName)
	if err != nil {
		// If collaborator doesn't exist, return that it needs to be created
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.RepositoryCollaboratorObservation{
		FullName:  &collaborator.FullName,
		Email:     &collaborator.Email,
		AvatarURL: &collaborator.AvatarURL,
		Permissions: &v1alpha1.RepositoryCollaboratorPermissions{
			Admin: &collaborator.Permissions.Admin,
			Push:  &collaborator.Permissions.Push,
			Pull:  &collaborator.Permissions.Pull,
		},
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, collaborator),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.RepositoryCollaborator)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepositoryCollaborator)
	}

	cr.SetConditions(xpv1.Creating())

	// Parse repository owner/name
	repoParts := strings.SplitN(cr.Spec.ForProvider.Repository, "/", 2)
	if len(repoParts) != 2 {
		return managed.ExternalCreation{}, errors.New(errParseRepo)
	}
	owner, repo := repoParts[0], repoParts[1]

	req := &giteaclients.AddCollaboratorRequest{
		Permission: cr.Spec.ForProvider.Permission,
	}

	err := c.client.AddRepositoryCollaborator(ctx, owner, repo, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errAddCollaborator)
	}

	// Set external name to the username
	meta.SetExternalName(cr, cr.Spec.ForProvider.Username)

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.RepositoryCollaborator)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepositoryCollaborator)
	}

	// Parse repository owner/name
	repoParts := strings.SplitN(cr.Spec.ForProvider.Repository, "/", 2)
	if len(repoParts) != 2 {
		return managed.ExternalUpdate{}, errors.New(errParseRepo)
	}
	owner, repo := repoParts[0], repoParts[1]

	req := &giteaclients.UpdateCollaboratorRequest{
		Permission: cr.Spec.ForProvider.Permission,
	}

	err := c.client.UpdateRepositoryCollaborator(ctx, owner, repo, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateCollaborator)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.RepositoryCollaborator)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepositoryCollaborator)
	}

	cr.SetConditions(xpv1.Deleting())

	// Parse repository owner/name
	repoParts := strings.SplitN(cr.Spec.ForProvider.Repository, "/", 2)
	if len(repoParts) != 2 {
		return managed.ExternalDelete{}, errors.New(errParseRepo)
	}
	owner, repo := repoParts[0], repoParts[1]

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		externalName = cr.Spec.ForProvider.Username
	}

	err := c.client.RemoveRepositoryCollaborator(ctx, owner, repo, externalName)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errRemoveCollaborator)
	}

	return managed.ExternalDelete{}, nil
}

// isUpToDate checks if the collaborator permission is up to date with the desired state
func (c *external) isUpToDate(cr *v1alpha1.RepositoryCollaborator, collaborator *giteaclients.RepositoryCollaborator) bool {
	// Check permission level - we need to derive the permission from the collaborator's permissions
	desiredPermission := cr.Spec.ForProvider.Permission

	var currentPermission string
	if collaborator.Permissions.Admin {
		currentPermission = "admin"
	} else if collaborator.Permissions.Push {
		currentPermission = "write"
	} else {
		currentPermission = "read"
	}

	return desiredPermission == currentPermission
}
