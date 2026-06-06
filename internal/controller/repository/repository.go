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

package repository

import (
	"context"

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

	repositoryv2 "github.com/rossigee/provider-gitea/apis/repository/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRepository      = "managed resource is not a Repository"
	errGetProviderConfig  = "cannot get ProviderConfig"
	errProviderNotReady   = "provider is not ready"
	errGetRepository      = "cannot get repository"
	errCreateRepository   = "cannot create repository"
	errUpdateRepository   = "cannot update repository"
	errDeleteRepository   = "cannot delete repository"
	controllerName        = "repositories.repository.gitea.m.crossplane.io"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(repositoryv2.RepositoryGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", "Repository")),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(controllerName))),
		managed.WithPollInterval(o.PollInterval),
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&repositoryv2.Repository{}).
		Complete(r)
}

type connector struct{ kube client.Client }
type external struct{ client giteaclients.Client }

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*repositoryv2.Repository)
	if !ok {
		return nil, errors.New(errNotRepository)
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
	cr, ok := mg.(*repositoryv2.Repository)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepository)
	}
	owner := cr.Spec.ForProvider.Owner
	if owner == nil {
		name := cr.GetName()
		owner = &name
	}
	repo, err := e.client.GetRepository(ctx, *owner, cr.Spec.ForProvider.Name)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepository)
	}
	cr.Status.AtProvider = repositoryv2.RepositoryObservation{
		ID:       &repo.ID,
		FullName: &repo.FullName,
		HTMLURL:  &repo.HTMLURL,
		SSHURL:   &repo.SSHURL,
		CloneURL: &repo.CloneURL,
	}
	cr.Status.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: repositoryUpToDate(&cr.Spec.ForProvider, repo),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*repositoryv2.Repository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}
	req := &giteaclients.CreateRepositoryRequest{
		Name: cr.Spec.ForProvider.Name,
	}
	if cr.Spec.ForProvider.Description != nil {
		req.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Private != nil {
		req.Private = *cr.Spec.ForProvider.Private
	}
	if cr.Spec.ForProvider.AutoInit != nil {
		req.AutoInit = *cr.Spec.ForProvider.AutoInit
	}
	if cr.Spec.ForProvider.Template != nil {
		req.Template = *cr.Spec.ForProvider.Template
	}
	if cr.Spec.ForProvider.DefaultBranch != nil {
		req.DefaultBranch = *cr.Spec.ForProvider.DefaultBranch
	}
	if cr.Spec.ForProvider.TrustModel != nil {
		req.TrustModel = *cr.Spec.ForProvider.TrustModel
	}
	var repo *giteaclients.Repository
	var err error
	if cr.Spec.ForProvider.Owner != nil {
		repo, err = e.client.CreateOrganizationRepository(ctx, *cr.Spec.ForProvider.Owner, req)
	} else {
		repo, err = e.client.CreateRepository(ctx, req)
	}
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepository)
	}
	meta.SetExternalName(cr, repo.FullName)
	cr.Status.SetConditions(xpv1.Creating())
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*repositoryv2.Repository)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepository)
	}
	owner := cr.Spec.ForProvider.Owner
	if owner == nil {
		name := cr.GetName()
		owner = &name
	}
	req := &giteaclients.UpdateRepositoryRequest{}
	if cr.Spec.ForProvider.Description != nil {
		req.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Private != nil {
		req.Private = cr.Spec.ForProvider.Private
	}
	if cr.Spec.ForProvider.DefaultBranch != nil {
		req.DefaultBranch = cr.Spec.ForProvider.DefaultBranch
	}
	if cr.Spec.ForProvider.Archived != nil {
		req.Archived = cr.Spec.ForProvider.Archived
	}
	if cr.Spec.ForProvider.Template != nil {
		req.Template = cr.Spec.ForProvider.Template
	}
	if _, err := e.client.UpdateRepository(ctx, *owner, cr.Spec.ForProvider.Name, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepository)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*repositoryv2.Repository)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepository)
	}
	owner := cr.Spec.ForProvider.Owner
	if owner == nil {
		name := cr.GetName()
		owner = &name
	}
	err := e.client.DeleteRepository(ctx, *owner, cr.Spec.ForProvider.Name)
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepository)
	}
	cr.Status.SetConditions(xpv1.Deleting())
	return managed.ExternalDelete{}, nil
}

func repositoryUpToDate(desired *repositoryv2.RepositoryParameters, actual *giteaclients.Repository) bool {
	if desired.Description != nil && *desired.Description != actual.Description {
		return false
	}
	if desired.Private != nil && *desired.Private != actual.Private {
		return false
	}
	if desired.Archived != nil && *desired.Archived != actual.Archived {
		return false
	}
	return true
}
