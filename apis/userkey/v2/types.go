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

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

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

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

// UserKeySpec defines the desired state of UserKey
type UserKeySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       UserKeyParameters `json:"forProvider"`
}

// UserKeyStatus defines the observed state of UserKey
type UserKeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserKeyObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=user
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// UserKey is the Schema for the userkeys API v2 (namespaced)
type UserKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserKeySpec   `json:"spec,omitempty"`
	Status UserKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserKeyList contains a list of UserKey
type UserKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserKey `json:"items"`
}

// UserKey type metadata
var (
	UserKeyKind             = "UserKey"
	UserKeyGroupKind        = schema.GroupKind{Group: Group, Kind: UserKeyKind}
	UserKeyKindAPIVersion   = UserKeyKind + "." + SchemeGroupVersion.String()
	UserKeyGroupVersionKind = SchemeGroupVersion.WithKind(UserKeyKind)
)

func init() {
	SchemeBuilder.Register(&UserKey{}, &UserKeyList{})
}
