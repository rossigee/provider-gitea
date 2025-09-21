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

// RepositoryParameters define the desired state of a Gitea Repository v2
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

	// Archived determines if the repository is archived
	// +kubebuilder:default=false
	Archived *bool `json:"archived,omitempty"`

	// DefaultBranch is the default branch name
	// +kubebuilder:default="master"
	DefaultBranch *string `json:"defaultBranch,omitempty"`

	// TrustModel sets the trust model for the repository
	// +kubebuilder:validation:Enum=default;collaborator;committer;collaboratorcommitter
	// +kubebuilder:default="default"
	TrustModel *string `json:"trustModel,omitempty"`

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

// RepositoryObservation reflects the observed state of a Gitea Repository
type RepositoryObservation struct {
	// ID is the unique identifier of the repository
	ID *int64 `json:"id,omitempty"`

	// FullName is the full name of the repository including owner
	FullName *string `json:"fullName,omitempty"`

	// HTMLURL is the web URL of the repository
	HTMLURL *string `json:"htmlUrl,omitempty"`

	// SSHURL is the SSH URL for cloning
	SSHURL *string `json:"sshUrl,omitempty"`

	// CloneURL is the HTTPS URL for cloning
	CloneURL *string `json:"cloneUrl,omitempty"`

	// CreatedAt is the creation timestamp
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// UpdatedAt is the last update timestamp
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`

	// V2 Enhancement: Enhanced observability
	// Stars is the number of stars
	Stars *int64 `json:"stars,omitempty"`

	// Forks is the number of forks
	Forks *int64 `json:"forks,omitempty"`

	// Size is the repository size in KB
	Size *int64 `json:"size,omitempty"`

	// Language is the primary programming language
	Language *string `json:"language,omitempty"`
}

// RepositorySpec defines the desired state of Repository
type RepositorySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RepositoryParameters `json:"forProvider"`
}

// RepositoryStatus defines the observed state of Repository
type RepositoryStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RepositoryObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=grepo
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// Repository is the Schema for the repositories API v2 (namespaced)
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepositoryList contains a list of Repository
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}

// Repository type metadata
var (
	RepositoryKind             = "Repository"
	RepositoryGroupKind        = schema.GroupKind{Group: Group, Kind: RepositoryKind}
	RepositoryKindAPIVersion   = RepositoryKind + "." + SchemeGroupVersion.String()
	RepositoryGroupVersionKind = SchemeGroupVersion.WithKind(RepositoryKind)
)

func init() {
	SchemeBuilder.Register(&Repository{}, &RepositoryList{})
}