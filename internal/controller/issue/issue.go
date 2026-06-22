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

// Package issue implements the Crossplane managed-resource reconciler for the
// Gitea Issue resource. It follows the canonical pattern documented in
// internal/controller/repository/repository.go and the lessons in
// crossplane-provider-template dev/docs/09-lessons-learned.md.
package issue

import (
	"context"
	"strconv"

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

	v2 "github.com/rossigee/provider-gitea/apis/issue/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotIssue          = "managed resource is not an Issue custom resource"
	errGetIssue          = "failed to get issue"
	errCreateIssue       = "failed to create issue"
	errUpdateIssue       = "failed to update issue"
	errDeleteIssue       = "failed to delete issue"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected numeric issue number"
)

// Setup adds a controller that reconciles Issue managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.IssueKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.IssueGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.Issue{}).
		// A non-nil rate limiter is mandatory: ratelimiter.Reconciler.When()
		// dereferences it every reconcile, so a nil limiter panics on the first
		// event (lesson #1). o.ForControllerRuntime() also carries
		// MaxConcurrentReconciles through; without WithOptions both are dropped.
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.Issue)
	if !ok {
		return nil, errors.New(errNotIssue)
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

// external observes/creates/updates/deletes the backend issue.
type external struct {
	client clients.Client
}

// issueNumber parses the numeric issue number carried by the
// crossplane.io/external-name annotation. The annotation is authoritative for
// Observe, Update AND Delete (lesson #14). The owner/repo always come from
// cr.Spec.ForProvider.
func issueNumber(cr *v2.Issue) (int64, bool) {
	n, err := strconv.ParseInt(meta.GetExternalName(cr), 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Issue)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIssue)
	}

	number, ok := issueNumber(cr)
	if !ok {
		// No usable external-name yet -> not created. Don't try to GET it.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	owner := cr.Spec.ForProvider.Owner
	repo := cr.Spec.ForProvider.Repository

	issue, err := e.client.GetIssue(ctx, owner, repo, number)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). Real failures (auth/network/5xx) must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetIssue)
	}

	cr.Status.AtProvider = v2.IssueObservation{
		ID:        issue.ID,
		Number:    issue.Number,
		URL:       issue.HTMLURL,
		State:     issue.State,
		Comments:  issue.Comments,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
		ClosedAt:  issue.ClosedAt,
	}
	if issue.User != nil {
		cr.Status.AtProvider.Author = issue.User.Username
	}

	upToDate := issueUpToDate(cr, issue)

	// crossplane-runtime v2's managed reconciler no longer auto-sets
	// Available(); readiness is the provider's job (lesson #2/#6). Set Available
	// on the exists path; drift is signalled via ResourceUpToDate, never by
	// withholding Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// issueUpToDate reports whether the observed backend issue matches the managed
// fields of the desired spec. Only fields this provider pushes on Update are
// compared; a nil desired pointer means "do not manage".
func issueUpToDate(cr *v2.Issue, observed *clients.Issue) bool {
	p := cr.Spec.ForProvider
	if p.Title != observed.Title {
		return false
	}
	if p.Body != nil && *p.Body != observed.Body {
		return false
	}
	if p.State != nil && *p.State != observed.State {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Issue)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIssue)
	}
	cr.SetConditions(xpv1.Creating())

	createReq := &clients.CreateIssueOptions{
		Title:     cr.Spec.ForProvider.Title,
		Body:      cr.Spec.ForProvider.Body,
		Assignees: cr.Spec.ForProvider.Assignees,
		Labels:    cr.Spec.ForProvider.Labels,
		Milestone: cr.Spec.ForProvider.Milestone,
	}

	issue, err := e.client.CreateIssue(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateIssue)
	}

	// Capture the authoritative issue number from the backend response and pin
	// it as the external name (lesson #3/#7/#14).
	meta.SetExternalName(cr, strconv.FormatInt(issue.Number, 10))

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Issue)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIssue)
	}

	number, ok := issueNumber(cr)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	updateReq := &clients.UpdateIssueOptions{
		Title:     &cr.Spec.ForProvider.Title,
		Body:      cr.Spec.ForProvider.Body,
		State:     cr.Spec.ForProvider.State,
		Assignees: cr.Spec.ForProvider.Assignees,
		Labels:    cr.Spec.ForProvider.Labels,
		Milestone: cr.Spec.ForProvider.Milestone,
	}

	if _, err := e.client.UpdateIssue(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, number, updateReq); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateIssue)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Issue)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotIssue)
	}
	cr.SetConditions(xpv1.Deleting())

	number, ok := issueNumber(cr)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteIssue(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, number)
	// Treat an already-absent issue as a successful delete (idempotent, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteIssue)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
