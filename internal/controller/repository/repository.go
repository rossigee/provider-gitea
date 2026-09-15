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
	"sort"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/repository/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotRepository          = "managed resource is not a Repository custom resource"
	errGetRepository          = "failed to get repository"
	errCreateRepository       = "failed to create repository"
	errUpdateRepository       = "failed to update repository"
	errDeleteRepository       = "failed to delete repository"
	errGetProviderConfig      = "failed to get provider config"
	errGetRepositoryTopics    = "failed to get repository topics"
	errUpdateRepositoryTopics = "failed to update repository topics"
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
		tracing.SpanAttrs("repository", tracing.ResourceName(mg), "observe")...)
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
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	owner, name := parts[0], parts[1]

	repo, err := e.client.GetRepository(ctx, owner, name)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepository)
	}

	topics, err := e.client.GetRepositoryTopics(ctx, owner, name)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepositoryTopics)
	}

	var topicsList []string
	if topics != nil {
		topicsList = topics.Topics
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

	uo := isRepositoryUpToDate(cr, repo, topicsList)

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func strSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isRepositoryUpToDate(cr *v2.Repository, repo *clients.Repository, topics []string) bool {
	if cr.Spec.ForProvider.Description != nil && *cr.Spec.ForProvider.Description != repo.Description {
		return false
	}
	if cr.Spec.ForProvider.Private != nil && *cr.Spec.ForProvider.Private != repo.Private {
		return false
	}
	if cr.Spec.ForProvider.Template != nil && *cr.Spec.ForProvider.Template != repo.Template {
		return false
	}
	if cr.Spec.ForProvider.Archived != nil && *cr.Spec.ForProvider.Archived != repo.Archived {
		return false
	}
	if cr.Spec.ForProvider.DefaultBranch != nil && repo.DefaultBranch != "" && *cr.Spec.ForProvider.DefaultBranch != repo.DefaultBranch {
		return false
	}
	if cr.Spec.ForProvider.Website != nil && *cr.Spec.ForProvider.Website != repo.Website {
		return false
	}

	// Feature toggles
	if cr.Spec.ForProvider.HasIssues != nil && *cr.Spec.ForProvider.HasIssues != repo.HasIssues {
		return false
	}
	if cr.Spec.ForProvider.HasWiki != nil && *cr.Spec.ForProvider.HasWiki != repo.HasWiki {
		return false
	}
	if cr.Spec.ForProvider.HasPullRequests != nil && *cr.Spec.ForProvider.HasPullRequests != repo.HasPullRequests {
		return false
	}
	if cr.Spec.ForProvider.HasProjects != nil && *cr.Spec.ForProvider.HasProjects != repo.HasProjects {
		return false
	}
	if cr.Spec.ForProvider.HasReleases != nil && *cr.Spec.ForProvider.HasReleases != repo.HasReleases {
		return false
	}
	if cr.Spec.ForProvider.HasPackages != nil && *cr.Spec.ForProvider.HasPackages != repo.HasPackages {
		return false
	}
	if cr.Spec.ForProvider.HasActions != nil && *cr.Spec.ForProvider.HasActions != repo.HasActions {
		return false
	}

	// Merge strategy configuration
	if cr.Spec.ForProvider.AllowMergeCommits != nil && *cr.Spec.ForProvider.AllowMergeCommits != repo.AllowMergeCommits {
		return false
	}
	if cr.Spec.ForProvider.AllowRebase != nil && *cr.Spec.ForProvider.AllowRebase != repo.AllowRebase {
		return false
	}
	if cr.Spec.ForProvider.AllowRebaseExplicit != nil && *cr.Spec.ForProvider.AllowRebaseExplicit != repo.AllowRebaseExplicit {
		return false
	}
	if cr.Spec.ForProvider.AllowSquashMerge != nil && *cr.Spec.ForProvider.AllowSquashMerge != repo.AllowSquashMerge {
		return false
	}
	if cr.Spec.ForProvider.AllowRebaseUpdate != nil && *cr.Spec.ForProvider.AllowRebaseUpdate != repo.AllowRebaseUpdate {
		return false
	}
	if cr.Spec.ForProvider.DefaultDeleteBranchAfterMerge != nil && *cr.Spec.ForProvider.DefaultDeleteBranchAfterMerge != repo.DefaultDeleteBranchAfterMerge {
		return false
	}
	if cr.Spec.ForProvider.DefaultMergeStyle != nil && *cr.Spec.ForProvider.DefaultMergeStyle != repo.DefaultMergeStyle {
		return false
	}

	if cr.Spec.ForProvider.Topics != nil {
		want := append([]string(nil), cr.Spec.ForProvider.Topics...)
		got := append([]string(nil), topics...)
		sort.Strings(want)
		sort.Strings(got)
		if !strSliceEqual(want, got) {
			return false
		}
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "repository.create",
		tracing.SpanAttrs("repository", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}

	owner := ""
	if cr.Spec.ForProvider.Owner != nil {
		owner = *cr.Spec.ForProvider.Owner
	}

	// Check if repo already exists - if so, don't treat this as a new creation
	// This prevents the "Creating" condition from persisting
	name := cr.Spec.ForProvider.Name
	if name == "" {
		name = meta.GetExternalName(cr)
		if name != "" {
			parts := strings.Split(name, "/")
			if len(parts) == 2 {
				name = parts[1]
			}
		}
	}

	if name != "" {
		// Check if repo already exists
		var checkOwner string
		var checkName string
		if owner != "" {
			checkOwner = owner
			checkName = name
		} else {
			extName := meta.GetExternalName(cr)
			if extName != "" {
				parts := strings.Split(extName, "/")
				if len(parts) == 2 {
					checkOwner = parts[0]
					checkName = parts[1]
				}
			}
		}
		if checkOwner != "" && checkName != "" {
			_, err := e.client.GetRepository(ctx, checkOwner, checkName)
			if err == nil {
				return managed.ExternalCreation{}, nil
			}
		}
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

	if len(cr.Spec.ForProvider.Topics) > 0 {
		if err := e.client.UpdateRepositoryTopics(ctx, repo.Owner.Username, repo.Name, &clients.UpdateRepositoryTopicsRequest{
			Topics: cr.Spec.ForProvider.Topics,
		}); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errUpdateRepositoryTopics)
		}
	}

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "repository.update",
		tracing.SpanAttrs("repository", tracing.ResourceName(mg), "update")...)
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
	if cr.Spec.ForProvider.Archived != nil {
		updateReq.Archived = cr.Spec.ForProvider.Archived
	}
	if cr.Spec.ForProvider.Website != nil {
		updateReq.Website = cr.Spec.ForProvider.Website
	}
	if cr.Spec.ForProvider.DefaultBranch != nil {
		updateReq.DefaultBranch = cr.Spec.ForProvider.DefaultBranch
	}

	// Feature toggles
	if cr.Spec.ForProvider.HasIssues != nil {
		updateReq.HasIssues = cr.Spec.ForProvider.HasIssues
	}
	if cr.Spec.ForProvider.HasWiki != nil {
		updateReq.HasWiki = cr.Spec.ForProvider.HasWiki
	}
	if cr.Spec.ForProvider.HasPullRequests != nil {
		updateReq.HasPullRequests = cr.Spec.ForProvider.HasPullRequests
	}
	if cr.Spec.ForProvider.HasProjects != nil {
		updateReq.HasProjects = cr.Spec.ForProvider.HasProjects
	}
	if cr.Spec.ForProvider.HasReleases != nil {
		updateReq.HasReleases = cr.Spec.ForProvider.HasReleases
	}
	if cr.Spec.ForProvider.HasPackages != nil {
		updateReq.HasPackages = cr.Spec.ForProvider.HasPackages
	}
	if cr.Spec.ForProvider.HasActions != nil {
		updateReq.HasActions = cr.Spec.ForProvider.HasActions
	}

	// Merge strategy configuration
	if cr.Spec.ForProvider.AllowMergeCommits != nil {
		updateReq.AllowMergeCommits = cr.Spec.ForProvider.AllowMergeCommits
	}
	if cr.Spec.ForProvider.AllowRebase != nil {
		updateReq.AllowRebase = cr.Spec.ForProvider.AllowRebase
	}
	if cr.Spec.ForProvider.AllowRebaseExplicit != nil {
		updateReq.AllowRebaseExplicit = cr.Spec.ForProvider.AllowRebaseExplicit
	}
	if cr.Spec.ForProvider.AllowSquashMerge != nil {
		updateReq.AllowSquashMerge = cr.Spec.ForProvider.AllowSquashMerge
	}
	if cr.Spec.ForProvider.AllowRebaseUpdate != nil {
		updateReq.AllowRebaseUpdate = cr.Spec.ForProvider.AllowRebaseUpdate
	}
	if cr.Spec.ForProvider.DefaultDeleteBranchAfterMerge != nil {
		updateReq.DefaultDeleteBranchAfterMerge = cr.Spec.ForProvider.DefaultDeleteBranchAfterMerge
	}
	if cr.Spec.ForProvider.DefaultMergeStyle != nil {
		updateReq.DefaultMergeStyle = cr.Spec.ForProvider.DefaultMergeStyle
	}

	_, err := e.client.UpdateRepository(ctx, owner, name, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepository)
	}

	if cr.Spec.ForProvider.Topics != nil {
		if err := e.client.UpdateRepositoryTopics(ctx, owner, name, &clients.UpdateRepositoryTopicsRequest{
			Topics: cr.Spec.ForProvider.Topics,
		}); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepositoryTopics)
		}
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "repository.delete",
		tracing.SpanAttrs("repository", tracing.ResourceName(mg), "delete")...)
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

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetRepository(ctx, owner, name); err == nil {
		if err := e.client.DeleteRepository(ctx, owner, name); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepository)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepository)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.RepositoryKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RepositoryGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Repository{}).
		Complete(r)
}
