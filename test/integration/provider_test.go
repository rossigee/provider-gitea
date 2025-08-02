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
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"

	"github.com/crossplane-contrib/provider-gitea/apis/organization/v1alpha1"
	repov1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/repository/v1alpha1"
	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
)

// TestProviderConfig tests the provider configuration
func TestProviderConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)

	// Create fake client
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	ctx := context.Background()

	// Create secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitea-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("test-token"),
		},
	}
	require.NoError(t, c.Create(ctx, secret))

	// Create provider config
	pc := &v1beta1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-config",
		},
		Spec: v1beta1.ProviderConfigSpec{
			BaseURL: "https://gitea.example.com",
			Credentials: v1beta1.ProviderCredentials{
				Source: "Secret",
				SecretRef: &v1beta1.SecretReference{
					Name:      "gitea-secret",
					Namespace: "default",
					Key:       "token",
				},
			},
		},
	}
	require.NoError(t, c.Create(ctx, pc))

	// Verify creation
	got := &v1beta1.ProviderConfig{}
	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: "test-config"}, got))
	assert.Equal(t, "https://gitea.example.com", got.Spec.BaseURL)
}

// TestRepositoryLifecycle tests the repository resource lifecycle
func TestRepositoryLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Skip if no real Gitea instance is available
	if os.Getenv("GITEA_URL") == "" {
		t.Skip("GITEA_URL not set, skipping real integration test")
	}

	scheme := runtime.NewScheme()
	_ = repov1alpha1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	ctx := context.Background()

	// Create repository
	repo := &repov1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repo",
			Annotations: map[string]string{
				meta.AnnotationKeyExternalName: "integration-test-repo",
			},
		},
		Spec: repov1alpha1.RepositorySpec{
			ForProvider: repov1alpha1.RepositoryParameters{
				Name:        "integration-test-repo",
				Owner:       stringPtr("test-org"),
				Description: stringPtr("Integration test repository"),
				Private:     boolPtr(false),
				AutoInit:    boolPtr(true),
			},
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: &xpv1.Reference{
					Name: "default",
				},
			},
		},
	}
	require.NoError(t, c.Create(ctx, repo))

	// Wait for repository to be created
	time.Sleep(2 * time.Second)

	// Update repository
	repo.Spec.ForProvider.Description = stringPtr("Updated description")
	require.NoError(t, c.Update(ctx, repo))

	// Delete repository
	require.NoError(t, c.Delete(ctx, repo))
}

// TestOrganizationLifecycle tests the organization resource lifecycle
func TestOrganizationLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Skip if no real Gitea instance is available
	if os.Getenv("GITEA_URL") == "" {
		t.Skip("GITEA_URL not set, skipping real integration test")
	}

	scheme := runtime.NewScheme()
	_ = v1alpha1.SchemeBuilder.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	ctx := context.Background()

	// Create organization
	org := &v1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-org",
			Annotations: map[string]string{
				meta.AnnotationKeyExternalName: "integration-test-org",
			},
		},
		Spec: v1alpha1.OrganizationSpec{
			ForProvider: v1alpha1.OrganizationParameters{
				Username:    "integration-test-org",
				Name:        stringPtr("Integration Test Org"),
				Description: stringPtr("Organization for integration tests"),
				Visibility:  stringPtr("public"),
			},
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: &xpv1.Reference{
					Name: "default",
				},
			},
		},
	}
	require.NoError(t, c.Create(ctx, org))

	// Wait for organization to be created
	time.Sleep(2 * time.Second)

	// Update organization
	org.Spec.ForProvider.Description = stringPtr("Updated organization description")
	require.NoError(t, c.Update(ctx, org))

	// Delete organization
	require.NoError(t, c.Delete(ctx, org))
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}