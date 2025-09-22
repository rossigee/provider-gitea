#!/bin/bash

# Fix specific missing types identified in compilation errors

set -e

echo "Fixing specific missing types in v2 APIs..."

# Fix RunnerGroupInfo in runner/v2
echo "Fixing RunnerGroupInfo in runner/v2..."
if grep -q "RunnerGroupInfo" apis/runner/v1alpha1/types.go; then
    # Extract and add the type
    awk '/type RunnerGroupInfo struct/,/^}/' apis/runner/v1alpha1/types.go > /tmp/runner_type.txt
    # Insert before RunnerParameters
    sed -i '/type RunnerParameters struct/i\
// RunnerGroupInfo contains runner group information\
type RunnerGroupInfo struct {\
\t// ID is the group identifier\
\tID *int64 `json:"id,omitempty"`\
\
\t// Name is the group name\
\tName *string `json:"name,omitempty"`\
\
\t// Description is the group description\
\tDescription *string `json:"description,omitempty"`\
}\
' apis/runner/v2/types.go
fi

# Fix DataFromSource in organizationsecret/v2
echo "Fixing DataFromSource in organizationsecret/v2..."
if grep -q "DataFromSource" apis/organizationsecret/v1alpha1/types.go; then
    sed -i '/type OrganizationSecretParameters struct/i\
// DataFromSource represents the source of secret data\
type DataFromSource struct {\
\t// SecretKeyRef is a reference to a key in a Secret\
\tSecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`\
\
\t// Value is the direct value of the secret\
\tValue *string `json:"value,omitempty"`\
}\
\
// SecretKeySelector selects a key from a Secret\
type SecretKeySelector struct {\
\t// Name of the secret\
\tName string `json:"name"`\
\
\t// Namespace of the secret\
\tNamespace string `json:"namespace"`\
\
\t// Key within the secret\
\tKey string `json:"key"`\
}\
' apis/organizationsecret/v2/types.go
fi

# Fix RepositoryCollaboratorPermissions in repositorycollaborator/v2
echo "Fixing RepositoryCollaboratorPermissions in repositorycollaborator/v2..."
sed -i '/type RepositoryCollaboratorParameters struct/i\
// RepositoryCollaboratorPermissions defines the permissions for a collaborator\
type RepositoryCollaboratorPermissions struct {\
\t// Admin permission\
\tAdmin *bool `json:"admin,omitempty"`\
\
\t// Push permission\
\tPush *bool `json:"push,omitempty"`\
\
\t// Pull permission\
\tPull *bool `json:"pull,omitempty"`\
}\
' apis/repositorycollaborator/v2/types.go

# Fix BranchProtectionAppliedSettings in branchprotection/v2
echo "Fixing BranchProtectionAppliedSettings in branchprotection/v2..."
sed -i '/type BranchProtectionParameters struct/i\
// BranchProtectionAppliedSettings contains the currently applied protection settings\
type BranchProtectionAppliedSettings struct {\
\t// EnablePush indicates if push protection is enabled\
\tEnablePush *bool `json:"enablePush,omitempty"`\
\
\t// EnableStatusCheck indicates if status checks are required\
\tEnableStatusCheck *bool `json:"enableStatusCheck,omitempty"`\
\
\t// RequiredStatusChecks lists the required status checks\
\tRequiredStatusChecks []string `json:"requiredStatusChecks,omitempty"`\
}\
' apis/branchprotection/v2/types.go

# Fix AppliedOrganizationSettings in organizationsettings/v2
echo "Fixing AppliedOrganizationSettings in organizationsettings/v2..."
sed -i '/type OrganizationSettingsParameters struct/i\
// AppliedOrganizationSettings contains the currently applied organization settings\
type AppliedOrganizationSettings struct {\
\t// Description is the applied organization description\
\tDescription *string `json:"description,omitempty"`\
\
\t// Website is the applied organization website\
\tWebsite *string `json:"website,omitempty"`\
\
\t// Location is the applied organization location\
\tLocation *string `json:"location,omitempty"`\
}\
' apis/organizationsettings/v2/types.go

# Fix AdminUserStats in adminuser/v2
echo "Fixing AdminUserStats in adminuser/v2..."
sed -i '/type AdminUserParameters struct/i\
// AdminUserStats contains statistics about the admin user\
type AdminUserStats struct {\
\t// TotalUsers is the total number of users\
\tTotalUsers *int64 `json:"totalUsers,omitempty"`\
\
\t// TotalOrganizations is the total number of organizations\
\tTotalOrganizations *int64 `json:"totalOrganizations,omitempty"`\
\
\t// TotalRepositories is the total number of repositories\
\tTotalRepositories *int64 `json:"totalRepositories,omitempty"`\
}\
' apis/adminuser/v2/types.go

echo "Fixed missing types in v2 APIs"
echo "Next: Test compilation with 'go build ./apis/'"