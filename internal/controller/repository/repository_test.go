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

package repository

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/repository/v1alpha1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

// MockClient is a mock implementation of the Gitea client
type MockClient struct {
	mock.Mock
}

func (m *MockClient) GetRepository(ctx context.Context, owner, name string) (*giteaclients.Repository, error) {
	args := m.Called(ctx, owner, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*giteaclients.Repository), args.Error(1)
}

func (m *MockClient) CreateRepository(ctx context.Context, req *giteaclients.CreateRepositoryRequest) (*giteaclients.Repository, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*giteaclients.Repository), args.Error(1)
}

func (m *MockClient) CreateOrganizationRepository(ctx context.Context, org string, req *giteaclients.CreateRepositoryRequest) (*giteaclients.Repository, error) {
	args := m.Called(ctx, org, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*giteaclients.Repository), args.Error(1)
}

func (m *MockClient) UpdateRepository(ctx context.Context, owner, name string, req *giteaclients.UpdateRepositoryRequest) (*giteaclients.Repository, error) {
	args := m.Called(ctx, owner, name, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*giteaclients.Repository), args.Error(1)
}

func (m *MockClient) DeleteRepository(ctx context.Context, owner, name string) error {
	args := m.Called(ctx, owner, name)
	return args.Error(0)
}

// Implement other required methods as stubs
func (m *MockClient) GetOrganization(ctx context.Context, name string) (*giteaclients.Organization, error) {
	return nil, nil
}

func (m *MockClient) CreateOrganization(ctx context.Context, req *giteaclients.CreateOrganizationRequest) (*giteaclients.Organization, error) {
	return nil, nil
}

func (m *MockClient) UpdateOrganization(ctx context.Context, name string, req *giteaclients.UpdateOrganizationRequest) (*giteaclients.Organization, error) {
	return nil, nil
}

func (m *MockClient) DeleteOrganization(ctx context.Context, name string) error {
	return nil
}

func (m *MockClient) GetUser(ctx context.Context, username string) (*giteaclients.User, error) {
	return nil, nil
}

func (m *MockClient) CreateUser(ctx context.Context, req *giteaclients.CreateUserRequest) (*giteaclients.User, error) {
	return nil, nil
}

func (m *MockClient) UpdateUser(ctx context.Context, username string, req *giteaclients.UpdateUserRequest) (*giteaclients.User, error) {
	return nil, nil
}

func (m *MockClient) DeleteUser(ctx context.Context, username string) error {
	return nil
}

func (m *MockClient) GetRepositoryWebhook(ctx context.Context, owner, repo string, id int64) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockClient) CreateRepositoryWebhook(ctx context.Context, owner, repo string, req *giteaclients.CreateWebhookRequest) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockClient) UpdateRepositoryWebhook(ctx context.Context, owner, repo string, id int64, req *giteaclients.UpdateWebhookRequest) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockClient) DeleteRepositoryWebhook(ctx context.Context, owner, repo string, id int64) error {
	return nil
}

func (m *MockClient) GetDeployKey(ctx context.Context, owner, repo string, id int64) (*giteaclients.DeployKey, error) {
	return nil, nil
}

func (m *MockClient) CreateDeployKey(ctx context.Context, owner, repo string, req *giteaclients.CreateDeployKeyRequest) (*giteaclients.DeployKey, error) {
	return nil, nil
}

func (m *MockClient) DeleteDeployKey(ctx context.Context, owner, repo string, id int64) error {
	return nil
}

// Organization webhook operations
func (m *MockClient) CreateOrganizationWebhook(ctx context.Context, org string, req *giteaclients.CreateWebhookRequest) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockClient) GetOrganizationWebhook(ctx context.Context, org string, id int64) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockClient) UpdateOrganizationWebhook(ctx context.Context, org string, id int64, req *giteaclients.UpdateWebhookRequest) (*giteaclients.Webhook, error) {
	return nil, nil
}

func (m *MockClient) DeleteOrganizationWebhook(ctx context.Context, org string, id int64) error {
	return nil
}

// Organization secret operations
func (m *MockClient) GetOrganizationSecret(ctx context.Context, org, secretName string) (*giteaclients.OrganizationSecret, error) {
	return nil, nil
}

