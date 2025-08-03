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

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane-contrib/provider-gitea/apis/branchprotection/v1alpha1"
	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
	giteaclients "github.com/crossplane-contrib/provider-gitea/internal/clients"
)

const (
	errNotBranchProtection = "managed resource is not a BranchProtection custom resource"
	errTrackPCUsage        = "cannot track ProviderConfig usage"
	errGetPC               = "cannot get ProviderConfig"
	errGetCreds            = "cannot get credentials"
	errNewClient           = "cannot create new Service"
	errGetBranchProtection = "cannot get branch protection"
	errCreateBranchProtection = "cannot create branch protection"
	errUpdateBranchProtection = "cannot update branch protection"
	errDeleteBranchProtection = "cannot delete branch protection"
)

// Setup adds a controller that reconciles BranchProtection managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.BranchProtectionKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.BranchProtectionGroupVersionKind),
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
		For(&v1alpha1.BranchProtection{}).
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
	cr, ok := mg.(*v1alpha1.BranchProtection)
	if !ok {
		return nil, errors.New(errNotBranchProtection)
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
	cr, ok := mg.(*v1alpha1.BranchProtection)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBranchProtection)
	}

	// External name format: repository/branch/rule (e.g., "myorg/myrepo/main/main-protection")
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		externalName = fmt.Sprintf("%s/%s/%s", cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.Branch, cr.Spec.ForProvider.RuleName)
		meta.SetExternalName(cr, externalName)
	}

	protection, err := c.client.GetBranchProtection(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.Branch)
	if err != nil {
		// If protection doesn't exist, it needs to be created
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.BranchProtectionObservation{
		RuleName:  &protection.RuleName,
		CreatedAt: &protection.CreatedAt,
		UpdatedAt: &protection.UpdatedAt,
		AppliedSettings: &v1alpha1.BranchProtectionAppliedSettings{
			EnablePush:                       &protection.EnablePush,
			EnablePushWhitelist:              &protection.EnablePushWhitelist,
			PushWhitelistUsernames:           protection.PushWhitelistUsernames,
			PushWhitelistTeams:               protection.PushWhitelistTeams,
			PushWhitelistDeployKeys:          &protection.PushWhitelistDeployKeys,
			EnableMergeWhitelist:             &protection.EnableMergeWhitelist,
			MergeWhitelistUsernames:          protection.MergeWhitelistUsernames,
			MergeWhitelistTeams:              protection.MergeWhitelistTeams,
			EnableStatusCheck:                &protection.EnableStatusCheck,
			StatusCheckContexts:              protection.StatusCheckContexts,
			RequiredApprovals:                &protection.RequiredApprovals,
			EnableApprovalsWhitelist:         &protection.EnableApprovalsWhitelist,
			ApprovalsWhitelistUsernames:      protection.ApprovalsWhitelistUsernames,
			ApprovalsWhitelistTeams:          protection.ApprovalsWhitelistTeams,
			BlockOnRejectedReviews:           &protection.BlockOnRejectedReviews,
			BlockOnOfficialReviewRequests:    &protection.BlockOnOfficialReviewRequests,
			BlockOnOutdatedBranch:            &protection.BlockOnOutdatedBranch,
			DismissStaleApprovals:            &protection.DismissStaleApprovals,
			RequireSignedCommits:             &protection.RequireSignedCommits,
			ProtectedFilePatterns:            &protection.ProtectedFilePatterns,
			UnprotectedFilePatterns:          &protection.UnprotectedFilePatterns,
		},
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, protection),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.BranchProtection)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotBranchProtection)
	}

	cr.SetConditions(xpv1.Creating())

	req := c.buildCreateRequest(cr)

	_, err := c.client.CreateBranchProtection(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.Branch, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateBranchProtection)
	}

	// Set external name to repository/branch/rule format
	externalName := fmt.Sprintf("%s/%s/%s", cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.Branch, cr.Spec.ForProvider.RuleName)
	meta.SetExternalName(cr, externalName)

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.BranchProtection)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotBranchProtection)
	}

	req := c.buildUpdateRequest(cr)

	_, err := c.client.UpdateBranchProtection(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.Branch, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBranchProtection)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.BranchProtection)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotBranchProtection)
	}

	err := c.client.DeleteBranchProtection(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.Branch)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteBranchProtection)
	}

	return managed.ExternalDelete{}, nil
}

