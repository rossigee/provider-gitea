#!/bin/bash
# Generate interface wrapper methods for all resource types
# This adds GetCondition and SetConditions wrapper methods to satisfy resource.Managed interface

set -e

RESOURCES=(
  "accesstoken"
  "action"
  "adminuser"
  "branchprotection"
  "deploykey"
  "githook"
  "issue"
  "label"
  "organization"
  "organizationmember"
  "organizationsecret"
  "organizationsettings"
  "pullrequest"
  "release"
  "repository"
  "repositorycollaborator"
  "repositorykey"
  "repositorysecret"
  "runner"
  "team"
  "user"
  "userkey"
  "webhook"
)

# Template for wrapper methods
generate_methods() {
  local resource_name=$1
  local resource_type=$2  # e.g., "Repository", "Organization"
  local api_version=$3     # v2

  cat > /tmp/methods_${resource_name}.go << EOF
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

package ${api_version}

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// GetCondition returns the condition for the given ConditionType if it exists, otherwise returns nil.
func (r *${resource_type}) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return r.Status.GetCondition(ct)
}

// SetConditions sets the supplied conditions, replacing any existing conditions of the same type.
func (r *${resource_type}) SetConditions(c ...xpv1.Condition) {
	r.Status.SetConditions(c...)
}
EOF
}

echo "Generating wrapper methods for resource.Managed interface..."
echo ""

# Map of resource names to type names
declare -A TYPE_MAP=(
  ["accesstoken"]="AccessToken"
  ["action"]="Action"
  ["adminuser"]="AdminUser"
  ["branchprotection"]="BranchProtection"
  ["deploykey"]="DeployKey"
  ["githook"]="GitHook"
  ["issue"]="Issue"
  ["label"]="Label"
  ["organization"]="Organization"
  ["organizationmember"]="OrganizationMember"
  ["organizationsecret"]="OrganizationSecret"
  ["organizationsettings"]="OrganizationSettings"
  ["pullrequest"]="PullRequest"
  ["release"]="Release"
  ["repository"]="Repository"
  ["repositorycollaborator"]="RepositoryCollaborator"
  ["repositorykey"]="RepositoryKey"
  ["repositorysecret"]="RepositorySecret"
  ["runner"]="Runner"
  ["team"]="Team"
  ["user"]="User"
  ["userkey"]="UserKey"
  ["webhook"]="Webhook"
)

for resource in "${RESOURCES[@]}"; do
  resource_type=${TYPE_MAP[$resource]}
  resource_dir="apis/${resource}/v2"

  if [ ! -d "$resource_dir" ]; then
    echo "❌ Directory not found: $resource_dir"
    continue
  fi

  # Check if methods already exist
  if grep -q "GetCondition" "$resource_dir/types.go" 2>/dev/null; then
    echo "⊘ ${resource_type}: Methods already exist"
    continue
  fi

  # Generate the methods file
  generate_methods "$resource" "$resource_type" "v2"

  # Append to types.go
  cat /tmp/methods_${resource}.go >> "$resource_dir/types.go"
  echo "✅ ${resource_type}: Methods generated"
done

echo ""
echo "✅ Done! Interface wrapper methods generated for all resources."
echo "Note: These methods delegate to the embedded Status type's Conditioned interface."
