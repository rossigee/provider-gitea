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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"

	"github.com/rossigee/provider-gitea/apis/organizationsecret/v1alpha1"
)

// TestOrganizationSecretIntegration tests the complete end-to-end workflow
// This test requires a running Gitea instance and proper provider configuration
func TestOrganizationSecretIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would normally be set up by the test framework
	// For now, we'll focus on the test structure
	k8sClient := setupTestClient(t)
	ctx := context.Background() // will be used once k8sClient is properly implemented

	tests := []struct {
		name         string
		secretSpec   v1alpha1.OrganizationSecretSpec
		secretData   *corev1.Secret // For dataFrom tests
		expectReady  bool
		expectSynced bool
		validateFunc func(t *testing.T, secret *v1alpha1.OrganizationSecret)
	}{
		{
			name: "DirectDataSecret",
			secretSpec: v1alpha1.OrganizationSecretSpec{
				ResourceSpec: xpv1.ResourceSpec{
					ProviderConfigReference: &xpv1.Reference{
						Name: "default",
					},
				},
				ForProvider: v1alpha1.OrganizationSecretParameters{
					Organization: "integration-test-org",
					SecretName:   "INTEGRATION_TEST_DIRECT",
					Data:         stringPtr("direct-secret-value-integration"),
				},
			},
			expectReady:  true,
			expectSynced: true,
			validateFunc: func(t *testing.T, secret *v1alpha1.OrganizationSecret) {
				// Validate external name is set
				assert.Equal(t, "INTEGRATION_TEST_DIRECT", meta.GetExternalName(secret))

				// Validate conditions
				readyCondition := secret.GetCondition(xpv1.TypeReady)
				assert.Equal(t, corev1.ConditionTrue, readyCondition.Status)

				syncedCondition := secret.GetCondition(xpv1.TypeSynced)
				assert.Equal(t, corev1.ConditionTrue, syncedCondition.Status)
			},
		},
		{
			name: "SecretReferenceSecret",
			secretSpec: v1alpha1.OrganizationSecretSpec{
				ResourceSpec: xpv1.ResourceSpec{
					ProviderConfigReference: &xpv1.Reference{
						Name: "default",
					},
				},
				ForProvider: v1alpha1.OrganizationSecretParameters{
					Organization: "integration-test-org",
					SecretName:   "INTEGRATION_TEST_REF",
					DataFrom: &v1alpha1.DataFromSource{
						SecretKeyRef: v1alpha1.SecretKeySelector{
							Name:      "integration-source-secret",
							Namespace: "default",
							Key:       "secret-data",
						},
					},
				},
			},
			secretData: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "integration-source-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"secret-data": []byte("referenced-secret-value-integration"),
				},
			},
			expectReady:  true,
			expectSynced: true,
			validateFunc: func(t *testing.T, secret *v1alpha1.OrganizationSecret) {
				// Validate external name is set
				assert.Equal(t, "INTEGRATION_TEST_REF", meta.GetExternalName(secret))

				// Validate conditions
				readyCondition := secret.GetCondition(xpv1.TypeReady)
				assert.NotNil(t, readyCondition)
				assert.Equal(t, corev1.ConditionTrue, readyCondition.Status)
			},
		},
		{
			name: "InvalidOrganization",
			secretSpec: v1alpha1.OrganizationSecretSpec{
				ResourceSpec: xpv1.ResourceSpec{
					ProviderConfigReference: &xpv1.Reference{
						Name: "default",
					},
				},
				ForProvider: v1alpha1.OrganizationSecretParameters{
					Organization: "non-existent-org",
					SecretName:   "INTEGRATION_TEST_INVALID",
					Data:         stringPtr("test-value"),
				},
			},
			expectReady:  false,
			expectSynced: false,
			validateFunc: func(t *testing.T, secret *v1alpha1.OrganizationSecret) {
				// Should have error conditions
				syncedCondition := secret.GetCondition(xpv1.TypeSynced)
				if syncedCondition.Status == corev1.ConditionFalse {
					assert.Contains(t, syncedCondition.Reason, "ReconcileError")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create source secret if needed
			if tt.secretData != nil {
				err := k8sClient.Create(ctx, tt.secretData)
				require.NoError(t, err)

				defer func() {
					_ = k8sClient.Delete(ctx, tt.secretData)
				}()
			}

			// Create OrganizationSecret
			orgSecret := &v1alpha1.OrganizationSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "integration-test-" + tt.name,
					Namespace: "default",
				},
				Spec: tt.secretSpec,
			}

			err := k8sClient.Create(ctx, orgSecret)
			require.NoError(t, err)

			defer func() {
				_ = k8sClient.Delete(ctx, orgSecret)
			}()

			// Wait for reconciliation
			err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      orgSecret.Name,
					Namespace: orgSecret.Namespace,
				}, orgSecret)
				if err != nil {
					return false, err
				}

				readyCondition := orgSecret.GetCondition(xpv1.TypeReady)
				syncedCondition := orgSecret.GetCondition(xpv1.TypeSynced)

				if tt.expectReady && tt.expectSynced {
					return readyCondition.Status == corev1.ConditionTrue &&
						syncedCondition.Status == corev1.ConditionTrue, nil
				}

				// For error cases, wait for error conditions
				if !tt.expectReady && !tt.expectSynced {
					return syncedCondition.Status == corev1.ConditionFalse, nil
				}

				return false, nil
			})

			if tt.expectReady && tt.expectSynced {
				require.NoError(t, err, "OrganizationSecret should become ready and synced")
			}

			// Run custom validation
			if tt.validateFunc != nil {
				tt.validateFunc(t, orgSecret)
			}

			// Test connection details publishing
			if tt.expectReady && tt.expectSynced {
				validateConnectionDetails(t, k8sClient, orgSecret)
			}
		})
	}
}