// buildCreateRequest creates a create request from the CR spec
func (c *external) buildCreateRequest(cr *v1alpha1.BranchProtection) *giteaclients.CreateBranchProtectionRequest {
	req := &giteaclients.CreateBranchProtectionRequest{
		RuleName: cr.Spec.ForProvider.RuleName,
	}

	if cr.Spec.ForProvider.EnablePush != nil {
		req.EnablePush = cr.Spec.ForProvider.EnablePush
	}
	if cr.Spec.ForProvider.EnablePushWhitelist != nil {
		req.EnablePushWhitelist = cr.Spec.ForProvider.EnablePushWhitelist
	}
	if cr.Spec.ForProvider.PushWhitelistUsernames != nil {
		req.PushWhitelistUsernames = cr.Spec.ForProvider.PushWhitelistUsernames
	}
	if cr.Spec.ForProvider.PushWhitelistTeams != nil {
		req.PushWhitelistTeams = cr.Spec.ForProvider.PushWhitelistTeams
	}
	if cr.Spec.ForProvider.PushWhitelistDeployKeys != nil {
		req.PushWhitelistDeployKeys = cr.Spec.ForProvider.PushWhitelistDeployKeys
	}
	if cr.Spec.ForProvider.EnableMergeWhitelist != nil {
		req.EnableMergeWhitelist = cr.Spec.ForProvider.EnableMergeWhitelist
	}
	if cr.Spec.ForProvider.MergeWhitelistUsernames != nil {
		req.MergeWhitelistUsernames = cr.Spec.ForProvider.MergeWhitelistUsernames
	}
	if cr.Spec.ForProvider.MergeWhitelistTeams != nil {
		req.MergeWhitelistTeams = cr.Spec.ForProvider.MergeWhitelistTeams
	}
	if cr.Spec.ForProvider.EnableStatusCheck != nil {
		req.EnableStatusCheck = cr.Spec.ForProvider.EnableStatusCheck
	}
	if cr.Spec.ForProvider.StatusCheckContexts != nil {
		req.StatusCheckContexts = cr.Spec.ForProvider.StatusCheckContexts
	}
	if cr.Spec.ForProvider.RequiredApprovals != nil {
		req.RequiredApprovals = cr.Spec.ForProvider.RequiredApprovals
	}
	if cr.Spec.ForProvider.EnableApprovalsWhitelist != nil {
		req.EnableApprovalsWhitelist = cr.Spec.ForProvider.EnableApprovalsWhitelist
	}
	if cr.Spec.ForProvider.ApprovalsWhitelistUsernames != nil {
		req.ApprovalsWhitelistUsernames = cr.Spec.ForProvider.ApprovalsWhitelistUsernames
	}
	if cr.Spec.ForProvider.ApprovalsWhitelistTeams != nil {
		req.ApprovalsWhitelistTeams = cr.Spec.ForProvider.ApprovalsWhitelistTeams
	}
	if cr.Spec.ForProvider.BlockOnRejectedReviews != nil {
		req.BlockOnRejectedReviews = cr.Spec.ForProvider.BlockOnRejectedReviews
	}
	if cr.Spec.ForProvider.BlockOnOfficialReviewRequests != nil {
		req.BlockOnOfficialReviewRequests = cr.Spec.ForProvider.BlockOnOfficialReviewRequests
	}
	if cr.Spec.ForProvider.BlockOnOutdatedBranch != nil {
		req.BlockOnOutdatedBranch = cr.Spec.ForProvider.BlockOnOutdatedBranch
	}
	if cr.Spec.ForProvider.DismissStaleApprovals != nil {
		req.DismissStaleApprovals = cr.Spec.ForProvider.DismissStaleApprovals
	}
	if cr.Spec.ForProvider.RequireSignedCommits != nil {
		req.RequireSignedCommits = cr.Spec.ForProvider.RequireSignedCommits
	}
	if cr.Spec.ForProvider.ProtectedFilePatterns != nil {
		req.ProtectedFilePatterns = cr.Spec.ForProvider.ProtectedFilePatterns
	}
	if cr.Spec.ForProvider.UnprotectedFilePatterns != nil {
		req.UnprotectedFilePatterns = cr.Spec.ForProvider.UnprotectedFilePatterns
	}

	return req
}

