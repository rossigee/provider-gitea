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

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

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

	// ValueSecretRef references the key in a Kubernetes Secret holding the secret
	// value (encrypted by Gitea on write). The value is never set inline; it is
	// always taken from a referenced Secret (secret-ref convention).
	// +kubebuilder:validation:Required
	ValueSecretRef *xpv1.SecretKeySelector `json:"valueSecretRef,omitempty"`

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

type OrganizationSecretObservation struct {
	// CreatedAt is the secret creation timestamp
	CreatedAt *string `json:"createdAt,omitempty"`

	// UpdatedAt is the secret last update timestamp
	UpdatedAt *string `json:"updatedAt,omitempty"`

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

// OrganizationSecretSpec defines the desired state of OrganizationSecret
type OrganizationSecretSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              OrganizationSecretParameters `json:"forProvider"`
}

// OrganizationSecretStatus defines the observed state of OrganizationSecret
type OrganizationSecretStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`
	AtProvider                 OrganizationSecretObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=orga
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// OrganizationSecret is the Schema for the organizationsecrets API v2 (namespaced)
type OrganizationSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSecretSpec   `json:"spec,omitempty"`
	Status OrganizationSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrganizationSecretList contains a list of OrganizationSecret
type OrganizationSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationSecret `json:"items"`
}

// OrganizationSecret type metadata
var (
	OrganizationSecretKind             = "OrganizationSecret"
	OrganizationSecretGroupKind        = schema.GroupKind{Group: Group, Kind: OrganizationSecretKind}
	OrganizationSecretKindAPIVersion   = OrganizationSecretKind + "." + SchemeGroupVersion.String()
	OrganizationSecretGroupVersionKind = SchemeGroupVersion.WithKind(OrganizationSecretKind)
)

// GetCondition returns the condition for the given ConditionType if it exists, otherwise returns nil.
func (r *OrganizationSecret) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return r.Status.GetCondition(ct)
}

// SetConditions sets the supplied conditions, replacing any existing conditions of the same type.
func (r *OrganizationSecret) SetConditions(c ...xpv1.Condition) {
	r.Status.SetConditions(c...)
}

// GetManagementPolicies returns the management policies for this resource.
func (r *OrganizationSecret) GetManagementPolicies() xpv1.ManagementPolicies {
	return r.Spec.ManagementPolicies
}

// SetManagementPolicies sets the management policies for this resource.
func (r *OrganizationSecret) SetManagementPolicies(p xpv1.ManagementPolicies) {
	r.Spec.ManagementPolicies = p
}

func init() {
	SchemeBuilder.Register(&OrganizationSecret{}, &OrganizationSecretList{})
}
