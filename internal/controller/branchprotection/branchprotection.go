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

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	branchprotectionv2 "github.com/rossigee/provider-gitea/apis/branchprotection/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotBranchProtection    = "managed resource is not a BranchProtection"
	errGetProviderConfig      = "cannot get ProviderConfig"
	errProviderNotReady       = "provider is not ready"
	errGetBranchProtection    = "cannot get branch protection"
	errCreateBranchProtection = "cannot create branch protection"
	errUpdateBranchProtection = "cannot update branch protection"
	errDeleteBranchProtection = "cannot delete branch protection"
	errParseRepository        = "cannot parse repository"
	controllerName            = "branchprotections.branchprotection.gitea.m.crossplane.io"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(branchprotectionv2.BranchProtectionGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", "BranchProtection")),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(controllerName))),
		managed.WithPollInterval(o.PollInterval),
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&branchprotectionv2.BranchProtection{}).
		Complete(r)
}

type connector struct{ kube client.Client }
type external struct{ client giteaclients.Client }

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*branchprotectionv2.BranchProtection)
	if !ok {
		return nil, errors.New(errNotBranchProtection)
	}
	pcRef := cr.Spec.ProviderConfigReference
	if pcRef == nil {
		return nil, errors.New(errGetProviderConfig + ": providerConfigRef is required")
	}
	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: pcRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}
	if pc.Status.GetCondition(xpv1.TypeReady).Status != corev1.ConditionTrue {
		return nil, errors.New(errProviderNotReady)
	}
	cl, err := giteaclients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Gitea client")
	}
	return &external{client: cl}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*branchprotectionv2.BranchProtection)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBranchProtection)
	}
	bp, err := e.client.GetBranchProtection(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.Branch)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetBranchProtection)
	}
	cr.Status.AtProvider = branchprotectionv2.BranchProtectionObservation{
		RuleName: &bp.RuleName,
		CreatedAt: &bp.CreatedAt,
		UpdatedAt: &bp.UpdatedAt,
	}
	cr.Status.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: branchProtectionUpToDate(&cr.Spec.ForProvider, bp),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*branchprotectionv2.BranchProtection)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotBranchProtection)
	}
	req := &giteaclients.CreateBranchProtectionRequest{
		RuleName: cr.Spec.ForProvider.RuleName,
	}
	mapBranchProtectionFields(&cr.Spec.ForProvider, req)
	_, err := e.client.CreateBranchProtection(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.Branch, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateBranchProtection)
	}
	cr.Status.SetConditions(xpv1.Creating())
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*branchprotectionv2.BranchProtection)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotBranchProtection)
	}
	req := &giteaclients.UpdateBranchProtectionRequest{}
	mapBranchProtectionFields(&cr.Spec.ForProvider, req)
	_, err := e.client.UpdateBranchProtection(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.Branch, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBranchProtection)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*branchprotectionv2.BranchProtection)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotBranchProtection)
	}
	err := e.client.DeleteBranchProtection(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.Branch)
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteBranchProtection)
	}
	cr.Status.SetConditions(xpv1.Deleting())
	return managed.ExternalDelete{}, nil
}

func mapBranchProtectionFields(desired *branchprotectionv2.BranchProtectionParameters, req interface{}) {
	if desired.EnablePush != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.EnablePush = desired.EnablePush
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.EnablePush = desired.EnablePush
		}
	}
	if desired.EnablePushWhitelist != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.EnablePushWhitelist = desired.EnablePushWhitelist
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.EnablePushWhitelist = desired.EnablePushWhitelist
		}
	}
	if desired.PushWhitelistUsernames != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.PushWhitelistUsernames = desired.PushWhitelistUsernames
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.PushWhitelistUsernames = desired.PushWhitelistUsernames
		}
	}
	if desired.PushWhitelistTeams != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.PushWhitelistTeams = desired.PushWhitelistTeams
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.PushWhitelistTeams = desired.PushWhitelistTeams
		}
	}
	if desired.EnableMergeWhitelist != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.EnableMergeWhitelist = desired.EnableMergeWhitelist
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.EnableMergeWhitelist = desired.EnableMergeWhitelist
		}
	}
	if desired.MergeWhitelistUsernames != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.MergeWhitelistUsernames = desired.MergeWhitelistUsernames
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.MergeWhitelistUsernames = desired.MergeWhitelistUsernames
		}
	}
	if desired.MergeWhitelistTeams != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.MergeWhitelistTeams = desired.MergeWhitelistTeams
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.MergeWhitelistTeams = desired.MergeWhitelistTeams
		}
	}
	if desired.EnableStatusCheck != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.EnableStatusCheck = desired.EnableStatusCheck
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.EnableStatusCheck = desired.EnableStatusCheck
		}
	}
	if desired.StatusCheckContexts != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.StatusCheckContexts = desired.StatusCheckContexts
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.StatusCheckContexts = desired.StatusCheckContexts
		}
	}
	if desired.RequiredApprovals != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.RequiredApprovals = desired.RequiredApprovals
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.RequiredApprovals = desired.RequiredApprovals
		}
	}
	if desired.EnableApprovalsWhitelist != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.EnableApprovalsWhitelist = desired.EnableApprovalsWhitelist
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.EnableApprovalsWhitelist = desired.EnableApprovalsWhitelist
		}
	}
	if desired.ApprovalsWhitelistUsernames != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.ApprovalsWhitelistUsernames = desired.ApprovalsWhitelistUsernames
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.ApprovalsWhitelistUsernames = desired.ApprovalsWhitelistUsernames
		}
	}
	if desired.ApprovalsWhitelistTeams != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.ApprovalsWhitelistTeams = desired.ApprovalsWhitelistTeams
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.ApprovalsWhitelistTeams = desired.ApprovalsWhitelistTeams
		}
	}
	if desired.BlockOnRejectedReviews != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.BlockOnRejectedReviews = desired.BlockOnRejectedReviews
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.BlockOnRejectedReviews = desired.BlockOnRejectedReviews
		}
	}
	if desired.BlockOnOfficialReviewRequests != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.BlockOnOfficialReviewRequests = desired.BlockOnOfficialReviewRequests
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.BlockOnOfficialReviewRequests = desired.BlockOnOfficialReviewRequests
		}
	}
	if desired.BlockOnOutdatedBranch != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.BlockOnOutdatedBranch = desired.BlockOnOutdatedBranch
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.BlockOnOutdatedBranch = desired.BlockOnOutdatedBranch
		}
	}
	if desired.DismissStaleApprovals != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.DismissStaleApprovals = desired.DismissStaleApprovals
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.DismissStaleApprovals = desired.DismissStaleApprovals
		}
	}
	if desired.RequireSignedCommits != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.RequireSignedCommits = desired.RequireSignedCommits
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.RequireSignedCommits = desired.RequireSignedCommits
		}
	}
	if desired.ProtectedFilePatterns != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.ProtectedFilePatterns = desired.ProtectedFilePatterns
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.ProtectedFilePatterns = desired.ProtectedFilePatterns
		}
	}
	if desired.UnprotectedFilePatterns != nil {
		if cr, ok := req.(*giteaclients.CreateBranchProtectionRequest); ok {
			cr.UnprotectedFilePatterns = desired.UnprotectedFilePatterns
		} else if ur, ok := req.(*giteaclients.UpdateBranchProtectionRequest); ok {
			ur.UnprotectedFilePatterns = desired.UnprotectedFilePatterns
		}
	}
}

