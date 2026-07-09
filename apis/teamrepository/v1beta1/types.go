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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// TeamRepositoryParameters define the desired state of a Gitea team-repository
// attachment. There is no mutable attribute — a repository is either attached
// to the team or it is not.
type TeamRepositoryParameters struct {
	// Organization is the repo owner and the team's organization.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Organization string `json:"organization"`

	// Team is the team name within the organization.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Team string `json:"team"`

	// Repository is the unqualified repository name (no owner prefix).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Repository string `json:"repository"`

	// TeamID optionally skips organization/team name resolution when the
	// numeric team id is already known.
	TeamID *int64 `json:"teamId,omitempty"`

	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

// TeamRepositoryObservation reflects the observed state of a team-repository
// attachment.
type TeamRepositoryObservation struct {
	// TeamID is the resolved numeric team id backing this attachment.
	TeamID *int64 `json:"teamId,omitempty"`
}

// TeamRepositorySpec defines the desired state of TeamRepository
type TeamRepositorySpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              TeamRepositoryParameters `json:"forProvider"`
}

// TeamRepositoryStatus defines the observed state of TeamRepository
type TeamRepositoryStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`
	AtProvider                 TeamRepositoryObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=trepo
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// TeamRepository is the Schema for the teamrepositories API v2 (namespaced)
type TeamRepository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamRepositorySpec   `json:"spec,omitempty"`
	Status TeamRepositoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamRepositoryList contains a list of TeamRepository
type TeamRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TeamRepository `json:"items"`
}

// TeamRepository type metadata
var (
	TeamRepositoryKind             = "TeamRepository"
	TeamRepositoryGroupKind        = schema.GroupKind{Group: Group, Kind: TeamRepositoryKind}
	TeamRepositoryKindAPIVersion   = TeamRepositoryKind + "." + SchemeGroupVersion.String()
	TeamRepositoryGroupVersionKind = SchemeGroupVersion.WithKind(TeamRepositoryKind)
)

// GetCondition returns the condition for the given ConditionType if it exists, otherwise returns nil.
func (r *TeamRepository) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return r.Status.GetCondition(ct)
}

// SetConditions sets the supplied conditions, replacing any existing conditions of the same type.
func (r *TeamRepository) SetConditions(c ...xpv1.Condition) {
	r.Status.SetConditions(c...)
}

// GetManagementPolicies returns the management policies for this resource.
func (r *TeamRepository) GetManagementPolicies() xpv1.ManagementPolicies {
	return r.Spec.ManagementPolicies
}

// SetManagementPolicies sets the management policies for this resource.
func (r *TeamRepository) SetManagementPolicies(p xpv1.ManagementPolicies) {
	r.Spec.ManagementPolicies = p
}

func init() {
	SchemeBuilder.Register(&TeamRepository{}, &TeamRepositoryList{})
}
