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

package organizationsecret

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"

	"github.com/crossplane-contrib/provider-gitea/apis/organizationsecret/v1alpha1"
	giteaclients "github.com/crossplane-contrib/provider-gitea/internal/clients"
)

// MockGiteaClient implements the Gitea client interface for testing
type MockGiteaClient struct {
	MockCreateOrganizationSecret func(ctx context.Context, org, secretName string, req *giteaclients.CreateOrganizationSecretRequest) error
	MockUpdateOrganizationSecret func(ctx context.Context, org, secretName string, req *giteaclients.CreateOrganizationSecretRequest) error
	MockDeleteOrganizationSecret func(ctx context.Context, org, secretName string) error
	MockGetOrganizationSecret    func(ctx context.Context, org, secretName string) (*giteaclients.OrganizationSecret, error)
}

func (m *MockGiteaClient) CreateOrganizationSecret(ctx context.Context, org, secretName string, req *giteaclients.CreateOrganizationSecretRequest) error {
	if m.MockCreateOrganizationSecret != nil {
		return m.MockCreateOrganizationSecret(ctx, org, secretName, req)
	}
	return nil
}

func (m *MockGiteaClient) UpdateOrganizationSecret(ctx context.Context, org, secretName string, req *giteaclients.CreateOrganizationSecretRequest) error {
	if m.MockUpdateOrganizationSecret != nil {
		return m.MockUpdateOrganizationSecret(ctx, org, secretName, req)
	}
	return nil
}

func (m *MockGiteaClient) DeleteOrganizationSecret(ctx context.Context, org, secretName string) error {
	if m.MockDeleteOrganizationSecret != nil {
		return m.MockDeleteOrganizationSecret(ctx, org, secretName)
	}
	return nil
}

func (m *MockGiteaClient) GetOrganizationSecret(ctx context.Context, org, secretName string) (*giteaclients.OrganizationSecret, error) {
	if m.MockGetOrganizationSecret != nil {
		return m.MockGetOrganizationSecret(ctx, org, secretName)
	}
	return nil, errors.New("method not allowed: 405")
}

// Implement the rest of the interface to satisfy the type
func (m *MockGiteaClient) CreateRepository(ctx context.Context, req *giteaclients.CreateRepositoryRequest) (*giteaclients.Repository, error) {
	return nil, nil
}

func (m *MockGiteaClient) CreateOrganizationRepository(ctx context.Context, org string, req *giteaclients.CreateRepositoryRequest) (*giteaclients.Repository, error) {
	return nil, nil
}

