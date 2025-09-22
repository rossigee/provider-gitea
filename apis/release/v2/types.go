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

type ReleaseParameters struct {
	// Repository is the name of the repository
	// +kubebuilder:validation:Required
	Repository string `json:"repository"`

	// Owner is the username or organization name that owns the repository
	// +kubebuilder:validation:Required
	Owner string `json:"owner"`

	// TagName is the name of the tag for this release
	// +kubebuilder:validation:Required
	TagName string `json:"tagName"`

	// TargetCommitish specifies the commitish value that determines where the Git tag is created from
	// Can be any branch or commit SHA. Defaults to repository default branch if omitted
	TargetCommitish *string `json:"targetCommitish,omitempty"`

	// Name is the title of the release
	Name *string `json:"name,omitempty"`

	// Body is the description/release notes for the release
	Body *string `json:"body,omitempty"`

	// Draft indicates if this is a draft release
	// Draft releases are not visible to users without repository write access
	// +kubebuilder:default=false
	Draft *bool `json:"draft,omitempty"`

	// Prerelease indicates if this is a prerelease
	// Prereleases are marked as not ready for production and may be unstable
	// +kubebuilder:default=false
	Prerelease *bool `json:"prerelease,omitempty"`

	// GenerateNotes automatically generates release notes from commits and PRs
	// +kubebuilder:default=false
	GenerateNotes *bool `json:"generateNotes,omitempty"`

	// Assets is a list of assets to attach to this release
	Assets []ReleaseAsset `json:"assets,omitempty"`

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference `json:"connectionRef,omitempty"`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference `json:"providerConfigRef,omitempty"`
}

type ReleaseObservation struct {
	// ID is the unique identifier of the release
	ID int64 `json:"id,omitempty"`

	// TagName is the Git tag associated with the release
	TagName string `json:"tagName,omitempty"`

	// TargetCommitish is the commit SHA or branch that the tag points to
	TargetCommitish string `json:"targetCommitish,omitempty"`

	// Name is the title of the release
	Name string `json:"name,omitempty"`

	// Body is the release notes/description
	Body string `json:"body,omitempty"`

	// URL is the API URL of the release
	URL string `json:"url,omitempty"`

	// HTMLURL is the web URL of the release
	HTMLURL string `json:"htmlUrl,omitempty"`

	// TarballURL is the URL to download the source code as tarball
	TarballURL string `json:"tarballUrl,omitempty"`

	// ZipballURL is the URL to download the source code as zipball
	ZipballURL string `json:"zipballUrl,omitempty"`

	// UploadURL is the URL for uploading release assets
	UploadURL string `json:"uploadUrl,omitempty"`

	// Draft indicates if this is a draft release
	Draft bool `json:"draft,omitempty"`

	// Prerelease indicates if this is a prerelease
	Prerelease bool `json:"prerelease,omitempty"`

	// Author is the username of the release author
	Author string `json:"author,omitempty"`

	// CreatedAt is the creation timestamp
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// PublishedAt is the publication timestamp
	PublishedAt *metav1.Time `json:"publishedAt,omitempty"`

	// Assets contains information about attached assets
	Assets []ReleaseAssetObservation `json:"assets,omitempty"`

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

// ReleaseSpec defines the desired state of Release
type ReleaseSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ReleaseParameters `json:"forProvider"`
}

// ReleaseStatus defines the observed state of Release
type ReleaseStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ReleaseObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=rele
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// Release is the Schema for the releases API v2 (namespaced)
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseSpec   `json:"spec,omitempty"`
	Status ReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReleaseList contains a list of Release
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Release `json:"items"`
}

// Release type metadata
var (
	ReleaseKind             = "Release"
	ReleaseGroupKind        = schema.GroupKind{Group: Group, Kind: ReleaseKind}
	ReleaseKindAPIVersion   = ReleaseKind + "." + SchemeGroupVersion.String()
	ReleaseGroupVersionKind = SchemeGroupVersion.WithKind(ReleaseKind)
)

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}
