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

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

type GitHookObservation struct {
	// Name is the hook name (typically same as hookType)
	Name *string `json:"name,omitempty"`

	// LastUpdated timestamp
	LastUpdated *string `json:"lastUpdated,omitempty"`

	// ContentHash is a hash of the content for drift detection
	ContentHash *string `json:"contentHash,omitempty"`

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

// GitHookSpec defines the desired state of GitHook
type GitHookSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       GitHookParameters `json:"forProvider"`
}

// GitHookStatus defines the observed state of GitHook
type GitHookStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          GitHookObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=gith
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// GitHook is the Schema for the githooks API v2 (namespaced)
type GitHook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitHookSpec   `json:"spec,omitempty"`
	Status GitHookStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitHookList contains a list of GitHook
type GitHookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitHook `json:"items"`
}

// GitHook type metadata
var (
	GitHookKind             = "GitHook"
	GitHookGroupKind        = schema.GroupKind{Group: Group, Kind: GitHookKind}
	GitHookKindAPIVersion   = GitHookKind + "." + SchemeGroupVersion.String()
	GitHookGroupVersionKind = SchemeGroupVersion.WithKind(GitHookKind)
)

func init() {
	SchemeBuilder.Register(&GitHook{}, &GitHookList{})
}
