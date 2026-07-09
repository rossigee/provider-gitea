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

// TeamMembershipParameters define the desired state of a Gitea team membership.
// Membership is binary — there is no per-member role; a member's rights come
// from the team's permission/units. Making a user an org owner is just
// membership in the org's auto-created "Owners" team.
type TeamMembershipParameters struct {
	// Organization is the organization that owns the team.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Organization string `json:"organization"`

	// Team is the team name within the organization (e.g. "Owners").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Team string `json:"team"`

	// Username is the user to add to the team.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Username string `json:"username"`

	// TeamID optionally skips organization/team name resolution when the
	// numeric team id is already known.
	TeamID *int64 `json:"teamId,omitempty"`

	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

// TeamMembershipObservation reflects the observed state of a team membership.
type TeamMembershipObservation struct {
	// TeamID is the resolved numeric team id backing this membership.
	TeamID *int64 `json:"teamId,omitempty"`
}

// TeamMembershipSpec defines the desired state of TeamMembership
type TeamMembershipSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              TeamMembershipParameters `json:"forProvider"`
}

// TeamMembershipStatus defines the observed state of TeamMembership
type TeamMembershipStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`
	AtProvider                 TeamMembershipObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=tmem
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// TeamMembership is the Schema for the teammemberships API v2 (namespaced)
type TeamMembership struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamMembershipSpec   `json:"spec,omitempty"`
	Status TeamMembershipStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamMembershipList contains a list of TeamMembership
type TeamMembershipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TeamMembership `json:"items"`
}

// TeamMembership type metadata
var (
	TeamMembershipKind             = "TeamMembership"
	TeamMembershipGroupKind        = schema.GroupKind{Group: Group, Kind: TeamMembershipKind}
	TeamMembershipKindAPIVersion   = TeamMembershipKind + "." + SchemeGroupVersion.String()
	TeamMembershipGroupVersionKind = SchemeGroupVersion.WithKind(TeamMembershipKind)
)

// GetCondition returns the condition for the given ConditionType if it exists, otherwise returns nil.
func (r *TeamMembership) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return r.Status.GetCondition(ct)
}

// SetConditions sets the supplied conditions, replacing any existing conditions of the same type.
func (r *TeamMembership) SetConditions(c ...xpv1.Condition) {
	r.Status.SetConditions(c...)
}

// GetManagementPolicies returns the management policies for this resource.
func (r *TeamMembership) GetManagementPolicies() xpv1.ManagementPolicies {
	return r.Spec.ManagementPolicies
}

// SetManagementPolicies sets the management policies for this resource.
func (r *TeamMembership) SetManagementPolicies(p xpv1.ManagementPolicies) {
	r.Spec.ManagementPolicies = p
}

func init() {
	SchemeBuilder.Register(&TeamMembership{}, &TeamMembershipList{})
}