func (m *MockGiteaClient) GetRepository(ctx context.Context, owner, repo string) (*giteaclients.Repository, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateRepository(ctx context.Context, owner, repo string, req *giteaclients.UpdateRepositoryRequest) (*giteaclients.Repository, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteRepository(ctx context.Context, owner, repo string) error {
	return nil
}

func (m *MockGiteaClient) CreateOrganization(ctx context.Context, req *giteaclients.CreateOrganizationRequest) (*giteaclients.Organization, error) {
	return nil, nil
}

func (m *MockGiteaClient) GetOrganization(ctx context.Context, org string) (*giteaclients.Organization, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateOrganization(ctx context.Context, org string, req *giteaclients.UpdateOrganizationRequest) (*giteaclients.Organization, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteOrganization(ctx context.Context, org string) error {
	return nil
}

func (m *MockGiteaClient) CreateUser(ctx context.Context, req *giteaclients.CreateUserRequest) (*giteaclients.User, error) {
	return nil, nil
}

func (m *MockGiteaClient) GetUser(ctx context.Context, username string) (*giteaclients.User, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateUser(ctx context.Context, username string, req *giteaclients.UpdateUserRequest) (*giteaclients.User, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteUser(ctx context.Context, username string) error {
	return nil
}

func (m *MockGiteaClient) CreateRepositoryWebhook(ctx context.Context, owner, repo string, req *giteaclients.CreateWebhookRequest) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockGiteaClient) GetRepositoryWebhook(ctx context.Context, owner, repo string, id int64) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateRepositoryWebhook(ctx context.Context, owner, repo string, id int64, req *giteaclients.UpdateWebhookRequest) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteRepositoryWebhook(ctx context.Context, owner, repo string, id int64) error {
	return nil
}

func (m *MockGiteaClient) CreateOrganizationWebhook(ctx context.Context, org string, req *giteaclients.CreateWebhookRequest) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockGiteaClient) GetOrganizationWebhook(ctx context.Context, org string, id int64) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateOrganizationWebhook(ctx context.Context, org string, id int64, req *giteaclients.UpdateWebhookRequest) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteOrganizationWebhook(ctx context.Context, org string, id int64) error {
	return nil
}

func (m *MockGiteaClient) CreateDeployKey(ctx context.Context, owner, repo string, req *giteaclients.CreateDeployKeyRequest) (*giteaclients.DeployKey, error) {
	return nil, nil
}

func (m *MockGiteaClient) GetDeployKey(ctx context.Context, owner, repo string, id int64) (*giteaclients.DeployKey, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteDeployKey(ctx context.Context, owner, repo string, id int64) error {
	return nil
}

// Team operations - mock implementations
func (m *MockGiteaClient) GetTeam(ctx context.Context, teamID int64) (*giteaclients.Team, error) {
	return nil, nil
}

func (m *MockGiteaClient) CreateTeam(ctx context.Context, org string, req *giteaclients.CreateTeamRequest) (*giteaclients.Team, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateTeam(ctx context.Context, teamID int64, req *giteaclients.UpdateTeamRequest) (*giteaclients.Team, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteTeam(ctx context.Context, teamID int64) error {
	return nil
}

func (m *MockGiteaClient) ListOrganizationTeams(ctx context.Context, org string) ([]*giteaclients.Team, error) {
	return nil, nil
}

// Label operations - mock implementations
func (m *MockGiteaClient) GetLabel(ctx context.Context, owner, repo string, id int64) (*giteaclients.Label, error) {
	return nil, nil
}

func (m *MockGiteaClient) CreateLabel(ctx context.Context, owner, repo string, req *giteaclients.CreateLabelRequest) (*giteaclients.Label, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateLabel(ctx context.Context, owner, repo string, id int64, req *giteaclients.UpdateLabelRequest) (*giteaclients.Label, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteLabel(ctx context.Context, owner, repo string, id int64) error {
	return nil
}

func (m *MockGiteaClient) ListRepositoryLabels(ctx context.Context, owner, repo string) ([]*giteaclients.Label, error) {
	return nil, nil
}

// Repository Collaborator operations - mock implementations
func (m *MockGiteaClient) GetRepositoryCollaborator(ctx context.Context, owner, repo, username string) (*giteaclients.RepositoryCollaborator, error) {
	return nil, nil
}

func (m *MockGiteaClient) AddRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *giteaclients.AddCollaboratorRequest) error {
	return nil
}

func (m *MockGiteaClient) UpdateRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *giteaclients.UpdateCollaboratorRequest) error {
	return nil
}

func (m *MockGiteaClient) RemoveRepositoryCollaborator(ctx context.Context, owner, repo, username string) error {
	return nil
}

func (m *MockGiteaClient) ListRepositoryCollaborators(ctx context.Context, owner, repo string) ([]*giteaclients.RepositoryCollaborator, error) {
	return nil, nil
}

// Organization member methods
func (m *MockGiteaClient) GetOrganizationMember(ctx context.Context, org, username string) (*giteaclients.OrganizationMember, error) {
	return nil, nil
}

func (m *MockGiteaClient) AddOrganizationMember(ctx context.Context, org, username string, req *giteaclients.AddOrganizationMemberRequest) (*giteaclients.OrganizationMember, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateOrganizationMember(ctx context.Context, org, username string, req *giteaclients.UpdateOrganizationMemberRequest) (*giteaclients.OrganizationMember, error) {
	return nil, nil
}

func (m *MockGiteaClient) RemoveOrganizationMember(ctx context.Context, org, username string) error {
	return nil
}

// Branch protection methods
func (m *MockGiteaClient) GetBranchProtection(ctx context.Context, repo, branch string) (*giteaclients.BranchProtection, error) {
	return nil, nil
}

func (m *MockGiteaClient) CreateBranchProtection(ctx context.Context, repo, branch string, req *giteaclients.CreateBranchProtectionRequest) (*giteaclients.BranchProtection, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateBranchProtection(ctx context.Context, repo, branch string, req *giteaclients.UpdateBranchProtectionRequest) (*giteaclients.BranchProtection, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteBranchProtection(ctx context.Context, repo, branch string) error {
	return nil
}

// Repository key methods
func (m *MockGiteaClient) GetRepositoryKey(ctx context.Context, repo string, keyID int64) (*giteaclients.RepositoryKey, error) {
	return nil, nil
}

func (m *MockGiteaClient) CreateRepositoryKey(ctx context.Context, repo string, req *giteaclients.CreateRepositoryKeyRequest) (*giteaclients.RepositoryKey, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateRepositoryKey(ctx context.Context, repo string, keyID int64, req *giteaclients.UpdateRepositoryKeyRequest) (*giteaclients.RepositoryKey, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteRepositoryKey(ctx context.Context, repo string, keyID int64) error {
	return nil
}

// Access token methods
func (m *MockGiteaClient) GetAccessToken(ctx context.Context, tokenName string, tokenID int64) (*giteaclients.AccessToken, error) {
	return nil, nil
}

func (m *MockGiteaClient) CreateAccessToken(ctx context.Context, username string, req *giteaclients.CreateAccessTokenRequest) (*giteaclients.AccessToken, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateAccessToken(ctx context.Context, tokenName string, tokenID int64, req *giteaclients.UpdateAccessTokenRequest) (*giteaclients.AccessToken, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteAccessToken(ctx context.Context, tokenName string, tokenID int64) error {
	return nil
}

// Repository secret methods
func (m *MockGiteaClient) GetRepositorySecret(ctx context.Context, repo, secretName string) (*giteaclients.RepositorySecret, error) {
	return nil, nil
}

func (m *MockGiteaClient) CreateRepositorySecret(ctx context.Context, repo, secretName string, req *giteaclients.CreateRepositorySecretRequest) error {
	return nil
}

func (m *MockGiteaClient) UpdateRepositorySecret(ctx context.Context, repo, secretName string, req *giteaclients.UpdateRepositorySecretRequest) error {
	return nil
}

func (m *MockGiteaClient) DeleteRepositorySecret(ctx context.Context, repo, secretName string) error {
	return nil
}

// User key methods
func (m *MockGiteaClient) GetUserKey(ctx context.Context, username string, keyID int64) (*giteaclients.UserKey, error) {
	return nil, nil
}

func (m *MockGiteaClient) CreateUserKey(ctx context.Context, username string, req *giteaclients.CreateUserKeyRequest) (*giteaclients.UserKey, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateUserKey(ctx context.Context, username string, keyID int64, req *giteaclients.UpdateUserKeyRequest) (*giteaclients.UserKey, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteUserKey(ctx context.Context, username string, keyID int64) error {
	return nil
}

// Action methods
func (m *MockGiteaClient) GetAction(ctx context.Context, repo, workflow string) (*giteaclients.Action, error) {
	return nil, nil
}

func (m *MockGiteaClient) CreateAction(ctx context.Context, repo string, req *giteaclients.CreateActionRequest) (*giteaclients.Action, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateAction(ctx context.Context, repo, workflow string, req *giteaclients.UpdateActionRequest) (*giteaclients.Action, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteAction(ctx context.Context, repo, workflow string) error {
	return nil
}

// Runner methods
func (m *MockGiteaClient) GetRunner(ctx context.Context, scope, scopeValue string, runnerID int64) (*giteaclients.Runner, error) {
	return nil, nil
}

func (m *MockGiteaClient) CreateRunner(ctx context.Context, scope, scopeValue string, req *giteaclients.CreateRunnerRequest) (*giteaclients.Runner, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateRunner(ctx context.Context, scope, scopeValue string, runnerID int64, req *giteaclients.UpdateRunnerRequest) (*giteaclients.Runner, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteRunner(ctx context.Context, scope, scopeValue string, runnerID int64) error {
	return nil
}

// Admin user methods
func (m *MockGiteaClient) GetAdminUser(ctx context.Context, username string) (*giteaclients.AdminUser, error) {
	return nil, nil
}

func (m *MockGiteaClient) CreateAdminUser(ctx context.Context, req *giteaclients.CreateAdminUserRequest) (*giteaclients.AdminUser, error) {
	return nil, nil
}

func (m *MockGiteaClient) UpdateAdminUser(ctx context.Context, username string, req *giteaclients.UpdateAdminUserRequest) (*giteaclients.AdminUser, error) {
	return nil, nil
}

func (m *MockGiteaClient) DeleteAdminUser(ctx context.Context, username string) error {
	return nil
}

// GitHook methods
func (m *MockGiteaClient) GetGitHook(ctx context.Context, repository, hookType string) (*giteaclients.GitHook, error) {
	return nil, nil
}
func (m *MockGiteaClient) CreateGitHook(ctx context.Context, repository string, req *giteaclients.CreateGitHookRequest) (*giteaclients.GitHook, error) {
	return nil, nil
}
func (m *MockGiteaClient) UpdateGitHook(ctx context.Context, repository, hookType string, req *giteaclients.UpdateGitHookRequest) (*giteaclients.GitHook, error) {
	return nil, nil
}
func (m *MockGiteaClient) DeleteGitHook(ctx context.Context, repository, hookType string) error {
	return nil
}

// Action methods
func (m *MockGiteaClient) EnableAction(ctx context.Context, repository, workflowName string) error {
	return nil
}
func (m *MockGiteaClient) DisableAction(ctx context.Context, repository, workflowName string) error {
	return nil
}

// OrganizationSettings methods
func (m *MockGiteaClient) GetOrganizationSettings(ctx context.Context, org string) (*giteaclients.OrganizationSettings, error) {
	return nil, nil
}
func (m *MockGiteaClient) UpdateOrganizationSettings(ctx context.Context, org string, req *giteaclients.UpdateOrganizationSettingsRequest) (*giteaclients.OrganizationSettings, error) {
	return nil, nil
}

func TestOrganizationSecretObserve(t *testing.T) {
	tests := []struct {
		name     string
		cr       *v1alpha1.OrganizationSecret
		mockFunc func(*MockGiteaClient)
		want     managed.ExternalObservation
		wantErr  bool
	}{
		{
			name: "NewResourceWithoutExternalName",
			cr: &v1alpha1.OrganizationSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
				},
				Spec: v1alpha1.OrganizationSecretSpec{
					ForProvider: v1alpha1.OrganizationSecretParameters{
						Organization: "testorg",
						SecretName:   "TEST_SECRET",
					},
				},
			},
			want: managed.ExternalObservation{
				ResourceExists:   false,
				ResourceUpToDate: false,
			},
		},
		{
			name: "ExistingResourceWithExternalName",
			cr: &v1alpha1.OrganizationSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "TEST_SECRET",
					},
				},
				Spec: v1alpha1.OrganizationSecretSpec{
					ForProvider: v1alpha1.OrganizationSecretParameters{
						Organization: "testorg",
						SecretName:   "TEST_SECRET",
					},
				},
			},
			want: managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: false, // Always false since we can't verify current state
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockGiteaClient{}
			if tt.mockFunc != nil {
				tt.mockFunc(mockClient)
			}

			e := &external{
				client: mockClient,
				kube:   fake.NewClientBuilder().Build(),
			}

			got, err := e.Observe(context.Background(), tt.cr)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.ResourceExists, got.ResourceExists)
			assert.Equal(t, tt.want.ResourceUpToDate, got.ResourceUpToDate)
		})
	}
}

func TestOrganizationSecretCreate(t *testing.T) {
	tests := []struct {
		name           string
		cr             *v1alpha1.OrganizationSecret
		secret         *corev1.Secret
		mockFunc       func(*MockGiteaClient)
		wantErr        bool
		wantSecretData string
	}{
		{
			name: "CreateWithDirectData",
			cr: &v1alpha1.OrganizationSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.OrganizationSecretSpec{
					ForProvider: v1alpha1.OrganizationSecretParameters{
						Organization: "testorg",
						SecretName:   "TEST_SECRET",
						Data:         stringPtr("direct-secret-value"),
					},
				},
			},
			mockFunc: func(mc *MockGiteaClient) {
				mc.MockCreateOrganizationSecret = func(ctx context.Context, org, secretName string, req *giteaclients.CreateOrganizationSecretRequest) error {
					assert.Equal(t, "testorg", org)
					assert.Equal(t, "TEST_SECRET", secretName)
					assert.Equal(t, "direct-secret-value", req.Data)
					return nil
				}
			},
			wantSecretData: "direct-secret-value",
		},
		{
			name: "CreateWithSecretReference",
			cr: &v1alpha1.OrganizationSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.OrganizationSecretSpec{
					ForProvider: v1alpha1.OrganizationSecretParameters{
						Organization: "testorg",
						SecretName:   "TEST_SECRET",
						DataFrom: &v1alpha1.DataFromSource{
							SecretKeyRef: v1alpha1.SecretKeySelector{
								Name:      "source-secret",
								Namespace: "test-namespace",
								Key:       "password",
							},
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "source-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"password": []byte("referenced-secret-value"),
				},
			},
			mockFunc: func(mc *MockGiteaClient) {
				mc.MockCreateOrganizationSecret = func(ctx context.Context, org, secretName string, req *giteaclients.CreateOrganizationSecretRequest) error {
					assert.Equal(t, "testorg", org)
					assert.Equal(t, "TEST_SECRET", secretName)
					assert.Equal(t, "referenced-secret-value", req.Data)
					return nil
				}
			},
			wantSecretData: "referenced-secret-value",
		},
		{
			name: "CreateWithMissingSecret",
			cr: &v1alpha1.OrganizationSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.OrganizationSecretSpec{
					ForProvider: v1alpha1.OrganizationSecretParameters{
						Organization: "testorg",
						SecretName:   "TEST_SECRET",
						DataFrom: &v1alpha1.DataFromSource{
							SecretKeyRef: v1alpha1.SecretKeySelector{
								Name:      "missing-secret",
								Namespace: "test-namespace",
								Key:       "password",
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockGiteaClient{}
			if tt.mockFunc != nil {
				tt.mockFunc(mockClient)
			}

			// Create fake client with secret if provided
			clientBuilder := fake.NewClientBuilder()
			if tt.secret != nil {
				clientBuilder = clientBuilder.WithObjects(tt.secret)
			}
			kubeClient := clientBuilder.Build()

			e := &external{
				client: mockClient,
				kube:   kubeClient,
			}

			result, err := e.Create(context.Background(), tt.cr)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result.ConnectionDetails)
			assert.Equal(t, tt.wantSecretData, string(result.ConnectionDetails["data"]))

			// Verify external name was set
			assert.Equal(t, tt.cr.Spec.ForProvider.SecretName, meta.GetExternalName(tt.cr))
		})
	}
}

func TestOrganizationSecretUpdate(t *testing.T) {
	cr := &v1alpha1.OrganizationSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-namespace",
		},
		Spec: v1alpha1.OrganizationSecretSpec{
			ForProvider: v1alpha1.OrganizationSecretParameters{
				Organization: "testorg",
				SecretName:   "TEST_SECRET",
				Data:         stringPtr("updated-secret-value"),
			},
		},
	}

	mockClient := &MockGiteaClient{
		MockUpdateOrganizationSecret: func(ctx context.Context, org, secretName string, req *giteaclients.CreateOrganizationSecretRequest) error {
			assert.Equal(t, "testorg", org)
			assert.Equal(t, "TEST_SECRET", secretName)
			assert.Equal(t, "updated-secret-value", req.Data)
			return nil
		},
	}

	e := &external{
		client: mockClient,
		kube:   fake.NewClientBuilder().Build(),
	}

	result, err := e.Update(context.Background(), cr)
	require.NoError(t, err)
	assert.NotNil(t, result.ConnectionDetails)
	assert.Equal(t, "updated-secret-value", string(result.ConnectionDetails["data"]))
}

func TestOrganizationSecretDelete(t *testing.T) {
	cr := &v1alpha1.OrganizationSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-namespace",
		},
		Spec: v1alpha1.OrganizationSecretSpec{
			ForProvider: v1alpha1.OrganizationSecretParameters{
				Organization: "testorg",
				SecretName:   "TEST_SECRET",
			},
		},
	}

	mockClient := &MockGiteaClient{
		MockDeleteOrganizationSecret: func(ctx context.Context, org, secretName string) error {
			assert.Equal(t, "testorg", org)
			assert.Equal(t, "TEST_SECRET", secretName)
			return nil
		},
	}

	e := &external{
		client: mockClient,
		kube:   fake.NewClientBuilder().Build(),
	}

	_, err := e.Delete(context.Background(), cr)
	require.NoError(t, err)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
