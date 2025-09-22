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

type WebhookParameters struct {
	// Repository is the repository name (required for repository webhooks)
	Repository *string `json:"repository,omitempty"`

	// Owner is the owner/organization of the repository
	Owner *string `json:"owner,omitempty"`

	// Organization is the organization name (for organization webhooks)
	Organization *string `json:"organization,omitempty"`

	// Type is the webhook type
	// +kubebuilder:validation:Enum=gitea;gogs;slack;discord;dingtalk;telegram;msteams;feishu;wechatwork;packagist
	// +kubebuilder:default="gitea"
	Type *string `json:"type,omitempty"`

	// URL is the webhook payload URL
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// ContentType is the content type for the payload
	// +kubebuilder:validation:Enum=json;form
	// +kubebuilder:default="json"
	ContentType *string `json:"contentType,omitempty"`

	// Secret is the webhook secret for payload validation
	Secret *string `json:"secret,omitempty"`

	// Active determines if the webhook is active
	// +kubebuilder:default=true
	Active *bool `json:"active,omitempty"`

	// Events is the list of events to trigger the webhook
	// +kubebuilder:default={"push"}
	Events []string `json:"events,omitempty"`

	// BranchFilter is the branch filter for push events
	BranchFilter *string `json:"branchFilter,omitempty"`

	// SSLVerification enables SSL certificate verification
	// +kubebuilder:default=true
	SSLVerification *bool `json:"sslVerification,omitempty"`

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

type WebhookObservation struct {
	// ID is the webhook ID
	ID *int64 `json:"id,omitempty"`

	// CreatedAt is the webhook creation timestamp
	CreatedAt *string `json:"createdAt,omitempty"`

	// UpdatedAt is the webhook last update timestamp
	UpdatedAt *string `json:"updatedAt,omitempty"`

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

// WebhookSpec defines the desired state of Webhook
type WebhookSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       WebhookParameters `json:"forProvider"`
}

// WebhookStatus defines the observed state of Webhook
type WebhookStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          WebhookObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=webh
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// Webhook is the Schema for the webhooks API v2 (namespaced)
type Webhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebhookSpec   `json:"spec,omitempty"`
	Status WebhookStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WebhookList contains a list of Webhook
type WebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Webhook `json:"items"`
}

// Webhook type metadata
var (
	WebhookKind             = "Webhook"
	WebhookGroupKind        = schema.GroupKind{Group: Group, Kind: WebhookKind}
	WebhookKindAPIVersion   = WebhookKind + "." + SchemeGroupVersion.String()
	WebhookGroupVersionKind = SchemeGroupVersion.WithKind(WebhookKind)
)

func init() {
	SchemeBuilder.Register(&Webhook{}, &WebhookList{})
}
