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

// GitHookParameters define the desired state of a Gitea Git Hook
type GitHookParameters struct {
	// Repository is the repository that owns this hook (owner/name format)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Repository string `json:"repository"`

	// HookType is the type of Git hook
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=pre-receive;update;post-receive;pre-push;post-update
	HookType string `json:"hookType"`

	// Content is the script content of the hook
	// +kubebuilder:validation:Required
	Content string `json:"content"`

	// IsActive controls whether the hook is active
	// +kubebuilder:default=true
	IsActive *bool `json:"isActive,omitempty"`
}

// GitHookObservation reflects the observed state of a Gitea Git Hook
type GitHookObservation struct {
	// Name is the hook name (typically same as hookType)
	Name *string `json:"name,omitempty"`

	// LastUpdated timestamp
	LastUpdated *string `json:"lastUpdated,omitempty"`

	// ContentHash is a hash of the content for drift detection
	ContentHash *string `json:"contentHash,omitempty"`
}

// A GitHookSpec defines the desired state of a GitHook.
type GitHookSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       GitHookParameters `json:"forProvider"`
}

// A GitHookStatus represents the observed state of a GitHook.
type GitHookStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          GitHookObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A GitHook is a managed resource that represents a Gitea Repository Git Hook.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="REPOSITORY",type="string",JSONPath=".spec.forProvider.repository"
// +kubebuilder:printcolumn:name="HOOK-TYPE",type="string",JSONPath=".spec.forProvider.hookType"
// +kubebuilder:printcolumn:name="ACTIVE",type="boolean",JSONPath=".spec.forProvider.isActive"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type GitHook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitHookSpec   `json:"spec"`
	Status GitHookStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitHookList contains a list of GitHook
type GitHookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitHook `json:"items"`
}
