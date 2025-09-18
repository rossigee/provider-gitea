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

// TeamParameters define the desired state of a Gitea Team
type TeamParameters struct {
	// Name is the team name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$"
	Name string `json:"name"`

	// Organization is the organization that owns this team
	// +kubebuilder:validation:Required
	Organization string `json:"organization"`

	// Description is the team description
	Description *string `json:"description,omitempty"`

	// Permission specifies the team permission level
	// +kubebuilder:validation:Enum=read;write;admin
	// +kubebuilder:default="read"
	Permission *string `json:"permission,omitempty"`

	// CanCreateOrgRepo allows team members to create repositories in the organization
	// +kubebuilder:default=false
	CanCreateOrgRepo *bool `json:"canCreateOrgRepo,omitempty"`

	// IncludesAllRepositories gives the team access to all repositories in the organization
	// +kubebuilder:default=false
	IncludesAllRepositories *bool `json:"includesAllRepositories,omitempty"`

	// Units specifies the units/features the team has access to
	// Valid values: repo.code, repo.issues, repo.pulls, repo.releases, repo.wiki, repo.ext_wiki, repo.ext_issues
	Units []string `json:"units,omitempty"`
}

// TeamObservation reflects the observed state of a Gitea Team
type TeamObservation struct {
	// ID is the team ID
	ID *int64 `json:"id,omitempty"`

	// OrganizationID is the organization ID that owns this team
	OrganizationID *int64 `json:"organizationId,omitempty"`
}

// A TeamSpec defines the desired state of a Team.
type TeamSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       TeamParameters `json:"forProvider"`
}

// A TeamStatus represents the observed state of a Team.
type TeamStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          TeamObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Team is a managed resource that represents a Gitea Team.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="ORGANIZATION",type="string",JSONPath=".spec.forProvider.organization"
// +kubebuilder:printcolumn:name="PERMISSION",type="string",JSONPath=".spec.forProvider.permission"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea}
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec"`
	Status TeamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamList contains a list of Team
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}
