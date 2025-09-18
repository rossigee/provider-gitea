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

// OrganizationMemberParameters define the desired state of a Gitea Organization Member
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
}

// OrganizationMemberObservation reflects the observed state of a Gitea Organization Member
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
}

// OrganizationMemberUserInfo contains basic user information
type OrganizationMemberUserInfo struct {
	// ID is the user's unique identifier
	ID *int64 `json:"id,omitempty"`

	// Email is the user's email address
	Email *string `json:"email,omitempty"`

	// FullName is the user's full name
	FullName *string `json:"fullName,omitempty"`

	// AvatarURL is the URL to the user's avatar
	AvatarURL *string `json:"avatarUrl,omitempty"`
}

// A OrganizationMemberSpec defines the desired state of a OrganizationMember.
type OrganizationMemberSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       OrganizationMemberParameters `json:"forProvider"`
}

// A OrganizationMemberStatus represents the observed state of a OrganizationMember.
type OrganizationMemberStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrganizationMemberObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A OrganizationMember is a managed resource that represents a Gitea Organization Member.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="ORGANIZATION",type="string",JSONPath=".spec.forProvider.organization"
// +kubebuilder:printcolumn:name="USERNAME",type="string",JSONPath=".spec.forProvider.username"
// +kubebuilder:printcolumn:name="ROLE",type="string",JSONPath=".spec.forProvider.role"
// +kubebuilder:printcolumn:name="VISIBILITY",type="string",JSONPath=".spec.forProvider.visibility"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type OrganizationMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationMemberSpec   `json:"spec"`
	Status OrganizationMemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrganizationMemberList contains a list of OrganizationMember
type OrganizationMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationMember `json:"items"`
}