func (m *MockClient) CreateOrganizationSecret(ctx context.Context, org, secretName string, req *giteaclients.CreateOrganizationSecretRequest) error {
	return nil
}

func (m *MockClient) UpdateOrganizationSecret(ctx context.Context, org, secretName string, req *giteaclients.CreateOrganizationSecretRequest) error {
	return nil
}

func (m *MockClient) DeleteOrganizationSecret(ctx context.Context, org, secretName string) error {
	return nil
}

// Team operations - mock implementations
func (m *MockClient) GetTeam(ctx context.Context, teamID int64) (*giteaclients.Team, error) {
	return nil, nil
}

func (m *MockClient) CreateTeam(ctx context.Context, org string, req *giteaclients.CreateTeamRequest) (*giteaclients.Team, error) {
	return nil, nil
}

func (m *MockClient) UpdateTeam(ctx context.Context, teamID int64, req *giteaclients.UpdateTeamRequest) (*giteaclients.Team, error) {
	return nil, nil
}

func (m *MockClient) DeleteTeam(ctx context.Context, teamID int64) error {
	return nil
}

func (m *MockClient) ListOrganizationTeams(ctx context.Context, org string) ([]*giteaclients.Team, error) {
	return nil, nil
}

// Label operations - mock implementations
func (m *MockClient) GetLabel(ctx context.Context, owner, repo string, id int64) (*giteaclients.Label, error) {
	return nil, nil
}

func (m *MockClient) CreateLabel(ctx context.Context, owner, repo string, req *giteaclients.CreateLabelRequest) (*giteaclients.Label, error) {
	return nil, nil
}

func (m *MockClient) UpdateLabel(ctx context.Context, owner, repo string, id int64, req *giteaclients.UpdateLabelRequest) (*giteaclients.Label, error) {
	return nil, nil
}

func (m *MockClient) DeleteLabel(ctx context.Context, owner, repo string, id int64) error {
	return nil
}

func (m *MockClient) ListRepositoryLabels(ctx context.Context, owner, repo string) ([]*giteaclients.Label, error) {
	return nil, nil
}

// Repository Collaborator operations - mock implementations
func (m *MockClient) GetRepositoryCollaborator(ctx context.Context, owner, repo, username string) (*giteaclients.RepositoryCollaborator, error) {
	return nil, nil
}

func (m *MockClient) AddRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *giteaclients.AddCollaboratorRequest) error {
	return nil
}

func (m *MockClient) UpdateRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *giteaclients.UpdateCollaboratorRequest) error {
	return nil
}

func (m *MockClient) RemoveRepositoryCollaborator(ctx context.Context, owner, repo, username string) error {
	return nil
}

func (m *MockClient) ListRepositoryCollaborators(ctx context.Context, owner, repo string) ([]*giteaclients.RepositoryCollaborator, error) {
	return nil, nil
}

// Organization member methods
func (m *MockClient) GetOrganizationMember(ctx context.Context, org, username string) (*giteaclients.OrganizationMember, error) {
	return nil, nil
}

func (m *MockClient) AddOrganizationMember(ctx context.Context, org, username string, req *giteaclients.AddOrganizationMemberRequest) (*giteaclients.OrganizationMember, error) {
	return nil, nil
}

func (m *MockClient) UpdateOrganizationMember(ctx context.Context, org, username string, req *giteaclients.UpdateOrganizationMemberRequest) (*giteaclients.OrganizationMember, error) {
	return nil, nil
}

func (m *MockClient) RemoveOrganizationMember(ctx context.Context, org, username string) error {
	return nil
}

// Branch protection methods
func (m *MockClient) GetBranchProtection(ctx context.Context, repo, branch string) (*giteaclients.BranchProtection, error) {
	return nil, nil
}

func (m *MockClient) CreateBranchProtection(ctx context.Context, repo, branch string, req *giteaclients.CreateBranchProtectionRequest) (*giteaclients.BranchProtection, error) {
	return nil, nil
}

func (m *MockClient) UpdateBranchProtection(ctx context.Context, repo, branch string, req *giteaclients.UpdateBranchProtectionRequest) (*giteaclients.BranchProtection, error) {
	return nil, nil
}

