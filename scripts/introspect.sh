#!/bin/bash
set -euo pipefail

# introspect.sh - Check introspection/drift detection compliance
# Runs the central introspection compliance script from the metarepo

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
METAREPO_ROOT="$(cd "$ROOT_DIR/../.." && pwd)"

SCRIPT_PATH="$METAREPO_ROOT/scripts/check_introspection_compliance.sh"

if [[ ! -f "$SCRIPT_PATH" ]]; then
    echo "Error: Central introspection script not found at $SCRIPT_PATH"
    exit 1
fi

echo "=== Running introspection compliance check for $PROVIDER_NAME ==="
"$SCRIPT_PATH" "$ROOT_DIR"