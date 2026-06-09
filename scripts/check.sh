#!/bin/bash
set -euo pipefail

# check.sh - Quick validation before commits
# Runs: generate, lint, test

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== Running code generation ==="
make generate

echo "=== Running linter ==="
make lint

echo "=== Running tests ==="
make test

echo "=== All checks passed ==="