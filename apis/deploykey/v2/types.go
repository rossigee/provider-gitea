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

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

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

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

type DeployKeyObservation struct {
	// ID is the deploy key ID
	ID *int64 `json:"id,omitempty"`

	// URL is the deploy key URL
	URL *string `json:"url,omitempty"`

	// Fingerprint is the key fingerprint
	Fingerprint *string `json:"fingerprint,omitempty"`

	// CreatedAt is the creation timestamp
	CreatedAt *string `json:"createdAt,omitempty"`

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

// DeployKeySpec defines the desired state of DeployKey
type DeployKeySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       DeployKeyParameters `json:"forProvider"`
}

// DeployKeyStatus defines the observed state of DeployKey
type DeployKeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DeployKeyObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=depl
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// DeployKey is the Schema for the deploykeys API v2 (namespaced)
type DeployKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeployKeySpec   `json:"spec,omitempty"`
	Status DeployKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DeployKeyList contains a list of DeployKey
type DeployKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeployKey `json:"items"`
}

// DeployKey type metadata
var (
	DeployKeyKind             = "DeployKey"
	DeployKeyGroupKind        = schema.GroupKind{Group: Group, Kind: DeployKeyKind}
	DeployKeyKindAPIVersion   = DeployKeyKind + "." + SchemeGroupVersion.String()
	DeployKeyGroupVersionKind = SchemeGroupVersion.WithKind(DeployKeyKind)
)

func init() {
	SchemeBuilder.Register(&DeployKey{}, &DeployKeyList{})
}
