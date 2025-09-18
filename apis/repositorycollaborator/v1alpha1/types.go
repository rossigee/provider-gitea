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

// RepositoryCollaboratorParameters define the desired state of a Gitea Repository Collaborator
type RepositoryCollaboratorParameters struct {
	// Username is the collaborator's username
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Username string `json:"username"`

	// Repository is the repository to grant access to (owner/name format)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Repository string `json:"repository"`

	// Permission is the access level for the collaborator
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=read;write;admin
	// +kubebuilder:default="read"
	Permission string `json:"permission"`
}

// RepositoryCollaboratorObservation reflects the observed state of a Gitea Repository Collaborator
type RepositoryCollaboratorObservation struct {
	// FullName is the collaborator's full name
	FullName *string `json:"fullName,omitempty"`

	// Email is the collaborator's email address
	Email *string `json:"email,omitempty"`

	// AvatarURL is the collaborator's avatar URL
	AvatarURL *string `json:"avatarUrl,omitempty"`

	// Permissions are the detailed permissions of the collaborator
	Permissions *RepositoryCollaboratorPermissions `json:"permissions,omitempty"`
}

// RepositoryCollaboratorPermissions represents the permissions of a collaborator
type RepositoryCollaboratorPermissions struct {
	// Admin indicates if the collaborator has admin access
	Admin *bool `json:"admin,omitempty"`

	// Push indicates if the collaborator has push access
	Push *bool `json:"push,omitempty"`

	// Pull indicates if the collaborator has pull access
	Pull *bool `json:"pull,omitempty"`
}

// A RepositoryCollaboratorSpec defines the desired state of a RepositoryCollaborator.
type RepositoryCollaboratorSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RepositoryCollaboratorParameters `json:"forProvider"`
}

// A RepositoryCollaboratorStatus represents the observed state of a RepositoryCollaborator.
type RepositoryCollaboratorStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RepositoryCollaboratorObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A RepositoryCollaborator is a managed resource that represents a Gitea Repository Collaborator.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="REPOSITORY",type="string",JSONPath=".spec.forProvider.repository"
// +kubebuilder:printcolumn:name="USERNAME",type="string",JSONPath=".spec.forProvider.username"
// +kubebuilder:printcolumn:name="PERMISSION",type="string",JSONPath=".spec.forProvider.permission"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,gitea}
type RepositoryCollaborator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositoryCollaboratorSpec   `json:"spec"`
	Status RepositoryCollaboratorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepositoryCollaboratorList contains a list of RepositoryCollaborator
type RepositoryCollaboratorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RepositoryCollaborator `json:"items"`
}
