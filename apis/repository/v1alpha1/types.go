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

// RepositoryParameters define the desired state of a Gitea Repository
type RepositoryParameters struct {
	// Name is the repository name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Name string `json:"name"`

	// Owner is the username or organization name that owns the repository
	// If not specified, the repository will be created under the authenticated user
	Owner *string `json:"owner,omitempty"`

	// Description is the repository description
	Description *string `json:"description,omitempty"`

	// Private determines if the repository is private
	// +kubebuilder:default=false
	Private *bool `json:"private,omitempty"`

	// AutoInit determines if the repository should be initialized with a README
	// +kubebuilder:default=false
	AutoInit *bool `json:"autoInit,omitempty"`

	// Template determines if the repository is a template repository
	// +kubebuilder:default=false
	Template *bool `json:"template,omitempty"`

	// Gitignores specifies the gitignore template to use
	Gitignores *string `json:"gitignores,omitempty"`

	// License specifies the license template to use
	License *string `json:"license,omitempty"`

	// Readme specifies the README template to use
	Readme *string `json:"readme,omitempty"`

	// IssueLabels specifies the issue label set to use
	IssueLabels *string `json:"issueLabels,omitempty"`

	// TrustModel specifies the trust model for the repository
	// +kubebuilder:validation:Enum=default;collaborator;committer;collaboratorcommitter
	// +kubebuilder:default="default"
	TrustModel *string `json:"trustModel,omitempty"`

	// DefaultBranch specifies the default branch name
	// +kubebuilder:default="master"
	DefaultBranch *string `json:"defaultBranch,omitempty"`

	// Website is the repository website URL
	Website *string `json:"website,omitempty"`

	// Settings for repository features
	HasIssues *bool `json:"hasIssues,omitempty"`
	HasWiki   *bool `json:"hasWiki,omitempty"`
	HasPullRequests *bool `json:"hasPullRequests,omitempty"`
	HasProjects *bool `json:"hasProjects,omitempty"`
	HasReleases *bool `json:"hasReleases,omitempty"`
	HasPackages *bool `json:"hasPackages,omitempty"`
	HasActions  *bool `json:"hasActions,omitempty"`

	// Merge settings
	AllowMergeCommits     *bool `json:"allowMergeCommits,omitempty"`
	AllowRebase           *bool `json:"allowRebase,omitempty"`
	AllowRebaseExplicit   *bool `json:"allowRebaseExplicit,omitempty"`
	AllowSquashMerge      *bool `json:"allowSquashMerge,omitempty"`
	AllowRebaseUpdate     *bool `json:"allowRebaseUpdate,omitempty"`
	DefaultDeleteBranchAfterMerge *bool `json:"defaultDeleteBranchAfterMerge,omitempty"`

	// DefaultMergeStyle specifies the default merge style
	// +kubebuilder:validation:Enum=merge;rebase;squash;rebase-merge
	DefaultMergeStyle *string `json:"defaultMergeStyle,omitempty"`

	// Archived determines if the repository is archived
	// +kubebuilder:default=false
	Archived *bool `json:"archived,omitempty"`
}

// RepositoryObservation reflects the observed state of a Gitea Repository
type RepositoryObservation struct {
	// ID is the repository ID
	ID *int64 `json:"id,omitempty"`

	// FullName is the full name (owner/name) of the repository
	FullName *string `json:"fullName,omitempty"`

	// Fork indicates if this repository is a fork
	Fork *bool `json:"fork,omitempty"`

	// Empty indicates if the repository is empty
	Empty *bool `json:"empty,omitempty"`

	// Size is the repository size in KB
	Size *int `json:"size,omitempty"`

	// HTMLURL is the repository HTML URL
	HTMLURL *string `json:"htmlUrl,omitempty"`

	// SSHURL is the repository SSH URL
	SSHURL *string `json:"sshUrl,omitempty"`

	// CloneURL is the repository clone URL
	CloneURL *string `json:"cloneUrl,omitempty"`

	// Language is the primary programming language
	Language *string `json:"language,omitempty"`

	// CreatedAt is the repository creation timestamp
	CreatedAt *string `json:"createdAt,omitempty"`

	// UpdatedAt is the repository last update timestamp
	UpdatedAt *string `json:"updatedAt,omitempty"`
}

// A RepositorySpec defines the desired state of a Repository.
type RepositorySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RepositoryParameters `json:"forProvider"`
}

// A RepositoryStatus represents the observed state of a Repository.
type RepositoryStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RepositoryObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Repository is a managed resource that represents a Gitea Repository.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec"`
	Status RepositoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepositoryList contains a list of Repository
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}