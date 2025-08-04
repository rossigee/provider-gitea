#!/bin/bash

# Check that all Go files have the required Apache 2.0 license header

set -euo pipefail

readonly LICENSE_HEADER="/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the \"License\");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an \"AS IS\" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/"

missing_license=()

for file in "$@"; do
    if ! head -n 15 "$file" | grep -q "Licensed under the Apache License, Version 2.0"; then
        missing_license+=("$file")
    fi
done

if [ ${#missing_license[@]} -gt 0 ]; then
    echo "The following files are missing the Apache 2.0 license header:"
    printf '%s\n' "${missing_license[@]}"
    echo ""
    echo "Please add the license header to these files."
    echo "You can use the boilerplate file at hack/boilerplate.go.txt"
    exit 1
fi

echo "All files have the required license header."
