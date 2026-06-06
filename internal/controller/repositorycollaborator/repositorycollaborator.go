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

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rccv2 "github.com/rossigee/provider-gitea/apis/repositorycollaborator/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotCollaborator    = "managed resource is not a RepositoryCollaborator"
	errGetProviderConfig  = "cannot get ProviderConfig"
	errProviderNotReady   = "provider is not ready"
	errGetCollaborator    = "cannot get collaborator"
	errAddCollaborator    = "cannot add collaborator"
	errUpdateCollaborator = "cannot update collaborator"
	errRemoveCollaborator = "cannot remove collaborator"
	errParseRepository    = "cannot parse repository"
	controllerName        = "repositorycollaborators.repositorycollaborator.gitea.m.crossplane.io"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(rccv2.RepositoryCollaboratorGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", "RepositoryCollaborator")),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(controllerName))),
		managed.WithPollInterval(o.PollInterval),
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&rccv2.RepositoryCollaborator{}).
		Complete(r)
}

type connector struct{ kube client.Client }
type external struct{ client giteaclients.Client }

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*rccv2.RepositoryCollaborator)
	if !ok {
		return nil, errors.New(errNotCollaborator)
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
	cr, ok := mg.(*rccv2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCollaborator)
	}
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalObservation{}, errors.New(errParseRepository)
	}
	collaborator, err := e.client.GetRepositoryCollaborator(ctx, parts[0], parts[1], cr.Spec.ForProvider.Username)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetCollaborator)
	}
	cr.Status.AtProvider = rccv2.RepositoryCollaboratorObservation{
		FullName: &collaborator.FullName,
		Email:    &collaborator.Email,
	}
	cr.Status.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: collaboratorUpToDate(&cr.Spec.ForProvider, collaborator),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*rccv2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCollaborator)
	}
	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalCreation{}, errors.New(errParseRepository)
	}
	req := &giteaclients.AddCollaboratorRequest{
		Permission: cr.Spec.ForProvider.Permission,
	}
	if err := e.client.AddRepositoryCollaborator(ctx, parts[0], parts[1], cr.Spec.ForProvider.Username, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errAddCollaborator)
	}
	meta.SetExternalName(cr, cr.Spec.ForProvider.Username)
	cr.Status.SetConditions(xpv1.Creating())
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*rccv2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCollaborator)
	}
	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalUpdate{}, errors.New(errParseRepository)
	}
	req := &giteaclients.UpdateCollaboratorRequest{
		Permission: cr.Spec.ForProvider.Permission,
	}
	if err := e.client.UpdateRepositoryCollaborator(ctx, parts[0], parts[1], cr.Spec.ForProvider.Username, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateCollaborator)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*rccv2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotCollaborator)
	}
	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalDelete{}, errors.New(errParseRepository)
	}
	err := e.client.RemoveRepositoryCollaborator(ctx, parts[0], parts[1], cr.Spec.ForProvider.Username)
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errRemoveCollaborator)
	}
	cr.Status.SetConditions(xpv1.Deleting())
	return managed.ExternalDelete{}, nil
}

func collaboratorUpToDate(desired *rccv2.RepositoryCollaboratorParameters, actual *giteaclients.RepositoryCollaborator) bool {
	actualPerm := permissionsToString(&actual.Permissions)
	if desired.Permission != actualPerm {
		return false
	}
	return true
}

func permissionsToString(perms *giteaclients.RepositoryCollaboratorPermissions) string {
	if perms.Admin {
		return "admin"
	}
	if perms.Push {
		return "write"
	}
	return "read"
}
