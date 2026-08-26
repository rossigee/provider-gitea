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

package branchprotection

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/branchprotection/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotBranchProtection    = "managed resource is not a BranchProtection custom resource"
	errGetBranchProtection    = "failed to get branch protection"
	errCreateBranchProtection = "failed to create branch protection"
	errUpdateBranchProtection = "failed to update branch protection"
	errDeleteBranchProtection = "failed to delete branch protection"
	errGetProviderConfig      = "failed to get provider config"
	errInvalidExternalID      = "external-name must be in format 'owner/repo:branch'"
)

// A connector is expected to produce an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect returns an ExternalClient by getting the provider config and creating a Gitea API client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.BranchProtection)
	if !ok {
		return nil, errors.New(errNotBranchProtection)
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

// An ExternalClient observes, then either creates, updates, or deletes an external resource.
type externalClient struct {
	client clients.Client
}

// parseExternalID splits an external-name of the form "owner/repo:branch" into owner/repo and branch.
func parseExternalID(externalID string) (repository, branch string, err error) {
	parts := strings.SplitN(externalID, ":", 2)
	if len(parts) != 2 {
		return "", "", errors.New(errInvalidExternalID)
	}
	repo := parts[0]
	branch = parts[1]
	rp := strings.Split(repo, "/")
	if len(rp) != 2 {
		return "", "", errors.New(errInvalidExternalID)
	}
	return repo, branch, nil
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "branchprotection.observe",
		tracing.SpanAttrs("branchprotection", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.BranchProtection)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBranchProtection)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	repository, branch, err := parseExternalID(externalID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	protection, err := e.client.GetBranchProtection(ctx, repository, branch)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetBranchProtection)
	}

	ruleName := protection.RuleName
	createdAt := protection.CreatedAt
	updatedAt := protection.UpdatedAt
	cr.Status.AtProvider = v2.BranchProtectionObservation{
		RuleName:  &ruleName,
		CreatedAt: &createdAt,
		UpdatedAt: &updatedAt,
		AppliedSettings: &v2.BranchProtectionAppliedSettings{
			EnablePush:        &protection.EnablePush,
			EnableStatusCheck: &protection.EnableStatusCheck,
		},
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUpToDate(cr, protection),
	}, nil
}

func boolPtrEqual(a *bool, b bool) bool {
	if a == nil {
		return !b
	}
	return *a == b
}

func strPtrEqual(a *string, b string) bool {
	if a == nil {
		return b == ""
	}
	return *a == b
}

