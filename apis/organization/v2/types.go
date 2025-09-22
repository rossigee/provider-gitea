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

// OrganizationParameters define the desired state of a Gitea Organization v2
type OrganizationParameters struct {
	// Username is the organization username
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Username string `json:"username"`

	// Name is the display name of the organization
	Name *string `json:"name,omitempty"`

	// FullName is the full name of the organization
	FullName *string `json:"fullName,omitempty"`

	// Description is the organization description
	Description *string `json:"description,omitempty"`

	// Website is the organization website URL
	Website *string `json:"website,omitempty"`

	// Location is the organization location
	Location *string `json:"location,omitempty"`

	// Visibility specifies the organization visibility
	// +kubebuilder:validation:Enum=public;limited;private
	// +kubebuilder:default="public"
	Visibility *string `json:"visibility,omitempty"`

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

// OrganizationObservation reflects the observed state of a Gitea Organization
type OrganizationObservation struct {
	// ID is the unique identifier of the organization
	ID *int64 `json:"id,omitempty"`

	// AvatarURL is the URL of the organization avatar
	AvatarURL *string `json:"avatarUrl,omitempty"`

	// Email is the organization email
	Email *string `json:"email,omitempty"`

	// RepoAdminChangeTeamAccess determines if repository admins can change team access
	RepoAdminChangeTeamAccess *bool `json:"repoAdminChangeTeamAccess,omitempty"`

	// CreatedAt is the creation timestamp
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// UpdatedAt is the last update timestamp
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`

	// V2 Enhancement: Enhanced observability
	// PublicRepos is the number of public repositories
	PublicRepos *int64 `json:"publicRepos,omitempty"`

	// PrivateRepos is the number of private repositories
	PrivateRepos *int64 `json:"privateRepos,omitempty"`

	// Members is the number of organization members
	Members *int64 `json:"members,omitempty"`

	// Teams is the number of teams in the organization
	Teams *int64 `json:"teams,omitempty"`
}

// OrganizationSpec defines the desired state of Organization
type OrganizationSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       OrganizationParameters `json:"forProvider"`
}

// OrganizationStatus defines the observed state of Organization
type OrganizationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrganizationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=gorg
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// Organization is the Schema for the organizations API v2 (namespaced)
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec,omitempty"`
	Status OrganizationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrganizationList contains a list of Organization
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Organization `json:"items"`
}

// Organization type metadata
var (
	OrganizationKind             = "Organization"
	OrganizationGroupKind        = schema.GroupKind{Group: Group, Kind: OrganizationKind}
	OrganizationKindAPIVersion   = OrganizationKind + "." + SchemeGroupVersion.String()
	OrganizationGroupVersionKind = SchemeGroupVersion.WithKind(OrganizationKind)
)

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}