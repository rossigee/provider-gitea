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

package pullrequest

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/pullrequest/v1alpha1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotPullRequest = "managed resource is not a PullRequest custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
	errNewClient      = "cannot create new Service"
	errCreatePR       = "cannot create pull request"
	errUpdatePR       = "cannot update pull request"
	errDeletePR       = "cannot delete pull request"
	errGetPR          = "cannot get pull request"
)

// Setup adds a controller that reconciles PullRequest managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.PullRequestKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.PullRequestGroupVersionKind),
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
		For(&v1alpha1.PullRequest{}).
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
	cr, ok := mg.(*v1alpha1.PullRequest)
	if !ok {
		return nil, errors.New(errNotPullRequest)
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
	cr, ok := mg.(*v1alpha1.PullRequest)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPullRequest)
	}

	externalName := meta.GetExternalName(cr)
	
	// If no external name is set or it's not a valid PR number, the resource hasn't been created yet
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Parse the external name as the pull request number
	prNumber, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		// If external name is not a valid number, treat as not created yet
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Get the pull request from Gitea
	pr, err := c.client.GetPullRequest(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, prNumber)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			// PR doesn't exist, mark for recreation
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetPR)
	}

	// Update observed state
	cr.Status.AtProvider = generatePullRequestObservation(pr)

	// Check if resource needs update
	upToDate := isPullRequestUpToDate(cr, pr)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        upToDate,
		ResourceLateInitialized: false,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.PullRequest)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPullRequest)
	}

	// Create pull request in Gitea
	pr, err := c.client.CreatePullRequest(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, &giteaclients.CreatePullRequestOptions{
		Title:         cr.Spec.ForProvider.Title,
		Body:          cr.Spec.ForProvider.Body,
		Head:          cr.Spec.ForProvider.Head,
		Base:          cr.Spec.ForProvider.Base,
		Assignees:     cr.Spec.ForProvider.Assignees,
		Reviewers:     cr.Spec.ForProvider.Reviewers,
		TeamReviewers: cr.Spec.ForProvider.TeamReviewers,
		Labels:        cr.Spec.ForProvider.Labels,
		Milestone:     cr.Spec.ForProvider.Milestone,
		Draft:         cr.Spec.ForProvider.Draft,
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreatePR)
	}

	// Set external name annotation
	meta.SetExternalName(cr, fmt.Sprintf("%d", pr.Number))

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.PullRequest)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPullRequest)
	}

	prNumber, err := strconv.ParseInt(meta.GetExternalName(cr), 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdatePR)
	}

	// Update pull request in Gitea
	_, err = c.client.UpdatePullRequest(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, prNumber, &giteaclients.UpdatePullRequestOptions{
		Title:     &cr.Spec.ForProvider.Title,
		Body:      cr.Spec.ForProvider.Body,
		State:     cr.Spec.ForProvider.State,
		Base:      &cr.Spec.ForProvider.Base,
		Assignees: cr.Spec.ForProvider.Assignees,
		Labels:    cr.Spec.ForProvider.Labels,
		Milestone: cr.Spec.ForProvider.Milestone,
		Draft:     cr.Spec.ForProvider.Draft,
	})
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdatePR)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.PullRequest)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotPullRequest)
	}

	prNumber, err := strconv.ParseInt(meta.GetExternalName(cr), 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeletePR)
	}

	// Close the pull request instead of deleting (PRs can't be truly deleted)
	closeState := "closed"
	_, err = c.client.UpdatePullRequest(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, prNumber, &giteaclients.UpdatePullRequestOptions{
		State: &closeState,
	})
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeletePR)
	}

	return managed.ExternalDelete{}, nil
}

// generatePullRequestObservation creates PullRequestObservation from Gitea pull request
func generatePullRequestObservation(pr *giteaclients.PullRequest) v1alpha1.PullRequestObservation {
	obs := v1alpha1.PullRequestObservation{
		ID:             pr.ID,
		Number:         pr.Number,
		URL:            pr.HTMLURL,
		State:          pr.State,
		Mergeable:      pr.Mergeable,
		Merged:         pr.Merged,
		Comments:       pr.Comments,
		ReviewComments: pr.ReviewComments,
		Additions:      pr.Additions,
		Deletions:      pr.Deletions,
		ChangedFiles:   pr.ChangedFiles,
		Author:         pr.User.Username,
	}

	if pr.CreatedAt != nil {
		obs.CreatedAt = pr.CreatedAt
	}

	if pr.UpdatedAt != nil {
		obs.UpdatedAt = pr.UpdatedAt
	}

	if pr.ClosedAt != nil {
		obs.ClosedAt = pr.ClosedAt
	}

	if pr.MergedAt != nil {
		obs.MergedAt = pr.MergedAt
	}

	return obs
}

// isPullRequestUpToDate checks if the pull request is up to date with desired state
func isPullRequestUpToDate(cr *v1alpha1.PullRequest, pr *giteaclients.PullRequest) bool {
	// Check title
	if cr.Spec.ForProvider.Title != pr.Title {
		return false
	}

	// Check body
	if cr.Spec.ForProvider.Body != nil && *cr.Spec.ForProvider.Body != pr.Body {
		return false
	}

	// Check state
	if cr.Spec.ForProvider.State != nil && *cr.Spec.ForProvider.State != pr.State {
		return false
	}

	// Check base branch
	if cr.Spec.ForProvider.Base != pr.Base.Ref {
		return false
	}

	// TODO: Add more detailed comparison for assignees, labels, milestone

	return true
}