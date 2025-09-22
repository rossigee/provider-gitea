#!/bin/bash

# Script to create v2 namespaced APIs for all Gitea provider resources
# This script reads v1alpha1 APIs and creates corresponding v2 APIs with .m. API group pattern

set -e

# Resources that need v2 APIs (excluding repository and organization which are already done)
RESOURCES=(
    "accesstoken"
    "action"
    "adminuser"
    "branchprotection"
    "deploykey"
    "githook"
    "issue"
    "label"
    "organizationmember"
    "organizationsecret"
    "organizationsettings"
    "pullrequest"
    "release"
    "repositorycollaborator"
    "repositorykey"
    "repositorysecret"
    "runner"
    "team"
    "user"
    "userkey"
    "webhook"
)

# Function to create v2 API for a resource
create_v2_api() {
    local resource=$1
    local resource_title=$(echo "$resource" | sed 's/.*/\u&/')  # Capitalize first letter

    echo "Creating v2 API for $resource..."

    # Create directory
    mkdir -p "apis/$resource/v2"

    # Create groupversion_info.go
    cat > "apis/$resource/v2/groupversion_info.go" << EOF
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

// Package v2 contains the v2 API of $resource
// +kubebuilder:object:generate=true
// +groupName=${resource}.gitea.m.crossplane.io
// +versionName=v2
package v2

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "${resource}.gitea.m.crossplane.io"
	Version = "v2"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
EOF

    # Read the v1alpha1 types.go to understand the structure
    local v1_types="apis/$resource/v1alpha1/types.go"
    if [[ ! -f "$v1_types" ]]; then
        echo "Warning: $v1_types not found, skipping $resource"
        return
    fi

    # Extract the main type name from v1alpha1 types.go
    local type_name=$(grep -E "^type [A-Z][a-zA-Z]*Parameters struct" "$v1_types" | head -1 | sed 's/type \([A-Z][a-zA-Z]*\)Parameters.*/\1/')
    if [[ -z "$type_name" ]]; then
        echo "Warning: Could not extract type name from $v1_types, skipping $resource"
        return
    fi

    echo "  Found type: $type_name"

    # Create a basic v2 types.go file - this will need manual refinement
    cat > "apis/$resource/v2/types.go" << EOF
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

// NOTE: This is a generated v2 API template.
// The Parameters and Observation types need to be copied and enhanced from v1alpha1.
// Add v2 enhancements like ConnectionRef and enhanced observability fields.

// ${type_name}Spec defines the desired state of ${type_name}
type ${type_name}Spec struct {
	xpv1.ResourceSpec \`json:",inline"\`
	ForProvider       ${type_name}Parameters \`json:"forProvider"\`
}

// ${type_name}Status defines the observed state of ${type_name}
type ${type_name}Status struct {
	xpv1.ResourceStatus \`json:",inline"\`
	AtProvider          ${type_name}Observation \`json:"atProvider,omitempty"\`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// ${type_name} is the Schema for the ${resource}s API v2 (namespaced)
type ${type_name} struct {
	metav1.TypeMeta   \`json:",inline"\`
	metav1.ObjectMeta \`json:"metadata,omitempty"\`

	Spec   ${type_name}Spec   \`json:"spec,omitempty"\`
	Status ${type_name}Status \`json:"status,omitempty"\`
}

// +kubebuilder:object:root=true

// ${type_name}List contains a list of ${type_name}
type ${type_name}List struct {
	metav1.TypeMeta \`json:",inline"\`
	metav1.ListMeta \`json:"metadata,omitempty"\`
	Items           []${type_name} \`json:"items"\`
}

// ${type_name} type metadata
var (
	${type_name}Kind             = "${type_name}"
	${type_name}GroupKind        = schema.GroupKind{Group: Group, Kind: ${type_name}Kind}
	${type_name}KindAPIVersion   = ${type_name}Kind + "." + SchemeGroupVersion.String()
	${type_name}GroupVersionKind = SchemeGroupVersion.WithKind(${type_name}Kind)
)

func init() {
	SchemeBuilder.Register(&${type_name}{}, &${type_name}List{})
}
EOF

    echo "  Created basic v2 API structure for $resource"
    echo "  WARNING: Manual enhancement needed for Parameters and Observation types"
}

# Main execution
echo "Creating v2 APIs for Gitea provider resources..."

for resource in "${RESOURCES[@]}"; do
    create_v2_api "$resource"
done

echo ""
echo "v2 API creation complete!"
echo ""
echo "NEXT STEPS:"
echo "1. Copy Parameters and Observation types from v1alpha1 to v2 for each resource"
echo "2. Add v2 enhancements (ConnectionRef, ProviderConfigRef, enhanced observability)"
echo "3. Run 'make generate' to generate the necessary boilerplate code"
echo "4. Update controllers to support dual-scope resources"
echo "5. Create v2 examples and documentation"