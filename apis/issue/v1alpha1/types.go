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

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// IssueParameters define the desired state of a Gitea Issue
type IssueParameters struct {
	// Title is the title of the issue
	// +kubebuilder:validation:Required
	Title string `json:"title"`

	// Body is the body content of the issue
	Body *string `json:"body,omitempty"`

	// Repository is the name of the repository
	// +kubebuilder:validation:Required
	Repository string `json:"repository"`

	// Owner is the username or organization name that owns the repository
	// +kubebuilder:validation:Required
	Owner string `json:"owner"`

	// State of the issue. Can be "open" or "closed"
	// +kubebuilder:validation:Enum=open;closed
	// +kubebuilder:default=open
	State *string `json:"state,omitempty"`

	// Assignees is a list of usernames to assign to the issue
	Assignees []string `json:"assignees,omitempty"`

	// Labels is a list of labels to add to the issue
	Labels []string `json:"labels,omitempty"`

	// Milestone is the milestone to associate with the issue
	Milestone *string `json:"milestone,omitempty"`
}

// IssueObservation represents the observed state of a Gitea Issue
type IssueObservation struct {
	// ID is the unique identifier of the issue
	ID int64 `json:"id,omitempty"`

	// Number is the issue number
	Number int64 `json:"number,omitempty"`

	// URL is the web URL of the issue
	URL string `json:"url,omitempty"`

	// State is the current state of the issue
	State string `json:"state,omitempty"`

	// CreatedAt is the timestamp when the issue was created
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// UpdatedAt is the timestamp when the issue was last updated
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`

	// ClosedAt is the timestamp when the issue was closed
	ClosedAt *metav1.Time `json:"closedAt,omitempty"`

	// Comments is the number of comments on the issue
	Comments int `json:"comments,omitempty"`

	// Author is the username of the issue creator
	Author string `json:"author,omitempty"`
}

// An IssueSpec defines the desired state of an Issue.
type IssueSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       IssueParameters `json:"forProvider"`
}

// An IssueStatus represents the observed state of an Issue.
type IssueStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          IssueObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An Issue represents a Gitea issue.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="REPOSITORY",type="string",JSONPath=".spec.forProvider.repository"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type Issue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IssueSpec   `json:"spec"`
	Status IssueStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IssueList contains a list of Issues
type IssueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Issue `json:"items"`
}