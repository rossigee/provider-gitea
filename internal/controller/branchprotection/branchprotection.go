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

// Package branchprotection implements the Crossplane managed-resource reconciler
// for the Gitea BranchProtection resource. It mirrors the canonical reference
// controller (internal/controller/repository/repository.go). See
// crossplane-provider-template dev/docs/09-lessons-learned.md.
package branchprotection

import (
	"context"

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

	v2 "github.com/rossigee/provider-gitea/apis/branchprotection/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotBranchProtection    = "managed resource is not a BranchProtection custom resource"
	errGetBranchProtection    = "failed to get branchprotection"
	errCreateBranchProtection = "failed to create branchprotection"
	errUpdateBranchProtection = "failed to update branchprotection"
	errDeleteBranchProtection = "failed to delete branchprotection"
	errGetProviderConfig      = "failed to get provider config"
	errExternalName           = "invalid external-name, expected branch"
)

// Setup adds a controller that reconciles BranchProtection managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.BranchProtectionKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.BranchProtectionGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.BranchProtection{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube client.Client
}

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

	return &external{client: conn}, nil
}

type external struct {
	client clients.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.BranchProtection)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBranchProtection)
	}

	// Identity is the external-name (the branch); the parent repository is read
	// from spec every reconcile (lesson #14). Empty external-name -> not created
	// yet; don't issue a GET.
	branch := meta.GetExternalName(cr)
	if branch == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	bp, err := e.client.GetBranchProtection(ctx, cr.Spec.ForProvider.Repository, branch)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure (auth/network/5xx) must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetBranchProtection)
	}

	cr.Status.AtProvider = v2.BranchProtectionObservation{
		RuleName:  &bp.RuleName,
		CreatedAt: &bp.CreatedAt,
		UpdatedAt: &bp.UpdatedAt,
	}

	// crossplane-runtime v2 no longer auto-sets Available() (lesson #2/#6).
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: branchProtectionUpToDate(cr, bp),
	}, nil
}

// branchProtectionUpToDate compares the mutable scalar fields this provider
// manages against the observed rule. A nil desired pointer means "do not
// manage" and never causes drift.
func branchProtectionUpToDate(cr *v2.BranchProtection, observed *clients.BranchProtection) bool {
	p := cr.Spec.ForProvider
	if p.EnablePush != nil && *p.EnablePush != observed.EnablePush {
		return false
	}
	if p.EnableStatusCheck != nil && *p.EnableStatusCheck != observed.EnableStatusCheck {
		return false
	}
	if p.RequiredApprovals != nil && *p.RequiredApprovals != observed.RequiredApprovals {
		return false
	}
	if p.RequireSignedCommits != nil && *p.RequireSignedCommits != observed.RequireSignedCommits {
		return false
	}
	if p.ProtectedFilePatterns != nil && *p.ProtectedFilePatterns != observed.ProtectedFilePatterns {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.BranchProtection)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotBranchProtection)
	}
	cr.SetConditions(xpv1.Creating())

	p := cr.Spec.ForProvider
	req := &clients.CreateBranchProtectionRequest{
		RuleName:                      p.RuleName,
		EnablePush:                    p.EnablePush,
		EnablePushWhitelist:           p.EnablePushWhitelist,
		PushWhitelistUsernames:        p.PushWhitelistUsernames,
		PushWhitelistTeams:            p.PushWhitelistTeams,
		PushWhitelistDeployKeys:       p.PushWhitelistDeployKeys,
		EnableMergeWhitelist:          p.EnableMergeWhitelist,
		MergeWhitelistUsernames:       p.MergeWhitelistUsernames,
		MergeWhitelistTeams:           p.MergeWhitelistTeams,
		EnableStatusCheck:             p.EnableStatusCheck,
		StatusCheckContexts:           p.StatusCheckContexts,
		RequiredApprovals:             p.RequiredApprovals,
		EnableApprovalsWhitelist:      p.EnableApprovalsWhitelist,
		ApprovalsWhitelistUsernames:   p.ApprovalsWhitelistUsernames,
		ApprovalsWhitelistTeams:       p.ApprovalsWhitelistTeams,
		BlockOnRejectedReviews:        p.BlockOnRejectedReviews,
		BlockOnOfficialReviewRequests: p.BlockOnOfficialReviewRequests,
		BlockOnOutdatedBranch:         p.BlockOnOutdatedBranch,
		DismissStaleApprovals:         p.DismissStaleApprovals,
		RequireSignedCommits:          p.RequireSignedCommits,
		ProtectedFilePatterns:         p.ProtectedFilePatterns,
		UnprotectedFilePatterns:       p.UnprotectedFilePatterns,
	}
	if _, err := e.client.CreateBranchProtection(ctx, p.Repository, p.Branch, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateBranchProtection)
	}

	// For this resource the name (branch) IS the identity; pin it from spec
	// after a successful create so Observe/Update/Delete resolve from the
	// annotation.
	meta.SetExternalName(cr, p.Branch)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.BranchProtection)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotBranchProtection)
	}

	branch := meta.GetExternalName(cr)
	if branch == "" {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	p := cr.Spec.ForProvider
	req := &clients.UpdateBranchProtectionRequest{
		EnablePush:                    p.EnablePush,
		EnablePushWhitelist:           p.EnablePushWhitelist,
		PushWhitelistUsernames:        p.PushWhitelistUsernames,
		PushWhitelistTeams:            p.PushWhitelistTeams,
		PushWhitelistDeployKeys:       p.PushWhitelistDeployKeys,
		EnableMergeWhitelist:          p.EnableMergeWhitelist,
		MergeWhitelistUsernames:       p.MergeWhitelistUsernames,
		MergeWhitelistTeams:           p.MergeWhitelistTeams,
		EnableStatusCheck:             p.EnableStatusCheck,
		StatusCheckContexts:           p.StatusCheckContexts,
		RequiredApprovals:             p.RequiredApprovals,
		EnableApprovalsWhitelist:      p.EnableApprovalsWhitelist,
		ApprovalsWhitelistUsernames:   p.ApprovalsWhitelistUsernames,
		ApprovalsWhitelistTeams:       p.ApprovalsWhitelistTeams,
		BlockOnRejectedReviews:        p.BlockOnRejectedReviews,
		BlockOnOfficialReviewRequests: p.BlockOnOfficialReviewRequests,
		BlockOnOutdatedBranch:         p.BlockOnOutdatedBranch,
		DismissStaleApprovals:         p.DismissStaleApprovals,
		RequireSignedCommits:          p.RequireSignedCommits,
		ProtectedFilePatterns:         p.ProtectedFilePatterns,
		UnprotectedFilePatterns:       p.UnprotectedFilePatterns,
	}
	if _, err := e.client.UpdateBranchProtection(ctx, p.Repository, branch, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBranchProtection)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.BranchProtection)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotBranchProtection)
	}
	cr.SetConditions(xpv1.Deleting())

	branch := meta.GetExternalName(cr)
	if branch == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteBranchProtection(ctx, cr.Spec.ForProvider.Repository, branch)
	// An already-absent rule is a successful delete (idempotent, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteBranchProtection)
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}
