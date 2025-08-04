#!/bin/bash

# Crossplane-specific linting checks

set -euo pipefail

errors=0

# Check that API types follow Crossplane conventions
for file in "$@"; do
    if [[ "$file" == *"/apis/"* && "$file" == *"/types.go" ]]; then
        # Check for required runtime.Object implementation
        if grep -q "type.*struct" "$file" && ! grep -q "metav1.TypeMeta" "$file"; then
            echo "ERROR: $file - API types should embed metav1.TypeMeta"
            ((errors++))
        fi

        # Check for required Crossplane resource embedding for managed resources
        if grep -q "type.*struct" "$file" && [[ "$file" != *"/v1beta1/"* ]]; then
            if ! grep -q "xpv1.ResourceSpec" "$file" && ! grep -q "xpv1.ResourceStatus" "$file"; then
                echo "WARNING: $file - Managed resources should embed xpv1.ResourceSpec and xpv1.ResourceStatus"
            fi
        fi
    fi

    # Check controller patterns
    if [[ "$file" == *"/internal/controller/"* ]]; then
        # Check for proper error wrapping
        if grep -q "return.*err" "$file" && ! grep -q "errors.Wrap\|fmt.Errorf" "$file"; then
            echo "WARNING: $file - Consider using error wrapping for better error context"
        fi

        # Check for proper reconcile result patterns
        if grep -q "ctrl.Result" "$file" && ! grep -q "reconcile.Result" "$file"; then
            echo "WARNING: $file - Use reconcile.Result instead of ctrl.Result for consistency"
        fi
    fi
done

if [ $errors -gt 0 ]; then
    echo "Found $errors Crossplane linting errors"
    exit 1
fi

echo "Crossplane linting passed"