func intPtrEqual(a *int, b int) bool {
	if a == nil {
		return b == 0
	}
	return *a == b
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

func isUpToDate(cr *v2.BranchProtection, p *clients.BranchProtection) bool {
	fp := &cr.Spec.ForProvider
	if !boolPtrEqual(fp.EnablePush, p.EnablePush) {
		return false
	}
	if !boolPtrEqual(fp.EnablePushWhitelist, p.EnablePushWhitelist) {
		return false
	}
	if !boolPtrEqual(fp.EnableMergeWhitelist, p.EnableMergeWhitelist) {
		return false
	}
	if !boolPtrEqual(fp.EnableStatusCheck, p.EnableStatusCheck) {
		return false
	}
	if !boolPtrEqual(fp.EnableApprovalsWhitelist, p.EnableApprovalsWhitelist) {
		return false
	}
	if !boolPtrEqual(fp.BlockOnRejectedReviews, p.BlockOnRejectedReviews) {
		return false
	}
	if !boolPtrEqual(fp.BlockOnOfficialReviewRequests, p.BlockOnOfficialReviewRequests) {
		return false
	}
	if !boolPtrEqual(fp.BlockOnOutdatedBranch, p.BlockOnOutdatedBranch) {
		return false
	}
	if !boolPtrEqual(fp.DismissStaleApprovals, p.DismissStaleApprovals) {
		return false
	}
	if !boolPtrEqual(fp.RequireSignedCommits, p.RequireSignedCommits) {
		return false
	}
	if !intPtrEqual(fp.RequiredApprovals, p.RequiredApprovals) {
		return false
	}
	if !strPtrEqual(fp.ProtectedFilePatterns, p.ProtectedFilePatterns) {
		return false
	}
	if !strPtrEqual(fp.UnprotectedFilePatterns, p.UnprotectedFilePatterns) {
		return false
	}
	if !strSliceEqual(fp.PushWhitelistUsernames, p.PushWhitelistUsernames) {
		return false
	}
	if !strSliceEqual(fp.PushWhitelistTeams, p.PushWhitelistTeams) {
		return false
	}
	if !strSliceEqual(fp.MergeWhitelistUsernames, p.MergeWhitelistUsernames) {
		return false
	}
	if !strSliceEqual(fp.MergeWhitelistTeams, p.MergeWhitelistTeams) {
		return false
	}
	if !strSliceEqual(fp.StatusCheckContexts, p.StatusCheckContexts) {
		return false
	}
	if !strSliceEqual(fp.ApprovalsWhitelistUsernames, p.ApprovalsWhitelistUsernames) {
		return false
	}
	if !strSliceEqual(fp.ApprovalsWhitelistTeams, p.ApprovalsWhitelistTeams) {
		return false
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "branchprotection.create",
		tracing.SpanAttrs("branchprotection", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.BranchProtection)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotBranchProtection)
	}

	externalID := meta.GetExternalName(cr)
	repository, branch, err := parseExternalID(externalID)
	if err != nil {
		// Derive from spec if external-name not yet set.
		repository = cr.Spec.ForProvider.Repository
		branch = cr.Spec.ForProvider.Branch
		if repository == "" || branch == "" {
			return managed.ExternalCreation{}, errors.New(errInvalidExternalID)
		}
	}

	fp := &cr.Spec.ForProvider
	req := &clients.CreateBranchProtectionRequest{
		RuleName:                      fp.RuleName,
		EnablePush:                    fp.EnablePush,
		EnablePushWhitelist:           fp.EnablePushWhitelist,
		PushWhitelistUsernames:        fp.PushWhitelistUsernames,
		PushWhitelistTeams:            fp.PushWhitelistTeams,
		PushWhitelistDeployKeys:       fp.PushWhitelistDeployKeys,
		EnableMergeWhitelist:          fp.EnableMergeWhitelist,
		MergeWhitelistUsernames:       fp.MergeWhitelistUsernames,
		MergeWhitelistTeams:           fp.MergeWhitelistTeams,
		EnableStatusCheck:             fp.EnableStatusCheck,
		StatusCheckContexts:           fp.StatusCheckContexts,
		RequiredApprovals:             fp.RequiredApprovals,
		EnableApprovalsWhitelist:      fp.EnableApprovalsWhitelist,
		ApprovalsWhitelistUsernames:   fp.ApprovalsWhitelistUsernames,
		ApprovalsWhitelistTeams:       fp.ApprovalsWhitelistTeams,
		BlockOnRejectedReviews:        fp.BlockOnRejectedReviews,
		BlockOnOfficialReviewRequests: fp.BlockOnOfficialReviewRequests,
		BlockOnOutdatedBranch:         fp.BlockOnOutdatedBranch,
		DismissStaleApprovals:         fp.DismissStaleApprovals,
		RequireSignedCommits:          fp.RequireSignedCommits,
		ProtectedFilePatterns:         fp.ProtectedFilePatterns,
		UnprotectedFilePatterns:       fp.UnprotectedFilePatterns,
	}

	if _, err := e.client.CreateBranchProtection(ctx, repository, branch, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateBranchProtection)
	}

	meta.SetExternalName(cr, fmt.Sprintf("%s:%s", repository, branch))

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "branchprotection.update",
		tracing.SpanAttrs("branchprotection", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.BranchProtection)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotBranchProtection)
	}

	externalID := meta.GetExternalName(cr)
	repository, branch, err := parseExternalID(externalID)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	fp := &cr.Spec.ForProvider
	req := &clients.UpdateBranchProtectionRequest{
		EnablePush:                    fp.EnablePush,
		EnablePushWhitelist:           fp.EnablePushWhitelist,
		PushWhitelistUsernames:        fp.PushWhitelistUsernames,
		PushWhitelistTeams:            fp.PushWhitelistTeams,
		PushWhitelistDeployKeys:       fp.PushWhitelistDeployKeys,
		EnableMergeWhitelist:          fp.EnableMergeWhitelist,
		MergeWhitelistUsernames:       fp.MergeWhitelistUsernames,
		MergeWhitelistTeams:           fp.MergeWhitelistTeams,
		EnableStatusCheck:             fp.EnableStatusCheck,
		StatusCheckContexts:           fp.StatusCheckContexts,
		RequiredApprovals:             fp.RequiredApprovals,
		EnableApprovalsWhitelist:      fp.EnableApprovalsWhitelist,
		ApprovalsWhitelistUsernames:   fp.ApprovalsWhitelistUsernames,
		ApprovalsWhitelistTeams:       fp.ApprovalsWhitelistTeams,
		BlockOnRejectedReviews:        fp.BlockOnRejectedReviews,
		BlockOnOfficialReviewRequests: fp.BlockOnOfficialReviewRequests,
		BlockOnOutdatedBranch:         fp.BlockOnOutdatedBranch,
		DismissStaleApprovals:         fp.DismissStaleApprovals,
		RequireSignedCommits:          fp.RequireSignedCommits,
		ProtectedFilePatterns:         fp.ProtectedFilePatterns,
		UnprotectedFilePatterns:       fp.UnprotectedFilePatterns,
	}

	if _, err := e.client.UpdateBranchProtection(ctx, repository, branch, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBranchProtection)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "branchprotection.delete",
		tracing.SpanAttrs("branchprotection", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.BranchProtection)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotBranchProtection)
	}

	externalID := meta.GetExternalName(cr)
	repository, branch, err := parseExternalID(externalID)
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	if _, err := e.client.GetBranchProtection(ctx, repository, branch); err == nil {
		if err := e.client.DeleteBranchProtection(ctx, repository, branch); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteBranchProtection)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteBranchProtection)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

// Setup adds a controller that reconciles BranchProtection managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.BranchProtectionKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.BranchProtectionGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.BranchProtection{}).
		Complete(r)
}
