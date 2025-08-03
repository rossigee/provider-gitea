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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	actionv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/action/v1alpha1"
	adminuserv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/adminuser/v1alpha1"
	runnerv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/runner/v1alpha1"
	branchprotectionv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/branchprotection/v1alpha1"
	repositorykeyv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/repositorykey/v1alpha1"
	accesstokenv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/accesstoken/v1alpha1"
	repositorysecretv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/repositorysecret/v1alpha1"
	userkeyv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/userkey/v1alpha1"
	organizationmemberv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/organizationmember/v1alpha1"
)

// ExampleTest validates that example manifests are syntactically correct
// and can be parsed by the Kubernetes API machinery
type ExampleTest struct {
	name     string
	path     string
	scheme   *runtime.Scheme
	validate func(*testing.T, *unstructured.Unstructured)
}

func TestExampleManifests(t *testing.T) {
	scheme := runtime.NewScheme()
	
	// Add all our CRD types to the scheme
	if err := actionv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add action scheme: %v", err)
	}
	if err := adminuserv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add adminuser scheme: %v", err)
	}
	if err := runnerv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add runner scheme: %v", err)
	}
	if err := branchprotectionv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add branchprotection scheme: %v", err)
	}
	if err := repositorykeyv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add repositorykey scheme: %v", err)
	}
	if err := accesstokenv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add accesstoken scheme: %v", err)
	}
	if err := repositorysecretv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add repositorysecret scheme: %v", err)
	}
	if err := userkeyv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add userkey scheme: %v", err)
	}
	if err := organizationmemberv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add organizationmember scheme: %v", err)
	}

	tests := []ExampleTest{
		// Action examples
		{
			name:   "CI Pipeline Action",
			path:   "../../examples/action/ci-pipeline.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetKind() != "Action" {
					t.Errorf("Expected kind Action, got %s", obj.GetKind())
				}
				if obj.GetAPIVersion() != "action.gitea.crossplane.io/v1alpha1" {
					t.Errorf("Expected apiVersion action.gitea.crossplane.io/v1alpha1, got %s", obj.GetAPIVersion())
				}
			},
		},
		{
			name:   "Security Workflow Action",
			path:   "../../examples/action/security-workflow.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				spec, found, err := unstructured.NestedMap(obj.Object, "spec", "forProvider")
				if err != nil || !found {
					t.Errorf("Failed to find spec.forProvider: %v", err)
				}
				if content, found := spec["content"]; !found || content == "" {
					t.Errorf("Expected workflow content to be present")
				}
			},
		},
		
		// Runner examples
		{
			name:   "Repository Runner",
			path:   "../../examples/runner/repository-runner.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetKind() != "Runner" {
					t.Errorf("Expected kind Runner, got %s", obj.GetKind())
				}
				scope, _, _ := unstructured.NestedString(obj.Object, "spec", "forProvider", "scope")
				if scope != "repository" {
					t.Errorf("Expected scope repository, got %s", scope)
				}
			},
		},
		{
			name:   "Organization Runners",
			path:   "../../examples/runner/organization-runners.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				scope, _, _ := unstructured.NestedString(obj.Object, "spec", "forProvider", "scope")
				if scope != "organization" {
					t.Errorf("Expected scope organization, got %s", scope)
				}
			},
		},
		{
			name:   "System Runner",
			path:   "../../examples/runner/system-runner.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				scope, _, _ := unstructured.NestedString(obj.Object, "spec", "forProvider", "scope")
				if scope != "system" {
					t.Errorf("Expected scope system, got %s", scope)
				}
			},
		},

		// AdminUser examples
		{
			name:   "Admin Users",
			path:   "../../examples/adminuser/admin-users.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetKind() != "AdminUser" {
					t.Errorf("Expected kind AdminUser, got %s", obj.GetKind())
				}
				isAdmin, _, _ := unstructured.NestedBool(obj.Object, "spec", "forProvider", "isAdmin")
				if !isAdmin {
					t.Errorf("Expected isAdmin to be true for admin user")
				}
			},
		},
		{
			name:   "Service Accounts",
			path:   "../../examples/adminuser/service-accounts.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				visibility, _, _ := unstructured.NestedString(obj.Object, "spec", "forProvider", "visibility")
				if visibility != "private" {
					t.Errorf("Expected service account visibility to be private, got %s", visibility)
				}
			},
		},

		// BranchProtection examples
		{
			name:   "Enterprise Branch Protection",
			path:   "../../examples/branchprotection/enterprise-protection.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetKind() != "BranchProtection" {
					t.Errorf("Expected kind BranchProtection, got %s", obj.GetKind())
				}
				enableStatusCheck, _, _ := unstructured.NestedBool(obj.Object, "spec", "forProvider", "enableStatusCheck")
				if !enableStatusCheck {
					t.Errorf("Expected enterprise protection to enable status checks")
				}
			},
		},
		{
			name:   "Basic Branch Protection",
			path:   "../../examples/branchprotection/basic-protection.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				branch, _, _ := unstructured.NestedString(obj.Object, "spec", "forProvider", "branch")
				if branch == "" {
					t.Errorf("Expected branch to be specified")
				}
			},
		},

		// RepositoryKey examples
		{
			name:   "Deployment Keys",
			path:   "../../examples/repositorykey/deployment-keys.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetKind() != "RepositoryKey" {
					t.Errorf("Expected kind RepositoryKey, got %s", obj.GetKind())
				}
				key, _, _ := unstructured.NestedString(obj.Object, "spec", "forProvider", "key")
				if key == "" {
					t.Errorf("Expected SSH key to be present")
				}
			},
		},
		{
			name:   "Admin Keys",
			path:   "../../examples/repositorykey/admin-keys.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				readOnly, _, _ := unstructured.NestedBool(obj.Object, "spec", "forProvider", "readOnly")
				if readOnly {
					t.Errorf("Expected admin key to have write access (readOnly=false)")
				}
			},
		},

		// AccessToken examples
		{
			name:   "CI Automation Token",
			path:   "../../examples/accesstoken/ci-automation.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetKind() != "AccessToken" {
					t.Errorf("Expected kind AccessToken, got %s", obj.GetKind())
				}
				scopes, _, _ := unstructured.NestedSlice(obj.Object, "spec", "forProvider", "scopes")
				if len(scopes) == 0 {
					t.Errorf("Expected token to have scopes defined")
				}
			},
		},
		{
			name:   "Read-only Monitoring Token",
			path:   "../../examples/accesstoken/readonly-monitoring.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				scopes, _, _ := unstructured.NestedSlice(obj.Object, "spec", "forProvider", "scopes")
				// Verify all scopes are read-only
				for _, scope := range scopes {
					if s, ok := scope.(string); ok && len(s) > 5 && s[:5] == "write" {
						t.Errorf("Expected read-only token, but found write scope: %s", s)
					}
				}
			},
		},

		// Enterprise complete setup
		{
			name:   "Enterprise Complete Setup",
			path:   "../../examples/enterprise-complete-setup.yaml",
			scheme: scheme,
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				// This is a multi-document YAML, so we just verify it parses
				if obj.GetKind() == "" {
					t.Errorf("Expected valid Kubernetes object")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateExampleFile(t, tt)
		})
	}
}

