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

// AdminUserParameters define the desired state of a Gitea Administrative User
type AdminUserParameters struct {
	// Username is the user's login name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=40
	Username string `json:"username"`

	// Email is the user's email address
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=email
	Email string `json:"email"`

	// PasswordSecretRef references a Kubernetes secret containing the user's password
	// +kubebuilder:validation:Required
	PasswordSecretRef xpv1.SecretKeySelector `json:"passwordSecretRef"`

	// FullName is the user's display name
	// +kubebuilder:validation:MaxLength=100
	FullName *string `json:"fullName,omitempty"`

	// IsAdmin indicates if the user should have admin privileges
	// +kubebuilder:default=false
	IsAdmin *bool `json:"isAdmin,omitempty"`

	// MustChangePassword forces the user to change password on first login
	// +kubebuilder:default=false
	MustChangePassword *bool `json:"mustChangePassword,omitempty"`

	// SendNotify sends a notification email to the user
	// +kubebuilder:default=false
	SendNotify *bool `json:"sendNotify,omitempty"`

	// Visibility controls the user's profile visibility
	// +kubebuilder:validation:Enum=public;private;limited
	// +kubebuilder:default=public
	Visibility *string `json:"visibility,omitempty"`

	// IsActive indicates if the user account is active
	// +kubebuilder:default=true
	IsActive *bool `json:"isActive,omitempty"`

	// IsRestricted indicates if the user has restricted access
	// +kubebuilder:default=false
	IsRestricted *bool `json:"isRestricted,omitempty"`

	// MaxRepoCreation limits the number of repositories the user can create (-1 for unlimited)
	// +kubebuilder:validation:Minimum=-1
	// +kubebuilder:default=-1
	MaxRepoCreation *int `json:"maxRepoCreation,omitempty"`

	// ProhibitLogin prevents the user from logging in (useful for bot accounts)
	// +kubebuilder:default=false
	ProhibitLogin *bool `json:"prohibitLogin,omitempty"`

	// Website is the user's website URL
	// +kubebuilder:validation:Format=uri
	Website *string `json:"website,omitempty"`

	// Location is the user's location
	// +kubebuilder:validation:MaxLength=50
	Location *string `json:"location,omitempty"`

	// Description is the user's bio/description
	// +kubebuilder:validation:MaxLength=255
	Description *string `json:"description,omitempty"`
}

// AdminUserObservation reflects the observed state of a Gitea Administrative User
type AdminUserObservation struct {
	// ID is the unique identifier of the user
	ID *int64 `json:"id,omitempty"`

	// Username is the user's login name
	Username *string `json:"username,omitempty"`

	// Email is the user's email address
	Email *string `json:"email,omitempty"`

	// FullName is the user's display name
	FullName *string `json:"fullName,omitempty"`

	// AvatarURL is the URL to the user's avatar
	AvatarURL *string `json:"avatarUrl,omitempty"`

	// IsAdmin indicates if the user has admin privileges
	IsAdmin *bool `json:"isAdmin,omitempty"`

	// IsActive indicates if the user account is active
	IsActive *bool `json:"isActive,omitempty"`

	// IsRestricted indicates if the user has restricted access
	IsRestricted *bool `json:"isRestricted,omitempty"`

	// ProhibitLogin indicates if the user is prevented from logging in
	ProhibitLogin *bool `json:"prohibitLogin,omitempty"`

	// Visibility shows the user's profile visibility
	Visibility *string `json:"visibility,omitempty"`

	// CreatedAt is the timestamp when the user was created
	CreatedAt *string `json:"createdAt,omitempty"`

	// LastLogin is the timestamp of the user's last login
	LastLogin *string `json:"lastLogin,omitempty"`

	// Language is the user's preferred language
	Language *string `json:"language,omitempty"`

	// MaxRepoCreation is the user's repository creation limit
	MaxRepoCreation *int `json:"maxRepoCreation,omitempty"`

	// Website is the user's website URL
	Website *string `json:"website,omitempty"`

	// Location is the user's location
	Location *string `json:"location,omitempty"`

	// Description is the user's bio/description
	Description *string `json:"description,omitempty"`

	// UserStats contains user statistics
	UserStats *AdminUserStats `json:"userStats,omitempty"`
}

// AdminUserStats contains user statistics
type AdminUserStats struct {
	// Repositories is the number of repositories owned
	Repositories *int `json:"repositories,omitempty"`

	// PublicRepos is the number of public repositories
	PublicRepos *int `json:"publicRepos,omitempty"`

	// Followers is the number of followers
	Followers *int `json:"followers,omitempty"`

	// Following is the number of users being followed
	Following *int `json:"following,omitempty"`

	// StarredRepos is the number of starred repositories
	StarredRepos *int `json:"starredRepos,omitempty"`
}

// A AdminUserSpec defines the desired state of a AdminUser.
type AdminUserSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       AdminUserParameters `json:"forProvider"`
}

// A AdminUserStatus represents the observed state of a AdminUser.
type AdminUserStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          AdminUserObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A AdminUser is a managed resource that represents a Gitea Administrative User.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="USERNAME",type="string",JSONPath=".spec.forProvider.username"
// +kubebuilder:printcolumn:name="EMAIL",type="string",JSONPath=".spec.forProvider.email"
// +kubebuilder:printcolumn:name="IS-ADMIN",type="boolean",JSONPath=".spec.forProvider.isAdmin"
// +kubebuilder:printcolumn:name="IS-ACTIVE",type="boolean",JSONPath=".status.atProvider.isActive"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type AdminUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AdminUserSpec   `json:"spec"`
	Status AdminUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AdminUserList contains a list of AdminUser
type AdminUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AdminUser `json:"items"`
}
