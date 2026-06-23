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

type VariableParameters struct {
	// Name is the name of the Actions variable (must follow Gitea naming rules).
	// Variable names can only contain alphanumeric characters ([a-z], [A-Z],
	// [0-9]) or underscores (_); must not start with a number; must not start
	// with the GITHUB_ or GITEA_ prefix.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z_][a-zA-Z0-9_]*$"
	Name string `json:"name"`

	// Value is the variable value. Unlike a secret this is non-secret and
	// readable back from the API, so it is set inline in the spec (NOT via a
	// Secret reference) and real drift is detected against the live value.
	// +kubebuilder:validation:Required
	Value string `json:"value"`

	// Organization is the organization that owns this variable (org scope).
	// Exactly one of organization or repository must be set; if repository is
	// set the variable is repo-scoped, otherwise it is org-scoped.
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Organization *string `json:"organization,omitempty"`

	// Repository is the repository that owns this variable (owner/name format,
	// repo scope). Exactly one of organization or repository must be set.
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Repository *string `json:"repository,omitempty"`

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

type VariableObservation struct {
	// ID is the variable ID, when surfaced by the backend.
	ID *int64 `json:"id,omitempty"`

	// CreatedAt is the variable creation timestamp, when surfaced.
	CreatedAt *string `json:"createdAt,omitempty"`

	// UpdatedAt is the variable last update timestamp, when surfaced.
	UpdatedAt *string `json:"updatedAt,omitempty"`
}

// VariableSpec defines the desired state of Variable
type VariableSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              VariableParameters `json:"forProvider"`
}

// VariableStatus defines the observed state of Variable
type VariableStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`
	AtProvider                 VariableObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=giteavar
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// Variable is the Schema for the variables API v2 (namespaced). It models a
// Gitea Actions variable (non-secret) at either organization or repository
// scope.
type Variable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VariableSpec   `json:"spec,omitempty"`
	Status VariableStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VariableList contains a list of Variable
type VariableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Variable `json:"items"`
}

// Variable type metadata
var (
	VariableKind             = "Variable"
	VariableGroupKind        = schema.GroupKind{Group: Group, Kind: VariableKind}
	VariableKindAPIVersion   = VariableKind + "." + SchemeGroupVersion.String()
	VariableGroupVersionKind = SchemeGroupVersion.WithKind(VariableKind)
)

// GetCondition returns the condition for the given ConditionType if it exists, otherwise returns nil.
func (r *Variable) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return r.Status.GetCondition(ct)
}

// SetConditions sets the supplied conditions, replacing any existing conditions of the same type.
func (r *Variable) SetConditions(c ...xpv1.Condition) {
	r.Status.SetConditions(c...)
}

// GetManagementPolicies returns the management policies for this resource.
func (r *Variable) GetManagementPolicies() xpv1.ManagementPolicies {
	return r.Spec.ManagementPolicies
}

// SetManagementPolicies sets the management policies for this resource.
func (r *Variable) SetManagementPolicies(p xpv1.ManagementPolicies) {
	r.Spec.ManagementPolicies = p
}

func init() {
	SchemeBuilder.Register(&Variable{}, &VariableList{})
}
