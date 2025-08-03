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

package integration

import (
	"encoding/base64"
	"strings"
	"testing"

	branchprotectionv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/branchprotection/v1alpha1"
	organizationmemberv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/organizationmember/v1alpha1"
)

// intPtr helper function for pointer types
func intPtr(i int) *int {
	return &i
}

// TestBranchProtectionSecurity validates security configurations for branch protection
func TestBranchProtectionSecurity(t *testing.T) {
	tests := []struct {
		name           string
		protection     *branchprotectionv1alpha1.BranchProtection
		expectedSecure bool
		reason         string
	}{
		{
			name: "Secure Enterprise Configuration",
			protection: &branchprotectionv1alpha1.BranchProtection{
				Spec: branchprotectionv1alpha1.BranchProtectionSpec{
					ForProvider: branchprotectionv1alpha1.BranchProtectionParameters{
						Repository:                    "org/repo",
						Branch:                        "main",
						EnablePush:                    boolPtr(false),
						EnableStatusCheck:             boolPtr(true),
						RequiredApprovals:             intPtr(2),
						BlockOnRejectedReviews:        boolPtr(true),
						RequireSignedCommits:          boolPtr(true),
						BlockOnOutdatedBranch:         boolPtr(true),
						DismissStaleApprovals:         boolPtr(true),
						ProtectedFilePatterns:         stringPtr("*.config,Dockerfile,secrets/*"),
						StatusCheckContexts:           []string{"ci/build", "security/scan"},
					},
				},
			},
			expectedSecure: true,
			reason:         "comprehensive security settings enabled",
		},
		{
			name: "Insecure Configuration - No Approvals",
			protection: &branchprotectionv1alpha1.BranchProtection{
				Spec: branchprotectionv1alpha1.BranchProtectionSpec{
					ForProvider: branchprotectionv1alpha1.BranchProtectionParameters{
						Repository:        "org/repo",
						Branch:            "main",
						EnablePush:        boolPtr(true),
						RequiredApprovals: intPtr(0),
					},
				},
			},
			expectedSecure: false,
			reason:         "no required approvals and push enabled",
		},
		{
			name: "Insecure Configuration - No Status Checks",
			protection: &branchprotectionv1alpha1.BranchProtection{
				Spec: branchprotectionv1alpha1.BranchProtectionSpec{
					ForProvider: branchprotectionv1alpha1.BranchProtectionParameters{
						Repository:        "org/repo",
						Branch:            "main",
						EnableStatusCheck: boolPtr(false),
						RequiredApprovals: intPtr(1),
					},
				},
			},
			expectedSecure: false,
			reason:         "status checks disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secure := evaluateBranchProtectionSecurity(tt.protection)
			if secure != tt.expectedSecure {
				t.Errorf("Expected security level %v but got %v: %s", tt.expectedSecure, secure, tt.reason)
			}
		})
	}
}

func evaluateBranchProtectionSecurity(bp *branchprotectionv1alpha1.BranchProtection) bool {
	params := bp.Spec.ForProvider
	
	// Check critical security settings
	if (params.EnablePush != nil && *params.EnablePush) && 
	   (params.EnablePushWhitelist == nil || !*params.EnablePushWhitelist) {
		return false // Direct push allowed without restrictions
	}
	
	if params.RequiredApprovals == nil || *params.RequiredApprovals < 1 {
		return false // No required approvals
	}
	
	if params.EnableStatusCheck == nil || !*params.EnableStatusCheck {
		return false // No status checks
	}
	
	// Recommended security settings
	score := 0
	if params.RequiredApprovals != nil && *params.RequiredApprovals >= 2 { score++ }
	if params.BlockOnRejectedReviews != nil && *params.BlockOnRejectedReviews { score++ }
	if params.RequireSignedCommits != nil && *params.RequireSignedCommits { score++ }
	if params.BlockOnOutdatedBranch != nil && *params.BlockOnOutdatedBranch { score++ }
	if params.DismissStaleApprovals != nil && *params.DismissStaleApprovals { score++ }
	if len(params.StatusCheckContexts) > 0 { score++ }
	if params.ProtectedFilePatterns != nil && *params.ProtectedFilePatterns != "" { score++ }
	
	// Consider secure if it has at least 4 out of 7 recommended settings
	return score >= 4
}

