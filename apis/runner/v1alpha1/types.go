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

// RunnerParameters define the desired state of a Gitea Actions Runner
type RunnerParameters struct {
	// Scope defines where the runner is registered (repository, organization, or system)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=repository;organization;system
	Scope string `json:"scope"`

	// ScopeValue is the repository (owner/name) or organization name for the scope
	// Required for repository and organization scopes, ignored for system scope
	// +kubebuilder:validation:Pattern="^([a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?(/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?)?)?$"
	ScopeValue *string `json:"scopeValue,omitempty"`

	// Name is the display name for the runner
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=100
	Name string `json:"name"`

	// Labels are the labels assigned to this runner for job targeting
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:UniqueItems=true
	Labels []string `json:"labels"`

	// Description is an optional description for the runner
	// +kubebuilder:validation:MaxLength=500
	Description *string `json:"description,omitempty"`

	// RunnerGroupID is the ID of the runner group (for organization/system runners)
	RunnerGroupID *int64 `json:"runnerGroupId,omitempty"`
}

// RunnerObservation reflects the observed state of a Gitea Actions Runner
type RunnerObservation struct {
	// ID is the unique identifier of the runner
	ID *int64 `json:"id,omitempty"`

	// Name is the display name for the runner
	Name *string `json:"name,omitempty"`

	// UUID is the unique UUID of the runner
	UUID *string `json:"uuid,omitempty"`

	// Status indicates the runner status (online, offline, idle, active)
	Status *string `json:"status,omitempty"`

	// LastOnline is the timestamp when the runner was last online
	LastOnline *string `json:"lastOnline,omitempty"`

	// CreatedAt is the timestamp when the runner was registered
	CreatedAt *string `json:"createdAt,omitempty"`

	// UpdatedAt is the timestamp when the runner was last updated
	UpdatedAt *string `json:"updatedAt,omitempty"`

	// Labels are the current labels assigned to this runner
	Labels []string `json:"labels,omitempty"`

	// Description is the runner description
	Description *string `json:"description,omitempty"`

	// Scope indicates where the runner is registered
	Scope *string `json:"scope,omitempty"`

	// ScopeValue is the repository or organization name
	ScopeValue *string `json:"scopeValue,omitempty"`

	// RunnerGroup contains runner group information
	RunnerGroup *RunnerGroupInfo `json:"runnerGroup,omitempty"`

	// Version is the runner agent version
	Version *string `json:"version,omitempty"`

	// Architecture is the runner system architecture
	Architecture *string `json:"architecture,omitempty"`

	// OperatingSystem is the runner OS
	OperatingSystem *string `json:"operatingSystem,omitempty"`

	// TokenExpiresAt is when the runner token expires
	TokenExpiresAt *string `json:"tokenExpiresAt,omitempty"`
}

// RunnerGroupInfo contains runner group information
type RunnerGroupInfo struct {
	// ID is the runner group identifier
	ID *int64 `json:"id,omitempty"`

	// Name is the runner group name
	Name *string `json:"name,omitempty"`

	// IsDefault indicates if this is the default runner group
	IsDefault *bool `json:"isDefault,omitempty"`
}

// A RunnerSpec defines the desired state of a Runner.
type RunnerSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RunnerParameters `json:"forProvider"`
}

// A RunnerStatus represents the observed state of a Runner.
type RunnerStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RunnerObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Runner is a managed resource that represents a Gitea Actions Runner.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="SCOPE",type="string",JSONPath=".spec.forProvider.scope"
// +kubebuilder:printcolumn:name="SCOPE-VALUE",type="string",JSONPath=".spec.forProvider.scopeValue"
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerSpec   `json:"spec"`
	Status RunnerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RunnerList contains a list of Runner
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Runner `json:"items"`
}