func (m *MockClient) DeleteBranchProtection(ctx context.Context, repo, branch string) error {
	return nil
}

// Repository key methods
func (m *MockClient) GetRepositoryKey(ctx context.Context, repo string, keyID int64) (*giteaclients.RepositoryKey, error) {
	return nil, nil
}

func (m *MockClient) CreateRepositoryKey(ctx context.Context, repo string, req *giteaclients.CreateRepositoryKeyRequest) (*giteaclients.RepositoryKey, error) {
	return nil, nil
}

func (m *MockClient) UpdateRepositoryKey(ctx context.Context, repo string, keyID int64, req *giteaclients.UpdateRepositoryKeyRequest) (*giteaclients.RepositoryKey, error) {
	return nil, nil
}

func (m *MockClient) DeleteRepositoryKey(ctx context.Context, repo string, keyID int64) error {
	return nil
}

// Access token methods
func (m *MockClient) GetAccessToken(ctx context.Context, tokenName string, tokenID int64) (*giteaclients.AccessToken, error) {
	return nil, nil
}

func (m *MockClient) CreateAccessToken(ctx context.Context, username string, req *giteaclients.CreateAccessTokenRequest) (*giteaclients.AccessToken, error) {
	return nil, nil
}

func (m *MockClient) UpdateAccessToken(ctx context.Context, tokenName string, tokenID int64, req *giteaclients.UpdateAccessTokenRequest) (*giteaclients.AccessToken, error) {
	return nil, nil
}

func (m *MockClient) DeleteAccessToken(ctx context.Context, tokenName string, tokenID int64) error {
	return nil
}

// Repository secret methods
func (m *MockClient) GetRepositorySecret(ctx context.Context, repo, secretName string) (*giteaclients.RepositorySecret, error) {
	return nil, nil
}

func (m *MockClient) CreateRepositorySecret(ctx context.Context, repo, secretName string, req *giteaclients.CreateRepositorySecretRequest) error {
	return nil
}

func (m *MockClient) UpdateRepositorySecret(ctx context.Context, repo, secretName string, req *giteaclients.UpdateRepositorySecretRequest) error {
	return nil
}

func (m *MockClient) DeleteRepositorySecret(ctx context.Context, repo, secretName string) error {
	return nil
}

// User key methods
func (m *MockClient) GetUserKey(ctx context.Context, username string, keyID int64) (*giteaclients.UserKey, error) {
	return nil, nil
}

func (m *MockClient) CreateUserKey(ctx context.Context, username string, req *giteaclients.CreateUserKeyRequest) (*giteaclients.UserKey, error) {
	return nil, nil
}

func (m *MockClient) UpdateUserKey(ctx context.Context, username string, keyID int64, req *giteaclients.UpdateUserKeyRequest) (*giteaclients.UserKey, error) {
	return nil, nil
}

func (m *MockClient) DeleteUserKey(ctx context.Context, username string, keyID int64) error {
	return nil
}

// Action methods
func (m *MockClient) GetAction(ctx context.Context, repo, workflow string) (*giteaclients.Action, error) {
	return nil, nil
}

func (m *MockClient) CreateAction(ctx context.Context, repo string, req *giteaclients.CreateActionRequest) (*giteaclients.Action, error) {
	return nil, nil
}

func (m *MockClient) UpdateAction(ctx context.Context, repo, workflow string, req *giteaclients.UpdateActionRequest) (*giteaclients.Action, error) {
	return nil, nil
}

func (m *MockClient) DeleteAction(ctx context.Context, repo, workflow string) error {
	return nil
}

// Runner methods
func (m *MockClient) GetRunner(ctx context.Context, scope, scopeValue string, runnerID int64) (*giteaclients.Runner, error) {
	return nil, nil
}

func (m *MockClient) CreateRunner(ctx context.Context, scope, scopeValue string, req *giteaclients.CreateRunnerRequest) (*giteaclients.Runner, error) {
	return nil, nil
}

func (m *MockClient) UpdateRunner(ctx context.Context, scope, scopeValue string, runnerID int64, req *giteaclients.UpdateRunnerRequest) (*giteaclients.Runner, error) {
	return nil, nil
}