// TestSSHKeySecurity validates SSH key formats and security
func TestSSHKeySecurity(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		expectValid bool
		reason     string
	}{
		{
			name: "Valid RSA Key",
			key: `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA+17FiQDlnT5hoKHDTkPAo6pN5aOtVw== user@example.com`,
			expectValid: true,
			reason: "properly formatted RSA key",
		},
		{
			name: "Valid Ed25519 Key", 
			key: `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG4rT3vTt99Eq/ieQXDFbYVTQyQsF3+lc0pO8vH+abcd user@example.com`,
			expectValid: true,
			reason: "properly formatted Ed25519 key",
		},
		{
			name: "Invalid Key Format",
			key: `invalid-key-format`,
			expectValid: false,
			reason: "malformed key",
		},
		{
			name: "Empty Key",
			key: ``,
			expectValid: false,
			reason: "empty key",
		},
		{
			name: "Key Without Email",
			key: `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA+17FiQDlnT5hoKHDTkPAo6pN5aOtVw==`,
			expectValid: true,
			reason: "key without email is still valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateSSHKey(tt.key)
			if valid != tt.expectValid {
				t.Errorf("Expected key validity %v but got %v: %s", tt.expectValid, valid, tt.reason)
			}
		})
	}
}

func validateSSHKey(key string) bool {
	if strings.TrimSpace(key) == "" {
		return false
	}
	
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return false
	}
	
	keyType := parts[0]
	keyData := parts[1]
	
	// Check valid key types
	validTypes := []string{"ssh-rsa", "ssh-ed25519", "ssh-ecdsa", "ssh-dss"}
	validType := false
	for _, t := range validTypes {
		if keyType == t {
			validType = true
			break
		}
	}
	
	if !validType {
		return false
	}
	
	// Basic length check for key data
	if len(keyData) < 50 {
		return false
	}
	
	return true
}

// TestAccessTokenSecurity validates access token scope security
func TestAccessTokenSecurity(t *testing.T) {
	tests := []struct {
		name         string
		scopes       []string
		expectSecure bool
		reason       string
	}{
		{
			name:         "Read-only Scopes",
			scopes:       []string{"read:repository", "read:issue"},
			expectSecure: true,
			reason:       "only read permissions",
		},
		{
			name:         "Minimal Write Scopes",
			scopes:       []string{"read:repository", "write:repository"},
			expectSecure: true,
			reason:       "minimal required write permissions",
		},
		{
			name:         "Excessive Permissions",
			scopes:       []string{"write:admin", "write:user", "write:organization"},
			expectSecure: false,
			reason:       "excessive administrative permissions",
		},
		{
			name:         "Mixed Secure Scopes",
			scopes:       []string{"read:repository", "write:issue", "read:pull_request"},
			expectSecure: true,
			reason:       "balanced read/write permissions",
		},
		{
			name:         "Admin Scope Present",
			scopes:       []string{"read:repository", "write:admin"},
			expectSecure: false,
			reason:       "admin scope is high risk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secure := evaluateTokenScopeSecurity(tt.scopes)
			if secure != tt.expectSecure {
				t.Errorf("Expected scope security %v but got %v: %s", tt.expectSecure, secure, tt.reason)
			}
		})
	}
}

func evaluateTokenScopeSecurity(scopes []string) bool {
	// Check for high-risk scopes
	highRiskScopes := []string{"write:admin", "write:user", "delete:repository"}
	for _, scope := range scopes {
		for _, risk := range highRiskScopes {
			if scope == risk {
				return false
			}
		}
	}
	
	// Count write permissions
	writeCount := 0
	for _, scope := range scopes {
		if strings.HasPrefix(scope, "write:") {
			writeCount++
		}
	}
	
	// Consider secure if fewer than 4 write permissions
	return writeCount < 4
}

// TestOrganizationMemberSecurity validates organization membership security
func TestOrganizationMemberSecurity(t *testing.T) {
	tests := []struct {
		name           string
		member         *organizationmemberv1alpha1.OrganizationMember
		expectedSecure bool
		reason         string
	}{
		{
			name: "Secure Member Role",
			member: &organizationmemberv1alpha1.OrganizationMember{
				Spec: organizationmemberv1alpha1.OrganizationMemberSpec{
					ForProvider: organizationmemberv1alpha1.OrganizationMemberParameters{
						Organization: "secure-org",
						Username:     "developer",
						Role:         "member",
						Visibility:   stringPtr("private"),
					},
				},
			},
			expectedSecure: true,
			reason:         "member role with private visibility",
		},
		{
			name: "Admin Role",
			member: &organizationmemberv1alpha1.OrganizationMember{
				Spec: organizationmemberv1alpha1.OrganizationMemberSpec{
					ForProvider: organizationmemberv1alpha1.OrganizationMemberParameters{
						Organization: "secure-org",
						Username:     "admin-user",
						Role:         "admin",
						Visibility:   stringPtr("public"),
					},
				},
			},
			expectedSecure: false,
			reason:         "admin role with public visibility",
		},
		{
			name: "Owner Role Private",
			member: &organizationmemberv1alpha1.OrganizationMember{
				Spec: organizationmemberv1alpha1.OrganizationMemberSpec{
					ForProvider: organizationmemberv1alpha1.OrganizationMemberParameters{
						Organization: "secure-org",
						Username:     "owner",
						Role:         "owner",
						Visibility:   stringPtr("private"),
					},
				},
			},
			expectedSecure: true,
			reason:         "owner role but private visibility",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secure := evaluateOrganizationMemberSecurity(tt.member)
			if secure != tt.expectedSecure {
				t.Errorf("Expected security level %v but got %v: %s", tt.expectedSecure, secure, tt.reason)
			}
		})
	}
}

