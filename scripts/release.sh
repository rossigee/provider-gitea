#!/bin/bash
set -euo pipefail

# release.sh - Automated release script
# Usage: ./scripts/release.sh <version>
# Example: ./scripts/release.sh v0.1.0

VERSION="${1:-}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
PROVIDER_NAME="$(basename "$ROOT_DIR")"

print_status() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

if [[ -z "$VERSION" ]]; then
    print_error "Usage: $0 <version>"
    print_error "Example: $0 v0.1.0"
    exit 1
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    print_error "Invalid version format: $VERSION"
    print_error "Version must be: vX.Y.Z"
    exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
    print_error "Working directory has uncommitted changes"
    exit 1
fi

print_status "Releasing $PROVIDER_NAME version $VERSION"

print_status "Building and publishing..."
PLATFORMS="${PLATFORMS:-linux/amd64}" make publish VERSION="$VERSION"

print_success "Released $PROVIDER_NAME $VERSION"