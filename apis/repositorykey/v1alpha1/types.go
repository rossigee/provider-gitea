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

// RepositoryKeyParameters define the desired state of a Gitea Repository SSH Key
type RepositoryKeyParameters struct {
	// Repository is the repository that owns this deploy key (owner/name format)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Repository string `json:"repository"`

	// Title is the display name for the deploy key
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=100
	Title string `json:"title"`

	// Key is the SSH public key content
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^(ssh-rsa|ssh-ed25519|ssh-dss|ecdsa-sha2-nistp256|ecdsa-sha2-nistp384|ecdsa-sha2-nistp521) AAAA[0-9A-Za-z+/]+[=]{0,3}( .*)?$"
	Key string `json:"key"`

	// ReadOnly determines if the key has write access (false) or read-only access (true)
	// +kubebuilder:default=true
	ReadOnly *bool `json:"readOnly,omitempty"`
}

// RepositoryKeyObservation reflects the observed state of a Gitea Repository SSH Key
type RepositoryKeyObservation struct {
	// ID is the unique identifier of the deploy key
	ID *int64 `json:"id,omitempty"`

	// Title is the display name for the deploy key
	Title *string `json:"title,omitempty"`

	// Fingerprint is the SSH key fingerprint
	Fingerprint *string `json:"fingerprint,omitempty"`

	// ReadOnly indicates if the key is read-only
	ReadOnly *bool `json:"readOnly,omitempty"`

	// CreatedAt is the timestamp when the key was created
	CreatedAt *string `json:"createdAt,omitempty"`

	// URL is the API URL for the deploy key
	URL *string `json:"url,omitempty"`

	// Repository is the repository information
	Repository *string `json:"repository,omitempty"`
}

// A RepositoryKeySpec defines the desired state of a RepositoryKey.
type RepositoryKeySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RepositoryKeyParameters `json:"forProvider"`
}

// A RepositoryKeyStatus represents the observed state of a RepositoryKey.
type RepositoryKeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RepositoryKeyObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A RepositoryKey is a managed resource that represents a Gitea Repository SSH Deploy Key.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="REPOSITORY",type="string",JSONPath=".spec.forProvider.repository"
// +kubebuilder:printcolumn:name="TITLE",type="string",JSONPath=".spec.forProvider.title"
// +kubebuilder:printcolumn:name="READ-ONLY",type="boolean",JSONPath=".spec.forProvider.readOnly"
// +kubebuilder:printcolumn:name="FINGERPRINT",type="string",JSONPath=".status.atProvider.fingerprint"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type RepositoryKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositoryKeySpec   `json:"spec"`
	Status RepositoryKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepositoryKeyList contains a list of RepositoryKey
type RepositoryKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RepositoryKey `json:"items"`
}
