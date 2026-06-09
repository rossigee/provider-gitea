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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/repositorycollaborator/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRepositoryCollaborator         = "managed resource is not a RepositoryCollaborator custom resource"
	errGetRepositoryCollaborator         = "failed to get repositorycollaborator"
	errCreateRepositoryCollaborator      = "failed to create repositorycollaborator"
	errUpdateRepositoryCollaborator      = "failed to update repositorycollaborator"
	errDeleteRepositoryCollaborator      = "failed to delete repositorycollaborator"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

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

	return &externalClient{client: conn}, nil
}

type externalClient struct {
	client clients.Client
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepositoryCollaborator)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	parts := strings.Split(externalID, ":")
	if len(parts) != 2 {
		return managed.ExternalObservation{}, errors.New("invalid external ID format")
	}

	repoParts := strings.Split(parts[0], "/")
	if len(repoParts) != 2 {
		return managed.ExternalObservation{}, errors.New("invalid repository format")
	}

	collab, err := e.client.GetRepositoryCollaborator(ctx, repoParts[0], repoParts[1], parts[1])
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepositoryCollaborator)
	}

	cr.Status.AtProvider = v2.RepositoryCollaboratorObservation{
		FullName:  &collab.FullName,
		Email:     &collab.Email,
		AvatarURL: &collab.AvatarURL,
		Permissions: &v2.RepositoryCollaboratorPermissions{
			Admin: &collab.Permissions.Admin,
		},
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepositoryCollaborator)
	}

	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalCreation{}, errors.New("invalid repository format")
	}

	createReq := &clients.AddCollaboratorRequest{
		Permission: cr.Spec.ForProvider.Permission,
	}

	err := e.client.AddRepositoryCollaborator(ctx, parts[0], parts[1], cr.Spec.ForProvider.Username, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepositoryCollaborator)
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.Repository+":"+cr.Spec.ForProvider.Username)
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepositoryCollaborator)
	}

	externalID := meta.GetExternalName(cr)
	parts := strings.Split(externalID, ":")
	if len(parts) != 2 {
		return managed.ExternalUpdate{}, errors.New("invalid external ID format")
	}

	repoParts := strings.Split(parts[0], "/")
	if len(repoParts) != 2 {
		return managed.ExternalUpdate{}, errors.New("invalid repository format")
	}

	updateReq := &clients.UpdateCollaboratorRequest{
		Permission: cr.Spec.ForProvider.Permission,
	}

	err := e.client.UpdateRepositoryCollaborator(ctx, repoParts[0], repoParts[1], parts[1], updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepositoryCollaborator)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.RepositoryCollaborator)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepositoryCollaborator)
	}

	externalID := meta.GetExternalName(cr)
	parts := strings.Split(externalID, ":")
	if len(parts) != 2 {
		return managed.ExternalDelete{}, errors.New("invalid external ID format")
	}

	repoParts := strings.Split(parts[0], "/")
	if len(repoParts) != 2 {
		return managed.ExternalDelete{}, errors.New("invalid repository format")
	}

	err := e.client.RemoveRepositoryCollaborator(ctx, repoParts[0], repoParts[1], parts[1])
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepositoryCollaborator)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.RepositoryCollaboratorKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RepositoryCollaboratorGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.RepositoryCollaborator{}).
		Complete(r)
}
