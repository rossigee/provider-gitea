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

type UserParameters struct {
	// Username is the user's username
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Username string `json:"username"`

	// Email is the user's email address
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format="email"
	Email string `json:"email"`

	// Password is the user's password (required for creation)
	Password string `json:"password"`

	// FullName is the user's full name
	FullName *string `json:"fullName,omitempty"`

	// LoginName is the user's login name (if different from username)
	LoginName *string `json:"loginName,omitempty"`

	// SendNotify determines if a notification email should be sent
	// +kubebuilder:default=false
	SendNotify *bool `json:"sendNotify,omitempty"`

	// SourceID is the authentication source ID
	SourceID *int64 `json:"sourceId,omitempty"`

	// MustChangePassword forces the user to change password on first login
	// +kubebuilder:default=false
	MustChangePassword *bool `json:"mustChangePassword,omitempty"`

	// Restricted determines if the user has restricted access
	// +kubebuilder:default=false
	Restricted *bool `json:"restricted,omitempty"`

	// Visibility specifies the user's visibility
	// +kubebuilder:validation:Enum=public;limited;private
	// +kubebuilder:default="public"
	Visibility *string `json:"visibility,omitempty"`

	// Website is the user's website URL
	Website *string `json:"website,omitempty"`

	// Location is the user's location
	Location *string `json:"location,omitempty"`

	// Description is the user's description/bio
	Description *string `json:"description,omitempty"`

	// Admin determines if the user has admin privileges (admin only)
	Admin *bool `json:"admin,omitempty"`

	// Active determines if the user account is active (admin only)
	Active *bool `json:"active,omitempty"`

	// ProhibitLogin prevents the user from logging in (admin only)
	ProhibitLogin *bool `json:"prohibitLogin,omitempty"`

	// AllowGitHook allows the user to use git hooks (admin only)
	AllowGitHook *bool `json:"allowGitHook,omitempty"`

	// AllowImportLocal allows the user to import local repositories (admin only)
	AllowImportLocal *bool `json:"allowImportLocal,omitempty"`

	// AllowCreateOrganization allows the user to create organizations (admin only)
	AllowCreateOrganization *bool `json:"allowCreateOrganization,omitempty"`

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

type UserObservation struct {
	// ID is the user ID
	ID *int64 `json:"id,omitempty"`

	// AvatarURL is the user's avatar URL
	AvatarURL *string `json:"avatarUrl,omitempty"`

	// IsAdmin indicates if the user has admin privileges
	IsAdmin *bool `json:"isAdmin,omitempty"`

	// LastLogin is the user's last login timestamp
	LastLogin *string `json:"lastLogin,omitempty"`

	// Created is the user creation timestamp
	Created *string `json:"created,omitempty"`

	// Language is the user's preferred language
	Language *string `json:"language,omitempty"`

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

// UserSpec defines the desired state of User
type UserSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       UserParameters `json:"forProvider"`
}

// UserStatus defines the observed state of User
type UserStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=user
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// User is the Schema for the users API v2 (namespaced)
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

// User type metadata
var (
	UserKind             = "User"
	UserGroupKind        = schema.GroupKind{Group: Group, Kind: UserKind}
	UserKindAPIVersion   = UserKind + "." + SchemeGroupVersion.String()
	UserGroupVersionKind = SchemeGroupVersion.WithKind(UserKind)
)

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
