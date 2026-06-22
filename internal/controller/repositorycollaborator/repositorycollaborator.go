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

// Package repositorycollaborator implements the Crossplane managed-resource
// reconciler for the Gitea RepositoryCollaborator resource. It follows the
// canonical pattern documented in repository.go and
// crossplane-provider-template dev/docs/09-lessons-learned.md.
package repositorycollaborator

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/repositorycollaborator/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRepositoryCollaborator = "managed resource is not a RepositoryCollaborator custom resource"
	errGetCollaborator           = "failed to get repository collaborator"
	errAddCollaborator           = "failed to add repository collaborator"
	errUpdateCollaborator        = "failed to update repository collaborator"
	errRemoveCollaborator        = "failed to remove repository collaborator"
	errGetProviderConfig         = "failed to get provider config"
	errExternalName              = "invalid external-name, expected the collaborator username"
	errRepositoryFormat          = "invalid spec.forProvider.repository, expected owner/name"
)

// Setup adds a controller that reconciles RepositoryCollaborator managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.RepositoryCollaboratorKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RepositoryCollaboratorGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.RepositoryCollaborator{}).
		// A non-nil rate limiter is mandatory (lesson #1).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.RepositoryCollaborator)
	if !ok {
		return nil, errors.New(errNotRepositoryCollaborator)
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

	return &external{client: conn}, nil
}

// external observes/creates/updates/deletes the backend collaborator.
type external struct {
	client clients.Client
}

// splitRepository parses the owner/name pair carried by spec.forProvider.repository.
func splitRepository(cr *v2.RepositoryCollaborator) (owner, repo string, ok bool) {
	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepositoryCollaborator)
	}

	// The collaborator's username is an immutable required spec field, so it is
	// the identity — resolve it from spec, not the external-name annotation
	// (which defaults to metadata.name before Create runs and would query the
	// wrong user). Create still pins external-name to it for the record.
	username := cr.Spec.ForProvider.Username
	if username == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	owner, repo, ok := splitRepository(cr)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errRepositoryFormat)
	}

	collab, err := e.client.GetRepositoryCollaborator(ctx, owner, repo, username)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3); real failures must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetCollaborator)
	}

	cr.Status.AtProvider = v2.RepositoryCollaboratorObservation{
		FullName:  &collab.FullName,
		Email:     &collab.Email,
		AvatarURL: &collab.AvatarURL,
		Permissions: &v2.RepositoryCollaboratorPermissions{
			Admin: &collab.Permissions.Admin,
			Push:  &collab.Permissions.Push,
			Pull:  &collab.Permissions.Pull,
		},
	}

	// Set Available on the exists path; drift is carried by ResourceUpToDate
	// (lesson #2/#6).
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: collaboratorUpToDate(cr, collab),
	}, nil
}

// collaboratorUpToDate reports whether the desired permission matches the
// observed permission flags. The backend exposes admin/push/pull bools rather
// than the requested permission string, so the spec permission is mapped to the
// flag it implies.
func collaboratorUpToDate(cr *v2.RepositoryCollaborator, observed *clients.RepositoryCollaborator) bool {
	switch cr.Spec.ForProvider.Permission {
	case "admin":
		return observed.Permissions.Admin
	case "write":
		return observed.Permissions.Push && !observed.Permissions.Admin
	case "read":
		return observed.Permissions.Pull && !observed.Permissions.Push && !observed.Permissions.Admin
	default:
		return true
	}
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepositoryCollaborator)
	}
	cr.SetConditions(xpv1.Creating())

	owner, repo, ok := splitRepository(cr)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errRepositoryFormat)
	}

	req := &clients.AddCollaboratorRequest{Permission: cr.Spec.ForProvider.Permission}
	if err := e.client.AddRepositoryCollaborator(ctx, owner, repo, cr.Spec.ForProvider.Username, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errAddCollaborator)
	}

	// Membership is keyed by username; pin it as the external-name so every
	// later Observe/Update/Delete resolves identity from the annotation.
	meta.SetExternalName(cr, cr.Spec.ForProvider.Username)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepositoryCollaborator)
	}

	// Identity is the immutable spec.Username (see Observe) — not the
	// external-name annotation, which defaults to metadata.name.
	username := cr.Spec.ForProvider.Username
	if username == "" {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	owner, repo, ok := splitRepository(cr)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errRepositoryFormat)
	}

	req := &clients.UpdateCollaboratorRequest{Permission: cr.Spec.ForProvider.Permission}
	if err := e.client.UpdateRepositoryCollaborator(ctx, owner, repo, username, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateCollaborator)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepositoryCollaborator)
	}
	cr.SetConditions(xpv1.Deleting())

	username := cr.Spec.ForProvider.Username
	if username == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	owner, repo, ok := splitRepository(cr)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errRepositoryFormat)
	}

	err := e.client.RemoveRepositoryCollaborator(ctx, owner, repo, username)
	// An already-absent collaborator is a successful delete (lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errRemoveCollaborator)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
