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
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/pullrequest/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotPullRequest    = "managed resource is not a PullRequest custom resource"
	errGetPullRequest    = "failed to get pull request"
	errCreatePullRequest = "failed to create pull request"
	errUpdatePullRequest = "failed to update pull request"
	errDeletePullRequest = "failed to delete pull request"
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
	cr, ok := mg.(*v2.PullRequest)
	if !ok {
		return nil, errors.New(errNotPullRequest)
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
	_, span := tracing.StartSpan(ctx, "pullrequest.observe",
		tracing.SpanAttrs("pullrequest", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.PullRequest)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPullRequest)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	owner, repo, number, err := parseExternalName(externalID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	pr, err := e.client.GetPullRequest(ctx, owner, repo, number)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetPullRequest)
	}

	// Update observed state
	cr.Status.AtProvider = v2.PullRequestObservation{
		ID:        pr.ID,
		Number:    pr.Number,
		URL:       pr.HTMLURL,
		State:     pr.State,
		Mergeable: pr.Mergeable,
		Merged:    pr.Merged,
		Comments:  pr.Comments,
	}

	uo := isPullRequestUpToDate(cr, pr)

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func isPullRequestUpToDate(cr *v2.PullRequest, pr *clients.PullRequest) bool {
	fp := cr.Spec.ForProvider
	if fp.Title != pr.Title {
		return false
	}
	if fp.Body != nil && *fp.Body != pr.Body {
		return false
	}
	wantState := "open"
	if fp.State != nil {
		wantState = *fp.State
	}
	// A merged PR reports state closed; treat merged as terminal match.
	if pr.Merged {
		return wantState == "merged" || wantState == "closed"
	}
	if pr.State != wantState {
		return false
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "pullrequest.create",
		tracing.SpanAttrs("pullrequest", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.PullRequest)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPullRequest)
	}

	fp := cr.Spec.ForProvider
	pr, err := e.client.CreatePullRequest(ctx, fp.Owner, fp.Repository, &clients.CreatePullRequestOptions{
		Title:         fp.Title,
		Body:          fp.Body,
		Head:          fp.Head,
		Base:          fp.Base,
		Assignees:     fp.Assignees,
		Reviewers:     fp.Reviewers,
		TeamReviewers: fp.TeamReviewers,
		Labels:        fp.Labels,
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreatePullRequest)
	}

	// Set external name to owner/repo/number format for future lookups
	meta.SetExternalName(cr, fmt.Sprintf("%s/%s/%d", fp.Owner, fp.Repository, pr.Number))

	// Close or merge immediately if that is the desired state.
	if fp.State != nil {
		switch *fp.State {
		case "closed":
			state := "closed"
			if _, err := e.client.UpdatePullRequest(ctx, fp.Owner, fp.Repository, pr.Number, &clients.UpdatePullRequestOptions{
				State: &state,
			}); err != nil {
				return managed.ExternalCreation{}, errors.Wrap(err, errCreatePullRequest)
			}
		case "merged":
			if _, err := e.client.MergePullRequest(ctx, fp.Owner, fp.Repository, pr.Number, &clients.MergePullRequestOptions{
				DoMerge: true,
			}); err != nil {
				return managed.ExternalCreation{}, errors.Wrap(err, errCreatePullRequest)
			}
		}
	}

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "pullrequest.update",
		tracing.SpanAttrs("pullrequest", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.PullRequest)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPullRequest)
	}

	owner, repo, number, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	fp := cr.Spec.ForProvider
	title := fp.Title
	if _, err := e.client.UpdatePullRequest(ctx, owner, repo, number, &clients.UpdatePullRequestOptions{
		Title:  &title,
		Body:   fp.Body,
		State:  fp.State,
		Labels: fp.Labels,
	}); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdatePullRequest)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "pullrequest.delete",
		tracing.SpanAttrs("pullrequest", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.PullRequest)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotPullRequest)
	}

	owner, repo, number, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetPullRequest(ctx, owner, repo, number); err == nil {
		if err := e.client.DeletePullRequest(ctx, owner, repo, number); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeletePullRequest)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeletePullRequest)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.PullRequestKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.PullRequestGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.PullRequest{}).
		Complete(r)
}
