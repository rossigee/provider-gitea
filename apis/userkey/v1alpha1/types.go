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

// UserKeyParameters define the desired state of a Gitea User SSH Key
type UserKeyParameters struct {
	// Username is the user that owns this SSH key
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=40
	Username string `json:"username"`

	// Title is the display name for the SSH key
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=100
	Title string `json:"title"`

	// Key is the SSH public key content
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^(ssh-rsa|ssh-ed25519|ssh-dss|ecdsa-sha2-nistp256|ecdsa-sha2-nistp384|ecdsa-sha2-nistp521) AAAA[0-9A-Za-z+/]+[=]{0,3}( .*)?$"
	Key string `json:"key"`
}

// UserKeyObservation reflects the observed state of a Gitea User SSH Key
type UserKeyObservation struct {
	// ID is the unique identifier of the SSH key
	ID *int64 `json:"id,omitempty"`

	// Title is the display name for the SSH key
	Title *string `json:"title,omitempty"`

	// Fingerprint is the SSH key fingerprint
	Fingerprint *string `json:"fingerprint,omitempty"`

	// CreatedAt is the timestamp when the key was created
	CreatedAt *string `json:"createdAt,omitempty"`

	// URL is the API URL for the SSH key
	URL *string `json:"url,omitempty"`

	// Username is the user that owns this SSH key
	Username *string `json:"username,omitempty"`
}

// A UserKeySpec defines the desired state of a UserKey.
type UserKeySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       UserKeyParameters `json:"forProvider"`
}

// A UserKeyStatus represents the observed state of a UserKey.
type UserKeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserKeyObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A UserKey is a managed resource that represents a Gitea User SSH Key.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="USERNAME",type="string",JSONPath=".spec.forProvider.username"
// +kubebuilder:printcolumn:name="TITLE",type="string",JSONPath=".spec.forProvider.title"
// +kubebuilder:printcolumn:name="FINGERPRINT",type="string",JSONPath=".status.atProvider.fingerprint"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type UserKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserKeySpec   `json:"spec"`
	Status UserKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserKeyList contains a list of UserKey
type UserKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserKey `json:"items"`
}