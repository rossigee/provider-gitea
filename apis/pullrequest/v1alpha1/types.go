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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// PullRequestParameters define the desired state of a Gitea Pull Request
type PullRequestParameters struct {
	// Title is the title of the pull request
	// +kubebuilder:validation:Required
	Title string `json:"title"`

	// Body is the body content of the pull request
	Body *string `json:"body,omitempty"`

	// Repository is the name of the repository
	// +kubebuilder:validation:Required
	Repository string `json:"repository"`

	// Owner is the username or organization name that owns the repository
	// +kubebuilder:validation:Required
	Owner string `json:"owner"`

	// Head is the name of the branch where your changes are implemented
	// +kubebuilder:validation:Required
	Head string `json:"head"`

	// Base is the name of the branch you want the changes pulled into
	// +kubebuilder:validation:Required
	// +kubebuilder:default="master"
	Base string `json:"base"`

	// State of the pull request. Can be "open", "closed", or "merged"
	// +kubebuilder:validation:Enum=open;closed;merged
	// +kubebuilder:default=open
	State *string `json:"state,omitempty"`

	// Assignees is a list of usernames to assign to the pull request
	Assignees []string `json:"assignees,omitempty"`

	// Reviewers is a list of usernames to request review from
	Reviewers []string `json:"reviewers,omitempty"`

	// TeamReviewers is a list of team names to request review from
	TeamReviewers []string `json:"teamReviewers,omitempty"`

	// Labels is a list of labels to add to the pull request
	Labels []string `json:"labels,omitempty"`

	// Milestone is the milestone to associate with the pull request
	Milestone *string `json:"milestone,omitempty"`

	// Draft indicates if the pull request is a draft
	Draft *bool `json:"draft,omitempty"`
}

// PullRequestObservation are the observable fields of a PullRequest.
type PullRequestObservation struct {
	// ID is the unique identifier of the pull request
	ID int64 `json:"id,omitempty"`

	// Number is the pull request number
	Number int64 `json:"number,omitempty"`

	// URL is the HTML URL of the pull request
	URL string `json:"url,omitempty"`

	// State is the current state of the pull request
	State string `json:"state,omitempty"`

	// Author is the username of the pull request author
	Author string `json:"author,omitempty"`

	// CreatedAt is the creation timestamp
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// UpdatedAt is the last update timestamp
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`

	// ClosedAt is the timestamp when the pull request was closed
	ClosedAt *metav1.Time `json:"closedAt,omitempty"`

	// MergedAt is the timestamp when the pull request was merged
	MergedAt *metav1.Time `json:"mergedAt,omitempty"`

	// Mergeable indicates if the pull request can be merged
	Mergeable *bool `json:"mergeable,omitempty"`

	// Merged indicates if the pull request has been merged
	Merged bool `json:"merged,omitempty"`

	// Comments is the number of comments on the pull request
	Comments int `json:"comments,omitempty"`

	// ReviewComments is the number of review comments
	ReviewComments int `json:"reviewComments,omitempty"`

	// Additions is the number of lines added
	Additions int `json:"additions,omitempty"`

	// Deletions is the number of lines deleted
	Deletions int `json:"deletions,omitempty"`

	// ChangedFiles is the number of files changed
	ChangedFiles int `json:"changedFiles,omitempty"`
}

// A PullRequestSpec defines the desired state of a PullRequest.
type PullRequestSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PullRequestParameters `json:"forProvider"`
}

// A PullRequestStatus represents the observed state of a PullRequest.
type PullRequestStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PullRequestObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="REPOSITORY",type="string",JSONPath=".spec.forProvider.repository"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="HEAD",type="string",JSONPath=".spec.forProvider.head"
// +kubebuilder:printcolumn:name="BASE",type="string",JSONPath=".spec.forProvider.base"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// A PullRequest is an API type.
type PullRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PullRequestSpec   `json:"spec"`
	Status PullRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PullRequestList contains a list of PullRequest
type PullRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PullRequest `json:"items"`
}

