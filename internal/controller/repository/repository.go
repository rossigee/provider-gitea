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
	"fmt"
	"strings"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/repository/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
)

const (
	errNotRepository         = "managed resource is not a Repository custom resource"
	errGetRepository         = "failed to get repository"
	errCreateRepository      = "failed to create repository"
	errUpdateRepository      = "failed to update repository"
	errDeleteRepository      = "failed to delete repository"
	errGetProviderConfig     = "failed to get provider config"
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
	cr, ok := mg.(*v2.Repository)
	if !ok {
		return nil, errors.New(errNotRepository)
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

	return &externalClient{client: conn}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it matches the managed resource's desired state.
type externalClient struct {
	client clients.Client
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "repository.observe",
		tracing.SpanAttrs("repository", mg.GetName(), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepository)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	parts := strings.Split(externalID, "/")
	if len(parts) != 2 {
		return managed.ExternalObservation{}, errors.New("invalid external-id format, expected owner/name")
	}

	owner, name := parts[0], parts[1]

	repo, err := e.client.GetRepository(ctx, owner, name)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepository)
	}

	// Update observed state
	cr.Status.AtProvider = v2.RepositoryObservation{
		ID:       &repo.ID,
		FullName: &repo.FullName,
		HTMLURL:  &repo.HTMLURL,
		SSHURL:   &repo.SSHURL,
		CloneURL: &repo.CloneURL,
		Language: &repo.Language,
	}

	// Assume resource is up-to-date (simplified - could compare desired vs actual)
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "repository.create",
		tracing.SpanAttrs("repository", mg.GetName(), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}

	owner := ""
	if cr.Spec.ForProvider.Owner != nil {
		owner = *cr.Spec.ForProvider.Owner
	}

	// Build create request from spec
	createReq := &clients.CreateRepositoryRequest{
		Name: cr.Spec.ForProvider.Name,
	}

	if cr.Spec.ForProvider.Description != nil {
		createReq.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Private != nil {
		createReq.Private = *cr.Spec.ForProvider.Private
	}
	if cr.Spec.ForProvider.AutoInit != nil {
		createReq.AutoInit = *cr.Spec.ForProvider.AutoInit
	}
	if cr.Spec.ForProvider.Template != nil {
		createReq.Template = *cr.Spec.ForProvider.Template
	}
	if cr.Spec.ForProvider.DefaultBranch != nil {
		createReq.DefaultBranch = *cr.Spec.ForProvider.DefaultBranch
	}
	if cr.Spec.ForProvider.TrustModel != nil {
		createReq.TrustModel = *cr.Spec.ForProvider.TrustModel
	}

	var repo *clients.Repository
	var err error

	if owner != "" {
		repo, err = e.client.CreateOrganizationRepository(ctx, owner, createReq)
	} else {
		repo, err = e.client.CreateRepository(ctx, createReq)
	}

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepository)
	}

	// Set external name to owner/name format for future lookups
	externalID := fmt.Sprintf("%s/%s", repo.Owner.Username, repo.Name)
	meta.SetExternalName(cr, externalID)

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "repository.update",
		tracing.SpanAttrs("repository", mg.GetName(), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepository)
	}

	externalID := meta.GetExternalName(cr)
	parts := strings.Split(externalID, "/")
	if len(parts) != 2 {
		return managed.ExternalUpdate{}, errors.New("invalid external-id format")
	}

	owner, name := parts[0], parts[1]

	// Build update request from spec
	updateReq := &clients.UpdateRepositoryRequest{}

	if cr.Spec.ForProvider.Description != nil {
		updateReq.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Private != nil {
		updateReq.Private = cr.Spec.ForProvider.Private
	}
	if cr.Spec.ForProvider.Template != nil {
		updateReq.Template = cr.Spec.ForProvider.Template
	}
	if cr.Spec.ForProvider.DefaultBranch != nil {
		updateReq.DefaultBranch = cr.Spec.ForProvider.DefaultBranch
	}

	_, err := e.client.UpdateRepository(ctx, owner, name, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepository)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "repository.delete",
		tracing.SpanAttrs("repository", mg.GetName(), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepository)
	}

	externalID := meta.GetExternalName(cr)
	parts := strings.Split(externalID, "/")
	if len(parts) != 2 {
		return managed.ExternalDelete{}, errors.New("invalid external-id format")
	}

	owner, name := parts[0], parts[1]

	err := e.client.DeleteRepository(ctx, owner, name)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepository)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

// Setup adds a controller that reconciles Repository managed resources.
func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.RepositoryKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RepositoryGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Repository{}).
		Complete(r)
}