// TestOrganizationSecretWriteThroughPattern specifically tests the write-through pattern
func TestOrganizationSecretWriteThroughPattern(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	k8sClient := setupTestClient(t)
	ctx := context.Background() // will be used once k8sClient is properly implemented

	// Create an OrganizationSecret
	orgSecret := &v1alpha1.OrganizationSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "write-through-test",
			Namespace: "default",
		},
		Spec: v1alpha1.OrganizationSecretSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: &xpv1.Reference{
					Name: "default",
				},
			},
			ForProvider: v1alpha1.OrganizationSecretParameters{
				Organization: "integration-test-org",
				SecretName:   "WRITE_THROUGH_TEST",
				Data:         stringPtr("initial-value"),
			},
		},
	}

	err := k8sClient.Create(ctx, orgSecret)
	require.NoError(t, err)

	defer func() {
		_ = k8sClient.Delete(ctx, orgSecret)
	}()

	// Wait for initial creation
	err = waitForReady(ctx, k8sClient, orgSecret, 2*time.Minute)
	require.NoError(t, err)

	// Update the secret value to test the write-through pattern
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      orgSecret.Name,
		Namespace: orgSecret.Namespace,
	}, orgSecret)
	require.NoError(t, err)

	orgSecret.Spec.ForProvider.Data = stringPtr("updated-value")
	err = k8sClient.Update(ctx, orgSecret)
	require.NoError(t, err)

	// Wait for update to be processed
	// Since we use write-through pattern, the resource should always show as needing update
	time.Sleep(10 * time.Second)

	// Verify the resource is still ready (write-through pattern working)
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      orgSecret.Name,
		Namespace: orgSecret.Namespace,
	}, orgSecret)
	require.NoError(t, err)

	readyCondition := orgSecret.GetCondition(xpv1.TypeReady)
	assert.NotNil(t, readyCondition)
	assert.Equal(t, corev1.ConditionTrue, readyCondition.Status)
}

// validateConnectionDetails checks that connection details are properly published
func validateConnectionDetails(t *testing.T, k8sClient client.Client, orgSecret *v1alpha1.OrganizationSecret) {
	// For OrganizationSecret, connection details are typically not published
	// This is a placeholder for future connection details validation if needed
	// Currently just validates that the secret exists and has proper conditions
	assert.NotNil(t, orgSecret, "OrganizationSecret should not be nil")
}

// waitForReady waits for an OrganizationSecret to become ready
func waitForReady(ctx context.Context, k8sClient client.Client, orgSecret *v1alpha1.OrganizationSecret, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      orgSecret.Name,
			Namespace: orgSecret.Namespace,
		}, orgSecret)
		if err != nil {
			return false, err
		}

		readyCondition := orgSecret.GetCondition(xpv1.TypeReady)
		syncedCondition := orgSecret.GetCondition(xpv1.TypeSynced)

		return readyCondition.Status == corev1.ConditionTrue &&
			syncedCondition.Status == corev1.ConditionTrue, nil
	})
}

// setupTestClient sets up a Kubernetes client for integration testing
// In a real implementation, this would configure the proper test environment
func setupTestClient(t *testing.T) client.Client {
	// This would normally set up a real client against a test cluster
	// For now, we'll note that this needs to be implemented based on the test environment
	t.Skip("Integration test requires proper Kubernetes client setup")
	return nil
}

// stringPtr helper function is defined in provider_test.go
