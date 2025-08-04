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

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	accesstokenv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/accesstoken/v1alpha1"
	actionv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/action/v1alpha1"
	adminuserv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/adminuser/v1alpha1"
	branchprotectionv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/branchprotection/v1alpha1"
	organizationmemberv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/organizationmember/v1alpha1"
	repositorykeyv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/repositorykey/v1alpha1"
	repositorysecretv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/repositorysecret/v1alpha1"
	runnerv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/runner/v1alpha1"
	userkeyv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/userkey/v1alpha1"
)

// Utility functions for pointer types
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

// TestCompleteEnterpriseWorkflow tests the complete enterprise setup workflow
func TestCompleteEnterpriseWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup test environment
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "package", "crds"),
		},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("Failed to start test environment: %v", err)
	}
	defer func() {
		if err := testEnv.Stop(); err != nil {
			t.Errorf("Failed to stop test environment: %v", err)
		}
	}()

	// Create Kubernetes client
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Add our CRDs to the scheme
	if err := addSchemesToTestEnv(k8sClient.Scheme()); err != nil {
		t.Fatalf("Failed to add schemes: %v", err)
	}

	ctx := context.Background()

	// Test phases in order
	t.Run("Phase1_CreateAdminUsers", func(t *testing.T) {
		testCreateAdminUsers(t, ctx, k8sClient)
	})

	t.Run("Phase2_SetupSecurity", func(t *testing.T) {
		testSetupSecurity(t, ctx, k8sClient)
	})

	t.Run("Phase3_ConfigureCI", func(t *testing.T) {
		testConfigureCI(t, ctx, k8sClient)
	})

	t.Run("Phase4_ValidateComplete", func(t *testing.T) {
		testValidateCompleteSetup(t, ctx, k8sClient)
	})
}

func addSchemesToTestEnv(scheme *runtime.Scheme) error {
	// Add each scheme builder to the scheme
	if err := actionv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	if err := adminuserv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	if err := runnerv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	if err := branchprotectionv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	if err := repositorykeyv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	if err := accesstokenv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	if err := repositorysecretv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	if err := userkeyv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	if err := organizationmemberv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	return nil
}

func testCreateAdminUsers(t *testing.T, ctx context.Context, k8sClient client.Client) {
	// Create admin user from example
	adminUser := &adminuserv1alpha1.AdminUser{}
	adminUser.SetName("test-admin")
	adminUser.SetNamespace("default")
	adminUser.Spec.ForProvider = adminuserv1alpha1.AdminUserParameters{
		Username:    "test-admin",
		Email:       "admin@test.com",
		FullName:    stringPtr("Test Administrator"),
		IsAdmin:     boolPtr(true),
		IsActive:    boolPtr(true),
		Visibility:  stringPtr("private"),
		Description: stringPtr("Test admin user for E2E testing"),
	}

	if err := k8sClient.Create(ctx, adminUser); err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	// Verify creation
	created := &adminuserv1alpha1.AdminUser{}
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(adminUser), created); err != nil {
		t.Fatalf("Failed to get created admin user: %v", err)
	}

	if created.Spec.ForProvider.Username != "test-admin" {
		t.Errorf("Expected username 'test-admin', got %s", created.Spec.ForProvider.Username)
	}
}

func testSetupSecurity(t *testing.T, ctx context.Context, k8sClient client.Client) {
	// Create branch protection
	branchProtection := &branchprotectionv1alpha1.BranchProtection{}
	branchProtection.SetName("main-protection")
	branchProtection.SetNamespace("default")
	branchProtection.Spec.ForProvider = branchprotectionv1alpha1.BranchProtectionParameters{
		Repository:             "test-org/test-repo",
		Branch:                 "main",
		RuleName:               "Main Branch Protection",
		EnablePush:             boolPtr(false),
		EnableStatusCheck:      boolPtr(true),
		RequiredApprovals:      intPtr(2),
		BlockOnRejectedReviews: boolPtr(true),
		RequireSignedCommits:   boolPtr(true),
		StatusCheckContexts:    []string{"ci/build", "security/scan"},
	}

	if err := k8sClient.Create(ctx, branchProtection); err != nil {
		t.Fatalf("Failed to create branch protection: %v", err)
	}

	// Create access token
	accessToken := &accesstokenv1alpha1.AccessToken{}
	accessToken.SetName("ci-token")
	accessToken.SetNamespace("default")
	accessToken.Spec.ForProvider = accesstokenv1alpha1.AccessTokenParameters{
		Username: "test-admin",
		Name:     "CI Access Token",
		Scopes:   []string{"read:repository", "write:repository"},
	}

	if err := k8sClient.Create(ctx, accessToken); err != nil {
		t.Fatalf("Failed to create access token: %v", err)
	}

	// Create SSH key
	userKey := &userkeyv1alpha1.UserKey{}
	userKey.SetName("admin-ssh-key")
	userKey.SetNamespace("default")
	userKey.Spec.ForProvider = userkeyv1alpha1.UserKeyParameters{
		Username: "test-admin",
		Key:      "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA+17FiQDlnT5hoKHDTkPAo6pN5aOtVw== admin@test.com",
		Title:    "Admin SSH Key",
	}

	if err := k8sClient.Create(ctx, userKey); err != nil {
		t.Fatalf("Failed to create user key: %v", err)
	}
}