func slicesEqual(a, b []string) bool {
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

func branchProtectionUpToDate(desired *branchprotectionv2.BranchProtectionParameters, actual *giteaclients.BranchProtection) bool {
	if desired.EnablePush != nil && *desired.EnablePush != actual.EnablePush {
		return false
	}
	if desired.EnablePushWhitelist != nil && *desired.EnablePushWhitelist != actual.EnablePushWhitelist {
		return false
	}
	if desired.PushWhitelistUsernames != nil && !slicesEqual(desired.PushWhitelistUsernames, actual.PushWhitelistUsernames) {
		return false
	}
	if desired.PushWhitelistTeams != nil && !slicesEqual(desired.PushWhitelistTeams, actual.PushWhitelistTeams) {
		return false
	}
	if desired.EnableMergeWhitelist != nil && *desired.EnableMergeWhitelist != actual.EnableMergeWhitelist {
		return false
	}
	if desired.MergeWhitelistUsernames != nil && !slicesEqual(desired.MergeWhitelistUsernames, actual.MergeWhitelistUsernames) {
		return false
	}
	if desired.MergeWhitelistTeams != nil && !slicesEqual(desired.MergeWhitelistTeams, actual.MergeWhitelistTeams) {
		return false
	}
	if desired.EnableStatusCheck != nil && *desired.EnableStatusCheck != actual.EnableStatusCheck {
		return false
	}
	if desired.StatusCheckContexts != nil && !slicesEqual(desired.StatusCheckContexts, actual.StatusCheckContexts) {
		return false
	}
	if desired.RequiredApprovals != nil && *desired.RequiredApprovals != actual.RequiredApprovals {
		return false
	}
	if desired.EnableApprovalsWhitelist != nil && *desired.EnableApprovalsWhitelist != actual.EnableApprovalsWhitelist {
		return false
	}
	if desired.ApprovalsWhitelistUsernames != nil && !slicesEqual(desired.ApprovalsWhitelistUsernames, actual.ApprovalsWhitelistUsernames) {
		return false
	}
	if desired.ApprovalsWhitelistTeams != nil && !slicesEqual(desired.ApprovalsWhitelistTeams, actual.ApprovalsWhitelistTeams) {
		return false
	}
	if desired.BlockOnRejectedReviews != nil && *desired.BlockOnRejectedReviews != actual.BlockOnRejectedReviews {
		return false
	}
	if desired.BlockOnOfficialReviewRequests != nil && *desired.BlockOnOfficialReviewRequests != actual.BlockOnOfficialReviewRequests {
		return false
	}
	if desired.BlockOnOutdatedBranch != nil && *desired.BlockOnOutdatedBranch != actual.BlockOnOutdatedBranch {
		return false
	}
	if desired.DismissStaleApprovals != nil && *desired.DismissStaleApprovals != actual.DismissStaleApprovals {
		return false
	}
	if desired.RequireSignedCommits != nil && *desired.RequireSignedCommits != actual.RequireSignedCommits {
		return false
	}
	if desired.ProtectedFilePatterns != nil && *desired.ProtectedFilePatterns != actual.ProtectedFilePatterns {
		return false
	}
	if desired.UnprotectedFilePatterns != nil && *desired.UnprotectedFilePatterns != actual.UnprotectedFilePatterns {
		return false
	}
	return true
}
