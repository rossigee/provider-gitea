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
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/issue/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotIssue          = "managed resource is not an Issue custom resource"
	errGetIssue          = "failed to get issue"
	errCreateIssue       = "failed to create issue"
	errUpdateIssue       = "failed to update issue"
	errDeleteIssue       = "failed to delete issue"
	errGetProviderConfig = "failed to get provider config"
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
	cr, ok := mg.(*v2.Issue)
	if !ok {
		return nil, errors.New(errNotIssue)
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

// parseExternalName splits an "owner/repo/number" external name.
func parseExternalName(externalID string) (owner, repo string, number int64, err error) {
	parts := strings.Split(externalID, "/")
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("invalid external-id format %q, expected owner/repo/number", externalID)
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &number); err != nil {
		return "", "", 0, fmt.Errorf("invalid external-id %q: %w", externalID, err)
	}
	return parts[0], parts[1], number, nil
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "issue.observe",
		tracing.SpanAttrs("issue", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.Issue)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIssue)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	owner, repo, number, err := parseExternalName(externalID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	issue, err := e.client.GetIssue(ctx, owner, repo, number)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetIssue)
	}

	// Update observed state
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

	uo := isIssueUpToDate(cr, issue)

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func isIssueUpToDate(cr *v2.Issue, issue *clients.Issue) bool {
	fp := cr.Spec.ForProvider
	if fp.Title != issue.Title {
		return false
	}
	if fp.Body != nil && *fp.Body != issue.Body {
		return false
	}
	wantState := "open"
	if fp.State != nil {
		wantState = *fp.State
	}
	if issue.State != wantState {
		return false
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "issue.create",
		tracing.SpanAttrs("issue", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.Issue)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIssue)
	}

	fp := cr.Spec.ForProvider
	issue, err := e.client.CreateIssue(ctx, fp.Owner, fp.Repository, &clients.CreateIssueOptions{
		Title:     fp.Title,
		Body:      fp.Body,
		Assignees: fp.Assignees,
		Labels:    fp.Labels,
		Milestone: fp.Milestone,
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateIssue)
	}

	// Set external name to owner/repo/number format for future lookups
	meta.SetExternalName(cr, fmt.Sprintf("%s/%s/%d", fp.Owner, fp.Repository, issue.Number))

	// Close immediately if desired state is closed
	if fp.State != nil && *fp.State == "closed" {
		state := "closed"
		if _, err := e.client.UpdateIssue(ctx, fp.Owner, fp.Repository, issue.Number, &clients.UpdateIssueOptions{
			State: &state,
		}); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateIssue)
		}
	}

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "issue.update",
		tracing.SpanAttrs("issue", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.Issue)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIssue)
	}

	owner, repo, number, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	fp := cr.Spec.ForProvider
	title := fp.Title
	_, err = e.client.UpdateIssue(ctx, owner, repo, number, &clients.UpdateIssueOptions{
		Title:     &title,
		Body:      fp.Body,
		State:     fp.State,
		Assignees: fp.Assignees,
		Labels:    fp.Labels,
		Milestone: fp.Milestone,
	})
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateIssue)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "issue.delete",
		tracing.SpanAttrs("issue", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.Issue)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotIssue)
	}

	owner, repo, number, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetIssue(ctx, owner, repo, number); err == nil {
		if err := e.client.DeleteIssue(ctx, owner, repo, number); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteIssue)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteIssue)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.IssueKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.IssueGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Issue{}).
		Complete(r)
}