func validateExampleFile(t *testing.T, test ExampleTest) {
	// Read the example file
	data, err := test.ReadExampleFile()
	if err != nil {
		t.Fatalf("Failed to read example file %s: %v", test.path, err)
	}

	// Parse YAML documents
	decoder := yaml.NewYAMLOrJSONDecoder(data, 4096)
	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("Failed to decode YAML from %s: %v", test.path, err)
		}

		// Skip empty objects
		if obj.GetKind() == "" {
			continue
		}

		// Skip Kubernetes native resources (Secrets, ConfigMaps, etc.)
		if obj.GetAPIVersion() == "v1" {
			continue
		}

		// Validate the object against our scheme
		if err := test.scheme.Convert(obj, obj, nil); err != nil {
			t.Fatalf("Failed to validate object against scheme: %v", err)
		}

		// Run custom validation if provided
		if test.validate != nil {
			test.validate(t, obj)
		}
	}
}

// ReadExampleFile reads the example file content
func (et *ExampleTest) ReadExampleFile() (*bytes.Buffer, error) {
	path := filepath.Join("../..", et.path)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(content), nil
}

// TestExampleCompleteness ensures all implemented resources have examples
func TestExampleCompleteness(t *testing.T) {
	requiredExamples := map[string][]string{
		"action": {
			"examples/action/ci-pipeline.yaml",
			"examples/action/security-workflow.yaml",
		},
		"runner": {
			"examples/runner/repository-runner.yaml",
			"examples/runner/organization-runners.yaml",
			"examples/runner/system-runner.yaml",
		},
		"adminuser": {
			"examples/adminuser/admin-users.yaml",
			"examples/adminuser/service-accounts.yaml",
		},
		"branchprotection": {
			"examples/branchprotection/enterprise-protection.yaml",
			"examples/branchprotection/basic-protection.yaml",
		},
		"repositorykey": {
			"examples/repositorykey/deployment-keys.yaml",
			"examples/repositorykey/admin-keys.yaml",
		},
		"accesstoken": {
			"examples/accesstoken/ci-automation.yaml",
			"examples/accesstoken/readonly-monitoring.yaml",
		},
		"repositorysecret": {
			"examples/repositorysecret/docker-registry.yaml",
			"examples/repositorysecret/api-keys.yaml",
		},
		"userkey": {
			"examples/userkey/developer-keys.yaml",
			"examples/userkey/multiple-devices.yaml",
		},
		"organizationmember": {
			"examples/organizationmember/team-membership.yaml",
		},
	}

	for resource, examples := range requiredExamples {
		for _, example := range examples {
			path := filepath.Join("../..", example)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("Missing required example for %s: %s", resource, example)
			}
		}
	}

	// Verify enterprise complete setup exists
	enterprisePath := filepath.Join("../..", "examples/enterprise-complete-setup.yaml")
	if _, err := os.Stat(enterprisePath); os.IsNotExist(err) {
		t.Errorf("Missing enterprise complete setup example")
	}
}

// TestExampleSecretRefs validates that all secret references are properly formatted
func TestExampleSecretRefs(t *testing.T) {
	exampleDirs := []string{
		"../../examples/accesstoken",
		"../../examples/adminuser", 
		"../../examples/repositorysecret",
		"../../examples/enterprise-complete-setup.yaml",
	}

	for _, dir := range exampleDirs {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				
				if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
					return nil
				}

				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				content := string(data)
				
				// Check for secret references
				if strings.Contains(content, "SecretRef") {
					// Validate secret ref has required fields
					if !strings.Contains(content, "name:") {
						t.Errorf("SecretRef in %s missing name field", path)
					}
					if !strings.Contains(content, "namespace:") {
						t.Errorf("SecretRef in %s missing namespace field", path)
					}
					if !strings.Contains(content, "key:") {
						t.Errorf("SecretRef in %s missing key field", path)
					}
				}
				
				return nil
			})
			
			if err != nil {
				t.Fatalf("Failed to walk directory %s: %v", dir, err)
			}
		})
	}
}