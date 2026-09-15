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
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeployTokenParameters struct {
	// Repository is the repository name
	// +kubebuilder:validation:Required
	Repository string `json:"repository"`

	// Owner is the owner/organization of the repository
	// +kubebuilder:validation:Required
	Owner string `json:"owner"`

	// Title is the deploy token title (max 50 characters)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=50
	Title string `json:"title"`

	// ReadOnly determines if the token has read-only access
	// +kubebuilder:default=true
	ReadOnly *bool `json:"readOnly,omitempty"`

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

type DeployTokenObservation struct {
	// ID is the deploy token ID
	ID *int64 `json:"id,omitempty"`

	// URL is the deploy token URL
	URL *string `json:"url,omitempty"`

	// Fingerprint is the token fingerprint (masked preview, e.g. gdt_xx********xx)
	Fingerprint *string `json:"fingerprint,omitempty"`

	// CreatedAt is the creation timestamp
	CreatedAt *string `json:"createdAt,omitempty"`
}

// DeployTokenSpec defines the desired state of DeployToken
type DeployTokenSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              DeployTokenParameters `json:"forProvider"`
}

// DeployTokenStatus defines the observed state of DeployToken
type DeployTokenStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`
	AtProvider                 DeployTokenObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=dtok
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// DeployToken is the Schema for the deploytokens API v2 (namespaced)
type DeployToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeployTokenSpec   `json:"spec,omitempty"`
	Status DeployTokenStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DeployTokenList contains a list of DeployToken
type DeployTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeployToken `json:"items"`
}

// GetCondition returns the condition for the given ConditionType if it exists, otherwise returns nil.
func (r *DeployToken) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return r.Status.GetCondition(ct)
}

// SetConditions sets the supplied conditions, replacing any existing conditions of the same type.
func (r *DeployToken) SetConditions(c ...xpv1.Condition) {
	r.Status.SetConditions(c...)
}

// GetManagementPolicies returns the management policies for this resource.
func (r *DeployToken) GetManagementPolicies() xpv1.ManagementPolicies {
	return r.Spec.ManagementPolicies
}

// SetManagementPolicies sets the management policies for this resource.
func (r *DeployToken) SetManagementPolicies(p xpv1.ManagementPolicies) {
	r.Spec.ManagementPolicies = p
}

// GetProviderConfigReference of this DeployToken.
func (r *DeployToken) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return r.Spec.ProviderConfigReference
}

// SetProviderConfigReference of this DeployToken.
func (r *DeployToken) SetProviderConfigReference(p *xpv1.ProviderConfigReference) {
	r.Spec.ProviderConfigReference = p
}

// GetWriteConnectionSecretToReference of this DeployToken.
func (r *DeployToken) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return r.Spec.WriteConnectionSecretToReference
}

// SetWriteConnectionSecretToReference of this DeployToken.
func (r *DeployToken) SetWriteConnectionSecretToReference(p *xpv1.LocalSecretReference) {
	r.Spec.WriteConnectionSecretToReference = p
}