// buildUpdateRequest creates an update request from the CR spec
func (c *external) buildUpdateRequest(cr *v1alpha1.BranchProtection) *giteaclients.UpdateBranchProtectionRequest {
	req := &giteaclients.UpdateBranchProtectionRequest{}

	if cr.Spec.ForProvider.EnablePush != nil {
		req.EnablePush = cr.Spec.ForProvider.EnablePush
	}
	if cr.Spec.ForProvider.EnablePushWhitelist != nil {
		req.EnablePushWhitelist = cr.Spec.ForProvider.EnablePushWhitelist
	}
	if cr.Spec.ForProvider.PushWhitelistUsernames != nil {
		req.PushWhitelistUsernames = cr.Spec.ForProvider.PushWhitelistUsernames
	}
	if cr.Spec.ForProvider.PushWhitelistTeams != nil {
		req.PushWhitelistTeams = cr.Spec.ForProvider.PushWhitelistTeams
	}
	if cr.Spec.ForProvider.PushWhitelistDeployKeys != nil {
		req.PushWhitelistDeployKeys = cr.Spec.ForProvider.PushWhitelistDeployKeys
	}
	if cr.Spec.ForProvider.EnableMergeWhitelist != nil {
		req.EnableMergeWhitelist = cr.Spec.ForProvider.EnableMergeWhitelist
	}
	if cr.Spec.ForProvider.MergeWhitelistUsernames != nil {
		req.MergeWhitelistUsernames = cr.Spec.ForProvider.MergeWhitelistUsernames
	}
	if cr.Spec.ForProvider.MergeWhitelistTeams != nil {
		req.MergeWhitelistTeams = cr.Spec.ForProvider.MergeWhitelistTeams
	}
	if cr.Spec.ForProvider.EnableStatusCheck != nil {
		req.EnableStatusCheck = cr.Spec.ForProvider.EnableStatusCheck
	}
	if cr.Spec.ForProvider.StatusCheckContexts != nil {
		req.StatusCheckContexts = cr.Spec.ForProvider.StatusCheckContexts
	}
	if cr.Spec.ForProvider.RequiredApprovals != nil {
		req.RequiredApprovals = cr.Spec.ForProvider.RequiredApprovals
	}
	if cr.Spec.ForProvider.EnableApprovalsWhitelist != nil {
		req.EnableApprovalsWhitelist = cr.Spec.ForProvider.EnableApprovalsWhitelist
	}
	if cr.Spec.ForProvider.ApprovalsWhitelistUsernames != nil {
		req.ApprovalsWhitelistUsernames = cr.Spec.ForProvider.ApprovalsWhitelistUsernames
	}
	if cr.Spec.ForProvider.ApprovalsWhitelistTeams != nil {
		req.ApprovalsWhitelistTeams = cr.Spec.ForProvider.ApprovalsWhitelistTeams
	}
	if cr.Spec.ForProvider.BlockOnRejectedReviews != nil {
		req.BlockOnRejectedReviews = cr.Spec.ForProvider.BlockOnRejectedReviews
	}
	if cr.Spec.ForProvider.BlockOnOfficialReviewRequests != nil {
		req.BlockOnOfficialReviewRequests = cr.Spec.ForProvider.BlockOnOfficialReviewRequests
	}
	if cr.Spec.ForProvider.BlockOnOutdatedBranch != nil {
		req.BlockOnOutdatedBranch = cr.Spec.ForProvider.BlockOnOutdatedBranch
	}
	if cr.Spec.ForProvider.DismissStaleApprovals != nil {
		req.DismissStaleApprovals = cr.Spec.ForProvider.DismissStaleApprovals
	}
	if cr.Spec.ForProvider.RequireSignedCommits != nil {
		req.RequireSignedCommits = cr.Spec.ForProvider.RequireSignedCommits
	}
	if cr.Spec.ForProvider.ProtectedFilePatterns != nil {
		req.ProtectedFilePatterns = cr.Spec.ForProvider.ProtectedFilePatterns
	}
	if cr.Spec.ForProvider.UnprotectedFilePatterns != nil {
		req.UnprotectedFilePatterns = cr.Spec.ForProvider.UnprotectedFilePatterns
	}

	return req
}

// isUpToDate checks if the branch protection is up to date with the desired state
func (c *external) isUpToDate(cr *v1alpha1.BranchProtection, protection *giteaclients.BranchProtection) bool {
	if cr.Spec.ForProvider.EnablePush != nil && *cr.Spec.ForProvider.EnablePush != protection.EnablePush {
		return false
	}
	if cr.Spec.ForProvider.EnablePushWhitelist != nil && *cr.Spec.ForProvider.EnablePushWhitelist != protection.EnablePushWhitelist {
		return false
	}
	if cr.Spec.ForProvider.RequiredApprovals != nil && *cr.Spec.ForProvider.RequiredApprovals != protection.RequiredApprovals {
		return false
	}
	if cr.Spec.ForProvider.RequireSignedCommits != nil && *cr.Spec.ForProvider.RequireSignedCommits != protection.RequireSignedCommits {
		return false
	}
	// Add more comparisons as needed for critical settings

	return true
}