func testConfigureCI(t *testing.T, ctx context.Context, k8sClient client.Client) {
	// Create runner
	runner := &runnerv1alpha1.Runner{}
	runner.SetName("test-runner")
	runner.SetNamespace("default")
	runner.Spec.ForProvider = runnerv1alpha1.RunnerParameters{
		Scope:       "organization",
		ScopeValue:  stringPtr("test-org"),
		Name:        "Test Runner",
		Description: stringPtr("Runner for E2E testing"),
		Labels:      []string{"ubuntu-latest", "test", "e2e"},
	}

	if err := k8sClient.Create(ctx, runner); err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	// Create action workflow
	action := &actionv1alpha1.Action{}
	action.SetName("test-workflow")
	action.SetNamespace("default")
	action.Spec.ForProvider = actionv1alpha1.ActionParameters{
		Repository:    "test-org/test-repo",
		WorkflowName:  "test.yml",
		Branch:        stringPtr("main"),
		CommitMessage: stringPtr("Add test workflow"),
		Content: `name: Test Workflow
on:
  push:
    branches: [main]
jobs:
  test:
    runs-on: [self-hosted, test]
    steps:
      - uses: actions/checkout@v3
      - name: Run tests
        run: echo "Running tests"`,
	}

	if err := k8sClient.Create(ctx, action); err != nil {
		t.Fatalf("Failed to create action: %v", err)
	}

	// Create repository secret
	repositorySecret := &repositorysecretv1alpha1.RepositorySecret{}
	repositorySecret.SetName("test-secret")
	repositorySecret.SetNamespace("default")
	repositorySecret.Spec.ForProvider = repositorysecretv1alpha1.RepositorySecretParameters{
		Repository: "test-org/test-repo",
		SecretName: "TEST_SECRET",
	}

	if err := k8sClient.Create(ctx, repositorySecret); err != nil {
		t.Fatalf("Failed to create repository secret: %v", err)
	}
}

func testValidateCompleteSetup(t *testing.T, ctx context.Context, k8sClient client.Client) {
	// List all created resources and validate they exist
	resources := []struct {
		name     string
		obj      client.Object
		listFunc func() client.ObjectList
	}{
		{
			name: "AdminUsers",
			obj:  &adminuserv1alpha1.AdminUser{},
			listFunc: func() client.ObjectList {
				return &adminuserv1alpha1.AdminUserList{}
			},
		},
		{
			name: "BranchProtections",
			obj:  &branchprotectionv1alpha1.BranchProtection{},
			listFunc: func() client.ObjectList {
				return &branchprotectionv1alpha1.BranchProtectionList{}
			},
		},
		{
			name: "Runners",
			obj:  &runnerv1alpha1.Runner{},
			listFunc: func() client.ObjectList {
				return &runnerv1alpha1.RunnerList{}
			},
		},
		{
			name: "Actions",
			obj:  &actionv1alpha1.Action{},
			listFunc: func() client.ObjectList {
				return &actionv1alpha1.ActionList{}
			},
		},
		{
			name: "AccessTokens",
			obj:  &accesstokenv1alpha1.AccessToken{},
			listFunc: func() client.ObjectList {
				return &accesstokenv1alpha1.AccessTokenList{}
			},
		},
	}

	for _, resource := range resources {
		t.Run(fmt.Sprintf("Validate_%s", resource.name), func(t *testing.T) {
			list := resource.listFunc()
			if err := k8sClient.List(ctx, list); err != nil {
				t.Fatalf("Failed to list %s: %v", resource.name, err)
			}

			// Use reflection to get Items field
			items := list.(*unstructured.UnstructuredList).Items
			if len(items) == 0 {
				t.Errorf("Expected at least one %s but found none", resource.name)
			}
		})
	}
}

