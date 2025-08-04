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

// RepositorySecretParameters define the desired state of a Gitea Repository Actions Secret
type RepositorySecretParameters struct {
	// Repository is the repository that owns this secret (owner/name format)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Repository string `json:"repository"`

	// SecretName is the name of the secret
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[A-Z_][A-Z0-9_]*$"
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=100
	SecretName string `json:"secretName"`

	// ValueSecretRef references a Kubernetes secret containing the secret value
	// +kubebuilder:validation:Required
	ValueSecretRef xpv1.SecretKeySelector `json:"valueSecretRef"`
}

// RepositorySecretObservation reflects the observed state of a Gitea Repository Actions Secret
type RepositorySecretObservation struct {
	// SecretName is the name of the secret
	SecretName *string `json:"secretName,omitempty"`

	// CreatedAt is the timestamp when the secret was created
	CreatedAt *string `json:"createdAt,omitempty"`

	// UpdatedAt is the timestamp when the secret was last updated
	UpdatedAt *string `json:"updatedAt,omitempty"`

	// Repository is the repository information
	Repository *string `json:"repository,omitempty"`
}

// A RepositorySecretSpec defines the desired state of a RepositorySecret.
type RepositorySecretSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RepositorySecretParameters `json:"forProvider"`
}

// A RepositorySecretStatus represents the observed state of a RepositorySecret.
type RepositorySecretStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RepositorySecretObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A RepositorySecret is a managed resource that represents a Gitea Repository Actions Secret.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="REPOSITORY",type="string",JSONPath=".spec.forProvider.repository"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.forProvider.secretName"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type RepositorySecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySecretSpec   `json:"spec"`
	Status RepositorySecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepositorySecretList contains a list of RepositorySecret
type RepositorySecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RepositorySecret `json:"items"`
}
