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

// DeployKeyParameters define the desired state of a Gitea Deploy Key
type DeployKeyParameters struct {
	// Repository is the repository name
	// +kubebuilder:validation:Required
	Repository string `json:"repository"`

	// Owner is the owner/organization of the repository
	// +kubebuilder:validation:Required
	Owner string `json:"owner"`

	// Title is the deploy key title
	// +kubebuilder:validation:Required
	Title string `json:"title"`

	// Key is the SSH public key
	// +kubebuilder:validation:Required
	Key string `json:"key"`

	// ReadOnly determines if the key has read-only access
	// +kubebuilder:default=true
	ReadOnly *bool `json:"readOnly,omitempty"`
}

// DeployKeyObservation reflects the observed state of a Gitea Deploy Key
type DeployKeyObservation struct {
	// ID is the deploy key ID
	ID *int64 `json:"id,omitempty"`

	// URL is the deploy key URL
	URL *string `json:"url,omitempty"`

	// Fingerprint is the key fingerprint
	Fingerprint *string `json:"fingerprint,omitempty"`

	// CreatedAt is the creation timestamp
	CreatedAt *string `json:"createdAt,omitempty"`
}

// A DeployKeySpec defines the desired state of a DeployKey.
type DeployKeySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       DeployKeyParameters `json:"forProvider"`
}

// A DeployKeyStatus represents the observed state of a DeployKey.
type DeployKeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DeployKeyObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A DeployKey is a managed resource that represents a Gitea Deploy Key.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type DeployKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeployKeySpec   `json:"spec"`
	Status DeployKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DeployKeyList contains a list of DeployKey
type DeployKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeployKey `json:"items"`
}
