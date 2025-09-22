#!/bin/bash

# Script to fix missing supporting types in v2 APIs
# Copies all supporting struct types from v1alpha1 to v2

set -e

# Function to add missing supporting types to a v2 API
fix_supporting_types() {
    local resource=$1
    local v1_types="apis/$resource/v1alpha1/types.go"
    local v2_types="apis/$resource/v2/types.go"

    if [[ ! -f "$v1_types" ]]; then
        echo "Warning: $v1_types not found, skipping $resource"
        return
    fi

    if [[ ! -f "$v2_types" ]]; then
        echo "Warning: $v2_types not found, skipping $resource"
        return
    fi

    echo "Fixing supporting types for $resource..."

    # Extract the main type name
    local type_name=$(grep -E "^type [A-Z][a-zA-Z]*Parameters struct" "$v1_types" | head -1 | sed 's/type \([A-Z][a-zA-Z]*\)Parameters.*/\1/')
    if [[ -z "$type_name" ]]; then
        echo "Warning: Could not extract type name from $v1_types, skipping $resource"
        return
    fi

    # Create a temporary file with all supporting types
    local temp_file=$(mktemp)

    # Extract all supporting types (not the main Parameters, Observation, Spec, Status, and List types)
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
    ' "$v1_types" > "$temp_file"

    # Check if there are any supporting types to add
    if [[ -s "$temp_file" ]]; then
        # Find the insertion point (after imports, before main types)
        local insertion_line=$(grep -n "^type ${type_name}Parameters struct" "$v2_types" | cut -d: -f1)
        if [[ -n "$insertion_line" ]]; then
            # Insert supporting types before the main Parameters type
            head -n $((insertion_line - 1)) "$v2_types" > "${v2_types}.tmp"
            cat "$temp_file" >> "${v2_types}.tmp"
            tail -n +$insertion_line "$v2_types" >> "${v2_types}.tmp"
            mv "${v2_types}.tmp" "$v2_types"
            echo "  Added supporting types to $resource v2 API"
        else
            echo "  Warning: Could not find insertion point in $v2_types"
        fi
    else
        echo "  No additional supporting types needed for $resource"
    fi

    rm -f "$temp_file"
}

# Resources that need supporting type fixes
FAILING_RESOURCES=(
    "organizationmember"
    "release"
    "runner"
    "organizationsecret"
    "repositorycollaborator"
    "branchprotection"
    "organizationsettings"
    "adminuser"
)

echo "Fixing missing supporting types in v2 APIs..."

for resource in "${FAILING_RESOURCES[@]}"; do
    fix_supporting_types "$resource"
done

echo ""
echo "Supporting type fixes complete!"
echo "Next: Test compilation with 'go build ./apis/'"