func evaluateOrganizationMemberSecurity(member *organizationmemberv1alpha1.OrganizationMember) bool {
	params := member.Spec.ForProvider
	
	// Admin with public visibility is concerning
	if params.Role == "admin" && params.Visibility != nil && *params.Visibility == "public" {
		return false
	}
	
	// Generally secure for most configurations
	return true
}

// TestSecretHandling validates that secrets are properly referenced
func TestSecretHandling(t *testing.T) {
	tests := []struct {
		name        string
		secretData  string
		expectValid bool
		reason      string
	}{
		{
			name:        "Base64 Encoded Secret",
			secretData:  "UGFzc3dvcmQtVGVzdC0yMDI0",
			expectValid: true,
			reason:      "properly base64 encoded",
		},
		{
			name:        "Plain Text Secret",
			secretData:  "plain-text-password",
			expectValid: false,
			reason:      "not base64 encoded",
		},
		{
			name:        "Empty Secret",
			secretData:  "",
			expectValid: false,
			reason:      "empty secret data",
		},
		{
			name:        "Complex Base64 Secret",
			secretData:  "Q29tcGxleC1QYXNzd29yZC13aXRoLVNwZWNpYWwtQ2hhcnMtIUAjJCUyMDI0",
			expectValid: true,
			reason:      "complex password properly encoded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateSecretData(tt.secretData)
			if valid != tt.expectValid {
				t.Errorf("Expected secret validity %v but got %v: %s", tt.expectValid, valid, tt.reason)
			}
		})
	}
}

func validateSecretData(data string) bool {
	if strings.TrimSpace(data) == "" {
		return false
	}
	
	// Check if it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return false
	}
	
	// Check minimum length for decoded secret
	if len(decoded) < 8 {
		return false
	}
	
	return true
}

// TestRunnerSecurity validates runner security configurations
func TestRunnerSecurity(t *testing.T) {
	tests := []struct {
		name           string
		labels         []string
		scope          string
		expectedSecure bool
		reason         string
	}{
		{
			name:           "Secure Organization Runner",
			labels:         []string{"ubuntu-latest", "docker", "security"},
			scope:          "organization",
			expectedSecure: true,
			reason:         "organization scope with security label",
		},
		{
			name:           "Repository Runner",
			labels:         []string{"ubuntu-latest", "build"},
			scope:          "repository", 
			expectedSecure: true,
			reason:         "repository scope is most secure",
		},
		{
			name:           "System Runner Risk",
			labels:         []string{"ubuntu-latest", "admin", "system"},
			scope:          "system",
			expectedSecure: false,
			reason:         "system scope with admin labels",
		},
		{
			name:           "Privileged Labels",
			labels:         []string{"privileged", "docker-daemon", "root"},
			scope:          "organization",
			expectedSecure: false,
			reason:         "privileged labels present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secure := evaluateRunnerSecurity(tt.scope, tt.labels)
			if secure != tt.expectedSecure {
				t.Errorf("Expected security level %v but got %v: %s", tt.expectedSecure, secure, tt.reason)
			}
		})
	}
}

func evaluateRunnerSecurity(scope string, labels []string) bool {
	// System scope runners have elevated risk
	if scope == "system" {
		// Check for admin-related labels
		for _, label := range labels {
			if strings.Contains(strings.ToLower(label), "admin") ||
			   strings.Contains(strings.ToLower(label), "system") {
				return false
			}
		}
	}
	
	// Check for privileged labels
	privilegedLabels := []string{"privileged", "root", "docker-daemon", "admin"}
	for _, label := range labels {
		for _, priv := range privilegedLabels {
			if strings.Contains(strings.ToLower(label), priv) {
				return false
			}
		}
	}
	
	return true
}