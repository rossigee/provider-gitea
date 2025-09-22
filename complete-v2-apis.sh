#!/bin/bash

# Script to completely copy all types from v1alpha1 to v2 APIs
# This includes all supporting structs and enums

set -e

# Function to completely copy and enhance a v2 API
complete_v2_api() {
    local resource=$1
    local v1_types="apis/$resource/v1alpha1/types.go"
    local v2_types="apis/$resource/v2/types.go"

    if [[ ! -f "$v1_types" ]]; then
        echo "Warning: $v1_types not found, skipping $resource"
        return
    fi

    echo "Completing v2 API for $resource..."

    # Extract the main type name
    local type_name=$(grep -E "^type [A-Z][a-zA-Z]*Parameters struct" "$v1_types" | head -1 | sed 's/type \([A-Z][a-zA-Z]*\)Parameters.*/\1/')
    if [[ -z "$type_name" ]]; then
        echo "Warning: Could not extract type name from $v1_types, skipping $resource"
        return
    fi

    # Get short name if available
    local short_name=$(grep -E "shortName=" "apis/$resource/v1alpha1/types.go" | sed 's/.*shortName=\([^,}]*\).*/\1/' | head -1 | tr -d '"')
    if [[ -z "$short_name" ]]; then
        short_name="${resource:0:4}"  # Default to first 4 chars
    fi

    # Create the header
    cat > "$v2_types" << 'EOF'
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

EOF

    # Copy all supporting types (anything that's not the main 4 types)
    awk '
    /^type [A-Z][a-zA-Z]*Parameters struct/,/^}/ { next }
    /^type [A-Z][a-zA-Z]*Observation struct/,/^}/ { next }
    /^type [A-Z][a-zA-Z]*Spec struct/,/^}/ { next }
    /^type [A-Z][a-zA-Z]*Status struct/,/^}/ { next }
    /^type [A-Z][a-zA-Z]* struct/,/^}/ { next }
    /^type [A-Z][a-zA-Z]*List struct/,/^}/ { next }
    /^\/\/ [A-Z][a-zA-Z]* type metadata/,/^func init/ { next }
    /^type / {
        # This is a supporting type, copy it
        intype = 1
        print
        next
    }
    intype && /^}/ {
        print
        print ""
        intype = 0
        next
    }
    intype { print }
    ' "$v1_types" >> "$v2_types"

    # Copy the Parameters struct and enhance it
    awk "/^type ${type_name}Parameters struct/,/^}/" "$v1_types" | sed '$d' >> "$v2_types"

    # Add v2 enhancements to Parameters
    cat >> "$v2_types" << EOF

	// V2 Enhancement: Connection reference for multi-tenant support
	// ConnectionRef specifies the Gitea connection to use
	ConnectionRef *xpv1.Reference \`json:"connectionRef,omitempty"\`

	// V2 Enhancement: Namespace-scoped provider config
	// ProviderConfigRef references a ProviderConfig resource in the same namespace
	ProviderConfigRef *xpv1.Reference \`json:"providerConfigRef,omitempty"\`
}

EOF

    # Copy the Observation struct if it exists, otherwise create a basic one
    if grep -q "^type ${type_name}Observation struct" "$v1_types"; then
        awk "/^type ${type_name}Observation struct/,/^}/" "$v1_types" | sed '$d' >> "$v2_types"
        cat >> "$v2_types" << EOF

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

EOF
    else
        cat >> "$v2_types" << EOF
// ${type_name}Observation reflects the observed state of a Gitea ${type_name}
type ${type_name}Observation struct {
	// ID is the unique identifier
	ID *int64 \`json:"id,omitempty"\`

	// CreatedAt is the creation timestamp
	CreatedAt *metav1.Time \`json:"createdAt,omitempty"\`

	// UpdatedAt is the last update timestamp
	UpdatedAt *metav1.Time \`json:"updatedAt,omitempty"\`

	// V2 Enhancement: Enhanced observability
	// Additional fields can be added here for better monitoring
}

EOF
    fi

    # Add the main v2 API structures
    cat >> "$v2_types" << EOF
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
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gitea},shortName=${short_name}
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

    echo "  Completed v2 API for $resource with all supporting types"
}

# Resources that need complete v2 APIs
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

echo "Creating complete v2 APIs with all supporting types..."

for resource in "${RESOURCES[@]}"; do
    complete_v2_api "$resource"
done

echo ""
echo "Complete v2 API creation finished!"
echo "Next: Run 'make generate' to generate boilerplate code"