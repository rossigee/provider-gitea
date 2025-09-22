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

type OrganizationMemberParameters struct {
	// Organization is the organization name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=40
	Organization string `json:"organization"`

	// Username is the username of the member
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=40
	Username string `json:"username"`

	// Role defines the member's role in the organization
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=owner;admin;member
	// +kubebuilder:default=member
	Role string `json:"role"`

	// Visibility defines if the membership is public or private
	// +kubebuilder:validation:Enum=public;private
	// +kubebuilder:default=private
	Visibility *string `json:"visibility,omitempty"`

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

type OrganizationMemberObservation struct {
	// Username is the username of the member
	Username *string `json:"username,omitempty"`

	// Role is the member's role in the organization
	Role *string `json:"role,omitempty"`

	// Visibility indicates if the membership is public or private
	Visibility *string `json:"visibility,omitempty"`

	// JoinedAt is the timestamp when the user joined the organization
	JoinedAt *string `json:"joinedAt,omitempty"`

	// Organization is the organization name
	Organization *string `json:"organization,omitempty"`

	// UserInfo contains additional user information
	UserInfo *OrganizationMemberUserInfo `json:"userInfo,omitempty"`

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

// OrganizationMemberSpec defines the desired state of OrganizationMember
type OrganizationMemberSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       OrganizationMemberParameters `json:"forProvider"`
}

// OrganizationMemberStatus defines the observed state of OrganizationMember
type OrganizationMemberStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrganizationMemberObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=orga
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// OrganizationMember is the Schema for the organizationmembers API v2 (namespaced)
type OrganizationMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationMemberSpec   `json:"spec,omitempty"`
	Status OrganizationMemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrganizationMemberList contains a list of OrganizationMember
type OrganizationMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationMember `json:"items"`
}

// OrganizationMember type metadata
var (
	OrganizationMemberKind             = "OrganizationMember"
	OrganizationMemberGroupKind        = schema.GroupKind{Group: Group, Kind: OrganizationMemberKind}
	OrganizationMemberKindAPIVersion   = OrganizationMemberKind + "." + SchemeGroupVersion.String()
	OrganizationMemberGroupVersionKind = SchemeGroupVersion.WithKind(OrganizationMemberKind)
)

func init() {
	SchemeBuilder.Register(&OrganizationMember{}, &OrganizationMemberList{})
}