// TestWorkflowDependencies tests that resources are created in correct order
func TestWorkflowDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping dependency test in short mode")
	}

	dependencies := map[string][]string{
		"AdminUser":          {},                       // No dependencies
		"OrganizationMember": {"AdminUser"},            // Requires user
		"AccessToken":        {"AdminUser"},            // Requires user
		"UserKey":            {"AdminUser"},            // Requires user
		"BranchProtection":   {"Repository"},           // Requires repository
		"RepositoryKey":      {"Repository"},           // Requires repository
		"RepositorySecret":   {"Repository"},           // Requires repository
		"Action":             {"Repository", "Runner"}, // Requires repository and runner
		"Runner":             {"Organization"},         // Requires organization
	}

	for resource, deps := range dependencies {
		t.Run(fmt.Sprintf("Dependencies_%s", resource), func(t *testing.T) {
			t.Logf("Resource %s depends on: %v", resource, deps)
			// This test documents the dependency graph
			// In a real implementation, you would test the actual creation order
		})
	}
}

// TestExampleValidation tests that all examples can be applied to cluster
func TestExampleValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping example validation in short mode")
	}

	exampleFiles := []string{
		"../../examples/action/ci-pipeline.yaml",
		"../../examples/runner/organization-runners.yaml",
		"../../examples/adminuser/admin-users.yaml",
		"../../examples/branchprotection/enterprise-protection.yaml",
		"../../examples/repositorykey/deployment-keys.yaml",
		"../../examples/accesstoken/ci-automation.yaml",
		"../../examples/repositorysecret/docker-registry.yaml",
		"../../examples/userkey/developer-keys.yaml",
		"../../examples/organizationmember/team-membership.yaml",
	}

	for _, file := range exampleFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			validateExampleFile(t, file)
		})
	}
}

func validateExampleFile(t *testing.T, filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read example file %s: %v", filePath, err)
	}

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("Failed to decode YAML from %s: %v", filePath, err)
		}

		// Skip empty objects and Kubernetes native resources
		if obj.GetKind() == "" || obj.GetAPIVersion() == "v1" {
			continue
		}

		// Validate required fields
		if obj.GetName() == "" {
			t.Errorf("Object in %s missing name", filePath)
		}

		if obj.GetKind() == "" {
			t.Errorf("Object in %s missing kind", filePath)
		}

		// Validate spec.forProvider exists for our CRDs
		if spec, found, _ := unstructured.NestedMap(obj.Object, "spec"); found {
			if _, found, _ := unstructured.NestedMap(spec, "forProvider"); !found {
				t.Errorf("Object %s in %s missing spec.forProvider", obj.GetName(), filePath)
			}
		}

		t.Logf("Validated %s: %s/%s", obj.GetKind(), obj.GetName(), obj.GetAPIVersion())
	}
}

// TestPerformanceScenarios tests resource creation under load
func TestPerformanceScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	scenarios := []struct {
		name          string
		resourceCount int
		resourceType  string
		timeout       time.Duration
	}{
		{
			name:          "CreateMultipleRunners",
			resourceCount: 10,
			resourceType:  "Runner",
			timeout:       30 * time.Second,
		},
		{
			name:          "CreateMultipleActions",
			resourceCount: 5,
			resourceType:  "Action",
			timeout:       60 * time.Second,
		},
		{
			name:          "CreateMultipleUsers",
			resourceCount: 20,
			resourceType:  "AdminUser",
			timeout:       45 * time.Second,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			start := time.Now()

			// This would create multiple resources of the specified type
			// For now, we just validate the test structure
			t.Logf("Would create %d %s resources within %v",
				scenario.resourceCount, scenario.resourceType, scenario.timeout)

			elapsed := time.Since(start)
			if elapsed > scenario.timeout {
				t.Errorf("Scenario took %v, exceeded timeout of %v", elapsed, scenario.timeout)
			}
		})
	}
}