func (m *MockClient) DeleteRunner(ctx context.Context, scope, scopeValue string, runnerID int64) error {
	return nil
}

// Admin user methods
func (m *MockClient) GetAdminUser(ctx context.Context, username string) (*giteaclients.AdminUser, error) {
	return nil, nil
}

func (m *MockClient) CreateAdminUser(ctx context.Context, req *giteaclients.CreateAdminUserRequest) (*giteaclients.AdminUser, error) {
	return nil, nil
}

func (m *MockClient) UpdateAdminUser(ctx context.Context, username string, req *giteaclients.UpdateAdminUserRequest) (*giteaclients.AdminUser, error) {
	return nil, nil
}

func (m *MockClient) DeleteAdminUser(ctx context.Context, username string) error {
	return nil
}

// GitHook methods
func (m *MockClient) GetGitHook(ctx context.Context, repository, hookType string) (*giteaclients.GitHook, error) {
	return nil, nil
}
func (m *MockClient) CreateGitHook(ctx context.Context, repository string, req *giteaclients.CreateGitHookRequest) (*giteaclients.GitHook, error) {
	return nil, nil
}
func (m *MockClient) UpdateGitHook(ctx context.Context, repository, hookType string, req *giteaclients.UpdateGitHookRequest) (*giteaclients.GitHook, error) {
	return nil, nil
}
func (m *MockClient) DeleteGitHook(ctx context.Context, repository, hookType string) error {
	return nil
}

// Action methods
func (m *MockClient) EnableAction(ctx context.Context, repository, workflowName string) error {
	return nil
}
func (m *MockClient) DisableAction(ctx context.Context, repository, workflowName string) error {
	return nil
}

// OrganizationSettings methods
func (m *MockClient) GetOrganizationSettings(ctx context.Context, org string) (*giteaclients.OrganizationSettings, error) {
	return nil, nil
}
func (m *MockClient) UpdateOrganizationSettings(ctx context.Context, org string, req *giteaclients.UpdateOrganizationSettingsRequest) (*giteaclients.OrganizationSettings, error) {
	return nil, nil
}

