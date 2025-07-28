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

	"github.com/crossplane-contrib/provider-gitea/apis/repository/v1alpha1"
	giteaclients "github.com/crossplane-contrib/provider-gitea/internal/clients"
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
			want: managed.ExternalCreation{},
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
			want: managed.ExternalCreation{},
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