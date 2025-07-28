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

// OrganizationSecretParameters define the desired state of a Gitea Organization Secret
type OrganizationSecretParameters struct {
	// Organization is the organization name that owns the secret  
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Organization string `json:"organization"`

	// SecretName is the name of the secret (must follow Gitea naming rules)
	// Secret names can only contain alphanumeric characters ([a-z], [A-Z], [0-9]) or underscores (_)
	// Must not start with GITHUB_ or GITEA_ prefix
	// Must not start with a number
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z_][a-zA-Z0-9_]*$"
	SecretName string `json:"secretName"`

	// Data is the secret value (plaintext, will be encrypted by Gitea)
	// Either Data or DataFrom must be specified, but not both
	// +kubebuilder:validation:Optional
	Data *string `json:"data,omitempty"`

	// DataFrom specifies a reference to get the secret value from a Kubernetes secret
	// Either Data or DataFrom must be specified, but not both
	// +kubebuilder:validation:Optional
	DataFrom *DataFromSource `json:"dataFrom,omitempty"`
}

// DataFromSource represents a source for the secret data
type DataFromSource struct {
	// SecretKeyRef selects a key of a secret in the provider's namespace
	// +kubebuilder:validation:Required
	SecretKeyRef SecretKeySelector `json:"secretKeyRef"`
}

// SecretKeySelector selects a key from a Secret
type SecretKeySelector struct {
	// Name is the name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the secret
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// Key is the key in the secret
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// OrganizationSecretObservation reflects the observed state of a Gitea Organization Secret
type OrganizationSecretObservation struct {
	// CreatedAt is the secret creation timestamp
	CreatedAt *string `json:"createdAt,omitempty"`

	// UpdatedAt is the secret last update timestamp
	UpdatedAt *string `json:"updatedAt,omitempty"`
}

// A OrganizationSecretSpec defines the desired state of an OrganizationSecret.
type OrganizationSecretSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       OrganizationSecretParameters `json:"forProvider"`
}

// A OrganizationSecretStatus represents the observed state of an OrganizationSecret.
type OrganizationSecretStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrganizationSecretObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An OrganizationSecret is a managed resource that represents a Gitea Organization Action Secret.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ORGANIZATION",type="string",JSONPath=".spec.forProvider.organization"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.forProvider.secretName"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea}
type OrganizationSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSecretSpec   `json:"spec"`
	Status OrganizationSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrganizationSecretList contains a list of OrganizationSecret
type OrganizationSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationSecret `json:"items"`
}