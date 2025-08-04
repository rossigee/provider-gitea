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

// ActionParameters define the desired state of a Gitea Actions Workflow
type ActionParameters struct {
	// Repository is the repository that owns this action (owner/name format)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Repository string `json:"repository"`

	// WorkflowName is the name of the workflow file
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9._-]+\\.ya?ml$"
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	WorkflowName string `json:"workflowName"`

	// Content is the YAML content of the workflow file
	// +kubebuilder:validation:Required
	Content string `json:"content"`

	// Branch is the branch where the workflow should be created/updated
	// +kubebuilder:default="main"
	Branch *string `json:"branch,omitempty"`

	// CommitMessage is the commit message for the workflow change
	// +kubebuilder:default="Update workflow via Crossplane"
	CommitMessage *string `json:"commitMessage,omitempty"`

	// Enabled indicates if the workflow is enabled
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`
}

// ActionObservation reflects the observed state of a Gitea Actions Workflow
type ActionObservation struct {
	// WorkflowName is the name of the workflow
	WorkflowName *string `json:"workflowName,omitempty"`

	// ID is the workflow identifier
	ID *int64 `json:"id,omitempty"`

	// State indicates the workflow state (active, disabled_manually, disabled_fork)
	State *string `json:"state,omitempty"`

	// CreatedAt is the timestamp when the workflow was created
	CreatedAt *string `json:"createdAt,omitempty"`

	// UpdatedAt is the timestamp when the workflow was last updated
	UpdatedAt *string `json:"updatedAt,omitempty"`

	// URL is the API URL for the workflow
	URL *string `json:"url,omitempty"`

	// HTMLURL is the web URL for the workflow
	HTMLURL *string `json:"htmlUrl,omitempty"`

	// BadgeURL is the status badge URL for the workflow
	BadgeURL *string `json:"badgeUrl,omitempty"`

	// Repository is the repository information
	Repository *string `json:"repository,omitempty"`

	// Branch is the branch where the workflow exists
	Branch *string `json:"branch,omitempty"`

	// LastRun contains information about the last workflow run
	LastRun *ActionLastRun `json:"lastRun,omitempty"`
}

// ActionLastRun contains information about the last workflow run
type ActionLastRun struct {
	// ID is the run identifier
	ID *int64 `json:"id,omitempty"`

	// Status is the run status (queued, in_progress, completed)
	Status *string `json:"status,omitempty"`

	// Conclusion is the run conclusion (success, failure, cancelled, skipped)
	Conclusion *string `json:"conclusion,omitempty"`

	// CreatedAt is when the run was created
	CreatedAt *string `json:"createdAt,omitempty"`

	// UpdatedAt is when the run was last updated
	UpdatedAt *string `json:"updatedAt,omitempty"`

	// RunNumber is the sequential run number
	RunNumber *int64 `json:"runNumber,omitempty"`

	// Event is the event that triggered the run
	Event *string `json:"event,omitempty"`
}

// A ActionSpec defines the desired state of a Action.
type ActionSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ActionParameters `json:"forProvider"`
}

// A ActionStatus represents the observed state of a Action.
type ActionStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ActionObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Action is a managed resource that represents a Gitea Actions Workflow.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="REPOSITORY",type="string",JSONPath=".spec.forProvider.repository"
// +kubebuilder:printcolumn:name="WORKFLOW",type="string",JSONPath=".spec.forProvider.workflowName"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="ENABLED",type="boolean",JSONPath=".spec.forProvider.enabled"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type Action struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionSpec   `json:"spec"`
	Status ActionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ActionList contains a list of Action
type ActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Action `json:"items"`
}
