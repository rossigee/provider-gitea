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

package issue

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/issue/v1alpha1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotIssue    = "managed resource is not an Issue custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC       = "cannot get ProviderConfig"
	errGetCreds    = "cannot get credentials"
	errNewClient   = "cannot create new Service"
	errCreateIssue = "cannot create issue"
	errUpdateIssue = "cannot update issue"
	errDeleteIssue = "cannot delete issue"
	errGetIssue    = "cannot get issue"
)

// Setup adds a controller that reconciles Issue managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.IssueKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.IssueGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.Issue{}).
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
	cr, ok := mg.(*v1alpha1.Issue)
	if !ok {
		return nil, errors.New(errNotIssue)
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
	cr, ok := mg.(*v1alpha1.Issue)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIssue)
	}

	externalName := meta.GetExternalName(cr)
	
	// If no external name is set or it's not a valid issue number, the resource hasn't been created yet
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Parse the external name as the issue number
	issueNumber, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		// If external name is not a valid number, treat as not created yet
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Get the issue from Gitea
	issue, err := c.client.GetIssue(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, issueNumber)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			// Issue doesn't exist, mark for recreation
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetIssue)
	}

	// Update observed state
	cr.Status.AtProvider = generateIssueObservation(issue)

	// Check if resource needs update
	upToDate := isIssueUpToDate(cr, issue)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        upToDate,
		ResourceLateInitialized: false,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Issue)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIssue)
	}

	// Create issue in Gitea
	issue, err := c.client.CreateIssue(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, &giteaclients.CreateIssueOptions{
		Title:     cr.Spec.ForProvider.Title,
		Body:      cr.Spec.ForProvider.Body,
		Assignees: cr.Spec.ForProvider.Assignees,
		Labels:    cr.Spec.ForProvider.Labels,
		Milestone: cr.Spec.ForProvider.Milestone,
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateIssue)
	}

	// Set external name annotation
	meta.SetExternalName(cr, fmt.Sprintf("%d", issue.Number))

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Issue)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIssue)
	}

	issueNumber, err := strconv.ParseInt(meta.GetExternalName(cr), 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateIssue)
	}

	// Update issue in Gitea
	_, err = c.client.UpdateIssue(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, issueNumber, &giteaclients.UpdateIssueOptions{
		Title:     &cr.Spec.ForProvider.Title,
		Body:      cr.Spec.ForProvider.Body,
		State:     cr.Spec.ForProvider.State,
		Assignees: cr.Spec.ForProvider.Assignees,
		Labels:    cr.Spec.ForProvider.Labels,
		Milestone: cr.Spec.ForProvider.Milestone,
	})
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateIssue)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Issue)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotIssue)
	}

	issueNumber, err := strconv.ParseInt(meta.GetExternalName(cr), 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteIssue)
	}

	// Close the issue instead of deleting (issues can't be truly deleted)
	closeState := "closed"
	_, err = c.client.UpdateIssue(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, issueNumber, &giteaclients.UpdateIssueOptions{
		State: &closeState,
	})
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteIssue)
	}

	return managed.ExternalDelete{}, nil
}

// generateIssueObservation creates IssueObservation from Gitea issue
func generateIssueObservation(issue *giteaclients.Issue) v1alpha1.IssueObservation {
	obs := v1alpha1.IssueObservation{
		ID:       issue.ID,
		Number:   issue.Number,
		URL:      issue.HTMLURL,
		State:    issue.State,
		Comments: issue.Comments,
		Author:   issue.User.Username,
	}

	if issue.CreatedAt != nil {
		obs.CreatedAt = issue.CreatedAt
	}

	if issue.UpdatedAt != nil {
		obs.UpdatedAt = issue.UpdatedAt
	}

	if issue.ClosedAt != nil {
		obs.ClosedAt = issue.ClosedAt
	}

	return obs
}

// isIssueUpToDate checks if the issue is up to date with desired state
func isIssueUpToDate(cr *v1alpha1.Issue, issue *giteaclients.Issue) bool {
	// Check title
	if cr.Spec.ForProvider.Title != issue.Title {
		return false
	}

	// Check body
	if cr.Spec.ForProvider.Body != nil && *cr.Spec.ForProvider.Body != issue.Body {
		return false
	}

	// Check state
	if cr.Spec.ForProvider.State != nil && *cr.Spec.ForProvider.State != issue.State {
		return false
	}

	// TODO: Add more detailed comparison for assignees, labels, milestone

	return true
}