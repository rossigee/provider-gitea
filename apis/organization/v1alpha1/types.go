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

// OrganizationParameters define the desired state of a Gitea Organization
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

	// RepoAdminChangeTeamAccess allows repository administrators to change team access
	// +kubebuilder:default=false
	RepoAdminChangeTeamAccess *bool `json:"repoAdminChangeTeamAccess,omitempty"`
}

// OrganizationObservation reflects the observed state of a Gitea Organization
type OrganizationObservation struct {
	// ID is the organization ID
	ID *int64 `json:"id,omitempty"`

	// Email is the organization email
	Email *string `json:"email,omitempty"`

	// AvatarURL is the organization avatar URL
	AvatarURL *string `json:"avatarUrl,omitempty"`
}

// A OrganizationSpec defines the desired state of an Organization.
type OrganizationSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       OrganizationParameters `json:"forProvider"`
}

// A OrganizationStatus represents the observed state of an Organization.
type OrganizationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrganizationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An Organization is a managed resource that represents a Gitea Organization.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea}
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec"`
	Status OrganizationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrganizationList contains a list of Organization
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Organization `json:"items"`
}
