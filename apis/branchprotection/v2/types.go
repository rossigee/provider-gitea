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

package v2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

type BranchProtectionParameters struct {
	// Repository is the repository that owns this branch protection rule (owner/name format)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Repository string `json:"repository"`

	// Branch is the branch name to protect
	// +kubebuilder:validation:Required
	Branch string `json:"branch"`

	// RuleName is the name of the protection rule
	// +kubebuilder:validation:Required
	RuleName string `json:"ruleName"`

	// EnablePush controls whether pushes are allowed
	// +kubebuilder:default=true
	EnablePush *bool `json:"enablePush,omitempty"`

	// EnablePushWhitelist enables push whitelist
	// +kubebuilder:default=false
	EnablePushWhitelist *bool `json:"enablePushWhitelist,omitempty"`

	// PushWhitelistUsernames is the list of usernames allowed to push
	PushWhitelistUsernames []string `json:"pushWhitelistUsernames,omitempty"`

	// PushWhitelistTeams is the list of teams allowed to push
	PushWhitelistTeams []string `json:"pushWhitelistTeams,omitempty"`

	// PushWhitelistDeployKeys allows deploy keys to push
	// +kubebuilder:default=false
	PushWhitelistDeployKeys *bool `json:"pushWhitelistDeployKeys,omitempty"`

	// EnableMergeWhitelist enables merge whitelist
	// +kubebuilder:default=false
	EnableMergeWhitelist *bool `json:"enableMergeWhitelist,omitempty"`

	// MergeWhitelistUsernames is the list of usernames allowed to merge
	MergeWhitelistUsernames []string `json:"mergeWhitelistUsernames,omitempty"`

	// MergeWhitelistTeams is the list of teams allowed to merge
	MergeWhitelistTeams []string `json:"mergeWhitelistTeams,omitempty"`

	// EnableStatusCheck enables status check requirements
	// +kubebuilder:default=false
	EnableStatusCheck *bool `json:"enableStatusCheck,omitempty"`

	// StatusCheckContexts is the list of required status check contexts
	StatusCheckContexts []string `json:"statusCheckContexts,omitempty"`

	// RequiredApprovals is the number of required approvals
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=0
	RequiredApprovals *int `json:"requiredApprovals,omitempty"`

	// EnableApprovalsWhitelist enables approval whitelist
	// +kubebuilder:default=false
	EnableApprovalsWhitelist *bool `json:"enableApprovalsWhitelist,omitempty"`

	// ApprovalsWhitelistUsernames is the list of usernames allowed to approve
	ApprovalsWhitelistUsernames []string `json:"approvalsWhitelistUsernames,omitempty"`

	// ApprovalsWhitelistTeams is the list of teams allowed to approve
	ApprovalsWhitelistTeams []string `json:"approvalsWhitelistTeams,omitempty"`

	// BlockOnRejectedReviews blocks merge when there are rejected reviews
	// +kubebuilder:default=false
	BlockOnRejectedReviews *bool `json:"blockOnRejectedReviews,omitempty"`

	// BlockOnOfficialReviewRequests blocks merge when official review is requested
	// +kubebuilder:default=false
	BlockOnOfficialReviewRequests *bool `json:"blockOnOfficialReviewRequests,omitempty"`

	// BlockOnOutdatedBranch blocks merge when branch is outdated
	// +kubebuilder:default=false
	BlockOnOutdatedBranch *bool `json:"blockOnOutdatedBranch,omitempty"`

	// DismissStaleApprovals dismisses stale approvals when new commits are pushed
	// +kubebuilder:default=false
	DismissStaleApprovals *bool `json:"dismissStaleApprovals,omitempty"`

	// RequireSignedCommits requires all commits to be signed
	// +kubebuilder:default=false
	RequireSignedCommits *bool `json:"requireSignedCommits,omitempty"`

	// ProtectedFilePatterns defines patterns for files that require special protection
	ProtectedFilePatterns *string `json:"protectedFilePatterns,omitempty"`

	// UnprotectedFilePatterns defines patterns for files that bypass protection
	UnprotectedFilePatterns *string `json:"unprotectedFilePatterns,omitempty"`

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

type BranchProtectionObservation struct {
	// RuleName is the name of the protection rule
	RuleName *string `json:"ruleName,omitempty"`

	// CreatedAt is the timestamp when the rule was created
	CreatedAt *string `json:"createdAt,omitempty"`

	// UpdatedAt is the timestamp when the rule was last updated
	UpdatedAt *string `json:"updatedAt,omitempty"`

	// Applied settings summary
	AppliedSettings *BranchProtectionAppliedSettings `json:"appliedSettings,omitempty"`

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

// BranchProtectionSpec defines the desired state of BranchProtection
type BranchProtectionSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       BranchProtectionParameters `json:"forProvider"`
}

// BranchProtectionStatus defines the observed state of BranchProtection
type BranchProtectionStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          BranchProtectionObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=bran
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// BranchProtection is the Schema for the branchprotections API v2 (namespaced)
type BranchProtection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BranchProtectionSpec   `json:"spec,omitempty"`
	Status BranchProtectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BranchProtectionList contains a list of BranchProtection
type BranchProtectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BranchProtection `json:"items"`
}

// BranchProtection type metadata
var (
	BranchProtectionKind             = "BranchProtection"
	BranchProtectionGroupKind        = schema.GroupKind{Group: Group, Kind: BranchProtectionKind}
	BranchProtectionKindAPIVersion   = BranchProtectionKind + "." + SchemeGroupVersion.String()
	BranchProtectionGroupVersionKind = SchemeGroupVersion.WithKind(BranchProtectionKind)
)

func init() {
	SchemeBuilder.Register(&BranchProtection{}, &BranchProtectionList{})
}