func TestObserve(t *testing.T) {
	tests := []struct {
		name    string
		mg      resource.Managed
		setup   func(*MockClient)
		want    managed.ExternalObservation
		wantErr bool
	}{
		{
			name: "repository exists",
			mg: func() resource.Managed {
				repo := &v1alpha1.Repository{
					Spec: v1alpha1.RepositorySpec{
						ForProvider: v1alpha1.RepositoryParameters{
							Name:  "test-repo",
							Owner: stringPtr("test-owner"),
						},
					},
				}
				meta.SetExternalName(repo, "test-repo")
				return repo
			}(),
			setup: func(mc *MockClient) {
				mc.On("GetRepository", mock.Anything, "test-owner", "test-repo").
					Return(&giteaclients.Repository{
						ID:          123,
						Name:        "test-repo",
						FullName:    "test-owner/test-repo",
						Description: "Test description",
						Private:     false,
						Empty:       false,
						Size:        1024,
						HTMLURL:     "https://gitea.example.com/test-owner/test-repo",
						SSHURL:      "git@gitea.example.com:test-owner/test-repo.git",
						CloneURL:    "https://gitea.example.com/test-owner/test-repo.git",
						Language:    "Go",
						CreatedAt:   "2024-01-01T00:00:00Z",
						UpdatedAt:   "2024-01-02T00:00:00Z",
					}, nil)
			},
			want: managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: true,
			},
			wantErr: false,
		},
		{
			name: "repository does not exist",
			mg: func() resource.Managed {
				repo := &v1alpha1.Repository{
					Spec: v1alpha1.RepositorySpec{
						ForProvider: v1alpha1.RepositoryParameters{
							Name:  "test-repo",
							Owner: stringPtr("test-owner"),
						},
					},
				}
				meta.SetExternalName(repo, "test-repo")
				return repo
			}(),
			setup: func(mc *MockClient) {
				mc.On("GetRepository", mock.Anything, "test-owner", "test-repo").
					Return(nil, errors.New("not found"))
			},
			want: managed.ExternalObservation{
				ResourceExists: false,
			},
			wantErr: false,
		},
		{
			name: "no external name",
			mg: &v1alpha1.Repository{
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:  "test-repo",
						Owner: stringPtr("test-owner"),
					},
				},
			},
			setup: func(mc *MockClient) {},
			want: managed.ExternalObservation{
				ResourceExists: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &MockClient{}
			if tt.setup != nil {
				tt.setup(mc)
			}

			e := &external{client: mc}
			got, err := e.Observe(context.Background(), tt.mg)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want.ResourceExists, got.ResourceExists)
			assert.Equal(t, tt.want.ResourceUpToDate, got.ResourceUpToDate)

			mc.AssertExpectations(t)
		})
	}
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name    string
		mg      resource.Managed
		setup   func(*MockClient)
		want    managed.ExternalCreation
		wantErr bool
	}{
		{
			name: "create repository in organization",
			mg: &v1alpha1.Repository{
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:        "test-repo",
						Owner:       stringPtr("test-org"),
						Description: stringPtr("Test description"),
						Private:     boolPtr(true),
						AutoInit:    boolPtr(true),
					},
				},
			},
			setup: func(mc *MockClient) {
				mc.On("CreateOrganizationRepository", mock.Anything, "test-org", mock.Anything).
					Return(&giteaclients.Repository{
						ID:   123,
						Name: "test-repo",
					}, nil)
			},
			want:    managed.ExternalCreation{},
			wantErr: false,
		},
		{
			name: "create repository for user",
			mg: &v1alpha1.Repository{
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:        "test-repo",
						Description: stringPtr("Test description"),
						Private:     boolPtr(false),
					},
				},
			},
			setup: func(mc *MockClient) {
				mc.On("CreateRepository", mock.Anything, mock.Anything).
					Return(&giteaclients.Repository{
						ID:   124,
						Name: "test-repo",
					}, nil)
			},
			want:    managed.ExternalCreation{},
			wantErr: false,
		},
		{
			name: "create fails",
			mg: &v1alpha1.Repository{
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:  "test-repo",
						Owner: stringPtr("test-org"),
					},
				},
			},
			setup: func(mc *MockClient) {
				mc.On("CreateOrganizationRepository", mock.Anything, "test-org", mock.Anything).
					Return(nil, errors.New("creation failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &MockClient{}
			if tt.setup != nil {
				tt.setup(mc)
			}

			e := &external{client: mc}
			_, err := e.Create(context.Background(), tt.mg)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify external name was set
				assert.NotEmpty(t, meta.GetExternalName(tt.mg))
			}

			mc.AssertExpectations(t)
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name    string
		mg      resource.Managed
		setup   func(*MockClient)
		wantErr bool
	}{
		{
			name: "delete repository",
			mg: func() resource.Managed {
				repo := &v1alpha1.Repository{
					Spec: v1alpha1.RepositorySpec{
						ForProvider: v1alpha1.RepositoryParameters{
							Name:  "test-repo",
							Owner: stringPtr("test-owner"),
						},
					},
				}
				meta.SetExternalName(repo, "test-repo")
				return repo
			}(),
			setup: func(mc *MockClient) {
				mc.On("DeleteRepository", mock.Anything, "test-owner", "test-repo").
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "delete fails",
			mg: func() resource.Managed {
				repo := &v1alpha1.Repository{
					Spec: v1alpha1.RepositorySpec{
						ForProvider: v1alpha1.RepositoryParameters{
							Name:  "test-repo",
							Owner: stringPtr("test-owner"),
						},
					},
				}
				meta.SetExternalName(repo, "test-repo")
				return repo
			}(),
			setup: func(mc *MockClient) {
				mc.On("DeleteRepository", mock.Anything, "test-owner", "test-repo").
					Return(errors.New("deletion failed"))
			},
			wantErr: true,
		},
		{
			name: "missing owner",
			mg: &v1alpha1.Repository{
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name: "test-repo",
					},
				},
			},
			setup:   func(mc *MockClient) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &MockClient{}
			if tt.setup != nil {
				tt.setup(mc)
			}

			e := &external{client: mc}
			_, err := e.Delete(context.Background(), tt.mg)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mc.AssertExpectations(t)
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
