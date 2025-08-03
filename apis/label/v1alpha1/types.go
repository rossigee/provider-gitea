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

// LabelParameters define the desired state of a Gitea Label
type LabelParameters struct {
	// Name is the label name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9 ._-]*[a-zA-Z0-9])?$"
	Name string `json:"name"`

	// Repository is the repository that owns this label (owner/name format)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Repository string `json:"repository"`

	// Color is the label color in hex format (without #)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[0-9a-fA-F]{6}$"
	Color string `json:"color"`

	// Description is the label description
	Description *string `json:"description,omitempty"`

	// Exclusive specifies if this is an exclusive (scoped) label
	// Exclusive labels ensure only one label with the same scope can be assigned
	// +kubebuilder:default=false
	Exclusive *bool `json:"exclusive,omitempty"`
}

// LabelObservation reflects the observed state of a Gitea Label
type LabelObservation struct {
	// ID is the label ID
	ID *int64 `json:"id,omitempty"`

	// URL is the label URL
	URL *string `json:"url,omitempty"`
}

// A LabelSpec defines the desired state of a Label.
type LabelSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       LabelParameters `json:"forProvider"`
}

// A LabelStatus represents the observed state of a Label.
type LabelStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          LabelObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Label is a managed resource that represents a Gitea Repository Label.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="REPOSITORY",type="string",JSONPath=".spec.forProvider.repository"
// +kubebuilder:printcolumn:name="COLOR",type="string",JSONPath=".spec.forProvider.color"
// +kubebuilder:printcolumn:name="EXCLUSIVE",type="boolean",JSONPath=".spec.forProvider.exclusive"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea}
type Label struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LabelSpec   `json:"spec"`
	Status LabelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LabelList contains a list of Label
type LabelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Label `json:"items"`
}