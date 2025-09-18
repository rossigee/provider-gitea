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

	"github.com/rossigee/provider-gitea/apis/repository/v1alpha1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRepository    = "managed resource is not a Repository custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errNewClient        = "cannot create new Service"
	errCreateRepository = "cannot create repository"
	errUpdateRepository = "cannot update repository"
	errDeleteRepository = "cannot delete repository"
	errGetRepository    = "cannot get repository"
)

// Setup adds a controller that reconciles Repository managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.RepositoryKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RepositoryGroupVersionKind),
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
		For(&v1alpha1.Repository{}).
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
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return nil, errors.New(errNotRepository)
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
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepository)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	owner := cr.Spec.ForProvider.Owner
	if owner == nil {
		return managed.ExternalObservation{}, errors.New("owner is required")
	}

	repository, err := c.client.GetRepository(ctx, *owner, externalName)
	if err != nil {
		// If repository doesn't exist, return that it needs to be created
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.RepositoryObservation{
		ID:        &repository.ID,
		FullName:  &repository.FullName,
		Fork:      &repository.Fork,
		Empty:     &repository.Empty,
		Size:      &repository.Size,
		HTMLURL:   &repository.HTMLURL,
		SSHURL:    &repository.SSHURL,
		CloneURL:  &repository.CloneURL,
		Language:  &repository.Language,
		CreatedAt: &repository.CreatedAt,
		UpdatedAt: &repository.UpdatedAt,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, repository),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}

	cr.SetConditions(xpv1.Creating())

	req := &giteaclients.CreateRepositoryRequest{
		Name:     cr.Spec.ForProvider.Name,
		AutoInit: cr.Spec.ForProvider.AutoInit != nil && *cr.Spec.ForProvider.AutoInit,
		Private:  cr.Spec.ForProvider.Private != nil && *cr.Spec.ForProvider.Private,
		Template: cr.Spec.ForProvider.Template != nil && *cr.Spec.ForProvider.Template,
	}

	if cr.Spec.ForProvider.Description != nil {
		req.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Gitignores != nil {
		req.Gitignores = *cr.Spec.ForProvider.Gitignores
	}
	if cr.Spec.ForProvider.License != nil {
		req.License = *cr.Spec.ForProvider.License
	}
	if cr.Spec.ForProvider.Readme != nil {
		req.Readme = *cr.Spec.ForProvider.Readme
	}
	if cr.Spec.ForProvider.IssueLabels != nil {
		req.IssueLabels = *cr.Spec.ForProvider.IssueLabels
	}
	if cr.Spec.ForProvider.TrustModel != nil {
		req.TrustModel = *cr.Spec.ForProvider.TrustModel
	}
	if cr.Spec.ForProvider.DefaultBranch != nil {
		req.DefaultBranch = *cr.Spec.ForProvider.DefaultBranch
	}

	var repository *giteaclients.Repository
	var err error

	if cr.Spec.ForProvider.Owner != nil {
		// Create repository in organization
		repository, err = c.client.CreateOrganizationRepository(ctx, *cr.Spec.ForProvider.Owner, req)
	} else {
		// Create repository for authenticated user
		repository, err = c.client.CreateRepository(ctx, req)
	}

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepository)
	}

	meta.SetExternalName(cr, repository.Name)

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepository)
	}

	owner := cr.Spec.ForProvider.Owner
	if owner == nil {
		return managed.ExternalUpdate{}, errors.New("owner is required")
	}

	externalName := meta.GetExternalName(cr)

	req := &giteaclients.UpdateRepositoryRequest{}

	if cr.Spec.ForProvider.Description != nil {
		req.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Website != nil {
		req.Website = cr.Spec.ForProvider.Website
	}
	if cr.Spec.ForProvider.Private != nil {
		req.Private = cr.Spec.ForProvider.Private
	}
	if cr.Spec.ForProvider.Template != nil {
		req.Template = cr.Spec.ForProvider.Template
	}
	if cr.Spec.ForProvider.HasIssues != nil {
		req.HasIssues = cr.Spec.ForProvider.HasIssues
	}
	if cr.Spec.ForProvider.HasWiki != nil {
		req.HasWiki = cr.Spec.ForProvider.HasWiki
	}
	if cr.Spec.ForProvider.HasPullRequests != nil {
		req.HasPullRequests = cr.Spec.ForProvider.HasPullRequests
	}
	if cr.Spec.ForProvider.HasProjects != nil {
		req.HasProjects = cr.Spec.ForProvider.HasProjects
	}
	if cr.Spec.ForProvider.HasReleases != nil {
		req.HasReleases = cr.Spec.ForProvider.HasReleases
	}
	if cr.Spec.ForProvider.HasPackages != nil {
		req.HasPackages = cr.Spec.ForProvider.HasPackages
	}
	if cr.Spec.ForProvider.HasActions != nil {
		req.HasActions = cr.Spec.ForProvider.HasActions
	}
	if cr.Spec.ForProvider.AllowMergeCommits != nil {
		req.AllowMergeCommits = cr.Spec.ForProvider.AllowMergeCommits
	}
	if cr.Spec.ForProvider.AllowRebase != nil {
		req.AllowRebase = cr.Spec.ForProvider.AllowRebase
	}
	if cr.Spec.ForProvider.AllowRebaseExplicit != nil {
		req.AllowRebaseExplicit = cr.Spec.ForProvider.AllowRebaseExplicit
	}
	if cr.Spec.ForProvider.AllowSquashMerge != nil {
		req.AllowSquashMerge = cr.Spec.ForProvider.AllowSquashMerge
	}
	if cr.Spec.ForProvider.AllowRebaseUpdate != nil {
		req.AllowRebaseUpdate = cr.Spec.ForProvider.AllowRebaseUpdate
	}
	if cr.Spec.ForProvider.DefaultDeleteBranchAfterMerge != nil {
		req.DefaultDeleteBranchAfterMerge = cr.Spec.ForProvider.DefaultDeleteBranchAfterMerge
	}
	if cr.Spec.ForProvider.DefaultMergeStyle != nil {
		req.DefaultMergeStyle = cr.Spec.ForProvider.DefaultMergeStyle
	}
	if cr.Spec.ForProvider.DefaultBranch != nil {
		req.DefaultBranch = cr.Spec.ForProvider.DefaultBranch
	}
	if cr.Spec.ForProvider.Archived != nil {
		req.Archived = cr.Spec.ForProvider.Archived
	}

	_, err := c.client.UpdateRepository(ctx, *owner, externalName, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepository)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepository)
	}

	cr.SetConditions(xpv1.Deleting())

	owner := cr.Spec.ForProvider.Owner
	if owner == nil {
		return managed.ExternalDelete{}, errors.New("owner is required")
	}

	externalName := meta.GetExternalName(cr)

	err := c.client.DeleteRepository(ctx, *owner, externalName)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepository)
	}

	return managed.ExternalDelete{}, nil
}

// isUpToDate checks if the repository is up to date with the desired state
func (c *external) isUpToDate(cr *v1alpha1.Repository, repository *giteaclients.Repository) bool {
	if cr.Spec.ForProvider.Description != nil && *cr.Spec.ForProvider.Description != repository.Description {
		return false
	}
	if cr.Spec.ForProvider.Website != nil && *cr.Spec.ForProvider.Website != repository.Website {
		return false
	}
	if cr.Spec.ForProvider.Private != nil && *cr.Spec.ForProvider.Private != repository.Private {
		return false
	}
	if cr.Spec.ForProvider.Template != nil && *cr.Spec.ForProvider.Template != repository.Template {
		return false
	}
	if cr.Spec.ForProvider.Archived != nil && *cr.Spec.ForProvider.Archived != repository.Archived {
		return false
	}

	return true
}
