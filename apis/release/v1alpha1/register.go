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

// Package v1alpha1 contains the v1alpha1 group Release resources of the Gitea provider.
// +kubebuilder:object:generate=true
// +groupName=release.gitea.crossplane.io
// +versionName=v1alpha1
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	CRDGroup   = "release.gitea.crossplane.io"
	CRDVersion = "v1alpha1"
)

var (
	// CRDGroupVersion is the API Group Version used to register the objects
	CRDGroupVersion = schema.GroupVersion{Group: CRDGroup, Version: CRDVersion}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: CRDGroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// Release type metadata.
const (
	ReleaseKind      = "Release"
	ReleaseGroupKind = "Release." + CRDGroup
)

// Release type metadata.
var (
	ReleaseKindAPIVersion   = ReleaseKind + "." + CRDGroupVersion.String()
	ReleaseGroupVersionKind = CRDGroupVersion.WithKind(ReleaseKind)
)

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}

// Group returns the API Group of Release resources.
func Group() string {
	return CRDGroup
}

// Version returns the API Version of Release resources.
func Version() string {
	return CRDVersion
}

// SchemeGroupVersion returns the GroupVersion for Release resources.
func SchemeGroupVersion() schema.GroupVersion {
	return CRDGroupVersion
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return CRDGroupVersion.WithResource(resource).GroupResource()
}