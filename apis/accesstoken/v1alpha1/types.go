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

// AccessTokenParameters define the desired state of a Gitea API Access Token
type AccessTokenParameters struct {
	// Username is the user that owns this access token
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=40
	Username string `json:"username"`

	// Name is the display name for the access token
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=100
	Name string `json:"name"`

	// Scopes defines the permissions granted to this token
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:UniqueItems=true
	Scopes []string `json:"scopes"`
}

// AccessTokenObservation reflects the observed state of a Gitea API Access Token
type AccessTokenObservation struct {
	// ID is the unique identifier of the access token
	ID *int64 `json:"id,omitempty"`

	// Name is the display name for the access token
	Name *string `json:"name,omitempty"`

	// Scopes lists the permissions granted to this token
	Scopes []string `json:"scopes,omitempty"`

	// TokenLastEight shows the last 8 characters of the token (for identification)
	TokenLastEight *string `json:"tokenLastEight,omitempty"`

	// CreatedAt is the timestamp when the token was created
	CreatedAt *string `json:"createdAt,omitempty"`

	// LastUsedAt is the timestamp when the token was last used
	LastUsedAt *string `json:"lastUsedAt,omitempty"`

	// Username is the user that owns this access token
	Username *string `json:"username,omitempty"`
}

// A AccessTokenSpec defines the desired state of a AccessToken.
type AccessTokenSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       AccessTokenParameters `json:"forProvider"`
}

// A AccessTokenStatus represents the observed state of a AccessToken.
type AccessTokenStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          AccessTokenObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A AccessToken is a managed resource that represents a Gitea API Access Token.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="USERNAME",type="string",JSONPath=".spec.forProvider.username"
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="SCOPES",type="string",JSONPath=".spec.forProvider.scopes"
// +kubebuilder:printcolumn:name="LAST-EIGHT",type="string",JSONPath=".status.atProvider.tokenLastEight"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type AccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessTokenSpec   `json:"spec"`
	Status AccessTokenStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AccessTokenList contains a list of AccessToken
type AccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AccessToken `json:"items"`
}
