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

// AppliedOrganizationSettings contains the currently applied organization settings
type AppliedOrganizationSettings struct {
	// Description is the applied organization description
	Description *string `json:"description,omitempty"`

	// Website is the applied organization website
	Website *string `json:"website,omitempty"`

	// Location is the applied organization location
	Location *string `json:"location,omitempty"`
}

type OrganizationSettingsParameters struct {
	// Organization is the organization name to configure
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Organization string `json:"organization"`

	// DefaultRepoPermission is the default permission for organization members on new repositories
	// +kubebuilder:validation:Enum=read;write;admin
	// +kubebuilder:default="read"
	DefaultRepoPermission *string `json:"defaultRepoPermission,omitempty"`

	// MembersCanCreateRepos controls whether organization members can create repositories
	// +kubebuilder:default=true
	MembersCanCreateRepos *bool `json:"membersCanCreateRepos,omitempty"`

	// MembersCanCreatePrivate controls whether organization members can create private repositories
	// +kubebuilder:default=true
	MembersCanCreatePrivate *bool `json:"membersCanCreatePrivate,omitempty"`

	// MembersCanCreateInternal controls whether organization members can create internal repositories
	// +kubebuilder:default=true
	MembersCanCreateInternal *bool `json:"membersCanCreateInternal,omitempty"`

	// MembersCanDeleteRepos controls whether organization members can delete repositories
	// +kubebuilder:default=false
	MembersCanDeleteRepos *bool `json:"membersCanDeleteRepos,omitempty"`

	// MembersCanFork controls whether organization members can fork repositories
	// +kubebuilder:default=true
	MembersCanFork *bool `json:"membersCanFork,omitempty"`

	// MembersCanCreatePages controls whether organization members can create GitHub Pages
	// +kubebuilder:default=true
	MembersCanCreatePages *bool `json:"membersCanCreatePages,omitempty"`

	// DefaultRepoVisibility is the default visibility for new repositories
	// +kubebuilder:validation:Enum=public;private;internal
	// +kubebuilder:default="public"
	DefaultRepoVisibility *string `json:"defaultRepoVisibility,omitempty"`

	// RequireSignedCommits controls whether signed commits are required
	// +kubebuilder:default=false
	RequireSignedCommits *bool `json:"requireSignedCommits,omitempty"`

	// EnableDependencyGraph controls whether dependency graph is enabled
	// +kubebuilder:default=false
	EnableDependencyGraph *bool `json:"enableDependencyGraph,omitempty"`

	// AllowGitHooks controls whether Git hooks are allowed in organization repositories
	// +kubebuilder:default=false
	AllowGitHooks *bool `json:"allowGitHooks,omitempty"`

	// AllowCustomGitHooks controls whether custom Git hooks are allowed
	// +kubebuilder:default=false
	AllowCustomGitHooks *bool `json:"allowCustomGitHooks,omitempty"`

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

type OrganizationSettingsObservation struct {
	// OrganizationID is the organization ID
	OrganizationID *int64 `json:"organizationId,omitempty"`

	// LastUpdated timestamp
	LastUpdated *string `json:"lastUpdated,omitempty"`

	// Applied settings (current state)
	AppliedSettings *AppliedOrganizationSettings `json:"appliedSettings,omitempty"`

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

// OrganizationSettingsSpec defines the desired state of OrganizationSettings
type OrganizationSettingsSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       OrganizationSettingsParameters `json:"forProvider"`
}

// OrganizationSettingsStatus defines the observed state of OrganizationSettings
type OrganizationSettingsStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrganizationSettingsObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=orga
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// OrganizationSettings is the Schema for the organizationsettingss API v2 (namespaced)
type OrganizationSettings struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSettingsSpec   `json:"spec,omitempty"`
	Status OrganizationSettingsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrganizationSettingsList contains a list of OrganizationSettings
type OrganizationSettingsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationSettings `json:"items"`
}

// OrganizationSettings type metadata
var (
	OrganizationSettingsKind             = "OrganizationSettings"
	OrganizationSettingsGroupKind        = schema.GroupKind{Group: Group, Kind: OrganizationSettingsKind}
	OrganizationSettingsKindAPIVersion   = OrganizationSettingsKind + "." + SchemeGroupVersion.String()
	OrganizationSettingsGroupVersionKind = SchemeGroupVersion.WithKind(OrganizationSettingsKind)
)

func init() {
	SchemeBuilder.Register(&OrganizationSettings{}, &OrganizationSettingsList{})
}
