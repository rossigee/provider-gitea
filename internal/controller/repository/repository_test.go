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
	"fmt"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/rossigee/provider-gitea/apis/repository/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

type mockRepoClient struct {
	getRepoFn    func(ctx context.Context, owner, name string) (*clients.Repository, error)
	createRepoFn func(ctx context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error)
	createOrgFn  func(ctx context.Context, org string, req *clients.CreateRepositoryRequest) (*clients.Repository, error)
	updateRepoFn func(ctx context.Context, owner, name string, req *clients.UpdateRepositoryRequest) (*clients.Repository, error)
	deleteRepoFn func(ctx context.Context, owner, name string) error
}

func (m *mockRepoClient) GetRepository(ctx context.Context, owner, name string) (*clients.Repository, error) {
	if m.getRepoFn != nil {
		return m.getRepoFn(ctx, owner, name)
	}
	return nil, nil
}
func (m *mockRepoClient) CreateRepository(ctx context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
	if m.createRepoFn != nil {
		return m.createRepoFn(ctx, req)
	}
	return nil, nil
}
func (m *mockRepoClient) CreateOrganizationRepository(ctx context.Context, org string, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
	if m.createOrgFn != nil {
		return m.createOrgFn(ctx, org, req)
	}
	return nil, nil
}
func (m *mockRepoClient) UpdateRepository(ctx context.Context, owner, name string, req *clients.UpdateRepositoryRequest) (*clients.Repository, error) {
	if m.updateRepoFn != nil {
		return m.updateRepoFn(ctx, owner, name, req)
	}
	return nil, nil
}
func (m *mockRepoClient) DeleteRepository(ctx context.Context, owner, name string) error {
	if m.deleteRepoFn != nil {
		return m.deleteRepoFn(ctx, owner, name)
	}
	return nil
}
func (m *mockRepoClient) Disconnect(ctx context.Context) error { return nil }
func (m *mockRepoClient) GetOrganization(ctx context.Context, name string) (*clients.Organization, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateOrganization(ctx context.Context, req *clients.CreateOrganizationRequest) (*clients.Organization, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateOrganization(ctx context.Context, name string, req *clients.UpdateOrganizationRequest) (*clients.Organization, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteOrganization(ctx context.Context, name string) error { return nil }
func (m *mockRepoClient) GetUser(ctx context.Context, username string) (*clients.User, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateUser(ctx context.Context, req *clients.CreateUserRequest) (*clients.User, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateUser(ctx context.Context, username string, req *clients.UpdateUserRequest) (*clients.User, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteUser(ctx context.Context, username string) error { return nil }
func (m *mockRepoClient) GetRepositoryWebhook(ctx context.Context, owner, repo string, id int64) (*clients.Webhook, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateRepositoryWebhook(ctx context.Context, owner, repo string, req *clients.CreateWebhookRequest) (*clients.Webhook, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateRepositoryWebhook(ctx context.Context, owner, repo string, id int64, req *clients.UpdateWebhookRequest) (*clients.Webhook, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteRepositoryWebhook(ctx context.Context, owner, repo string, id int64) error {
	return nil
}
func (m *mockRepoClient) GetOrganizationWebhook(ctx context.Context, org string, id int64) (*clients.Webhook, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateOrganizationWebhook(ctx context.Context, org string, req *clients.CreateWebhookRequest) (*clients.Webhook, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateOrganizationWebhook(ctx context.Context, org string, id int64, req *clients.UpdateWebhookRequest) (*clients.Webhook, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteOrganizationWebhook(ctx context.Context, org string, id int64) error {
	return nil
}
func (m *mockRepoClient) GetDeployKey(ctx context.Context, owner, repo string, id int64) (*clients.DeployKey, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateDeployKey(ctx context.Context, owner, repo string, req *clients.CreateDeployKeyRequest) (*clients.DeployKey, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteDeployKey(ctx context.Context, owner, repo string, id int64) error {
	return nil
}
func (m *mockRepoClient) GetOrganizationSecret(ctx context.Context, org, secretName string) (*clients.OrganizationSecret, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateOrganizationSecret(ctx context.Context, org, secretName string, req *clients.CreateOrganizationSecretRequest) error {
	return nil
}
func (m *mockRepoClient) UpdateOrganizationSecret(ctx context.Context, org, secretName string, req *clients.CreateOrganizationSecretRequest) error {
	return nil
}
func (m *mockRepoClient) DeleteOrganizationSecret(ctx context.Context, org, secretName string) error {
	return nil
}
func (m *mockRepoClient) GetTeam(ctx context.Context, teamID int64) (*clients.Team, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateTeam(ctx context.Context, org string, req *clients.CreateTeamRequest) (*clients.Team, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateTeam(ctx context.Context, teamID int64, req *clients.UpdateTeamRequest) (*clients.Team, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteTeam(ctx context.Context, teamID int64) error { return nil }
func (m *mockRepoClient) ListOrganizationTeams(ctx context.Context, org string) ([]*clients.Team, error) {
	return nil, nil
}
func (m *mockRepoClient) GetLabel(ctx context.Context, owner, repo string, labelID int64) (*clients.Label, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateLabel(ctx context.Context, owner, repo string, req *clients.CreateLabelRequest) (*clients.Label, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateLabel(ctx context.Context, owner, repo string, labelID int64, req *clients.UpdateLabelRequest) (*clients.Label, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteLabel(ctx context.Context, owner, repo string, labelID int64) error {
	return nil
}
func (m *mockRepoClient) ListRepositoryLabels(ctx context.Context, owner, repo string) ([]*clients.Label, error) {
	return nil, nil
}
func (m *mockRepoClient) GetRepositoryCollaborator(ctx context.Context, owner, repo, username string) (*clients.RepositoryCollaborator, error) {
	return nil, nil
}
func (m *mockRepoClient) AddRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *clients.AddCollaboratorRequest) error {
	return nil
}
func (m *mockRepoClient) UpdateRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *clients.UpdateCollaboratorRequest) error {
	return nil
}
func (m *mockRepoClient) RemoveRepositoryCollaborator(ctx context.Context, owner, repo, username string) error {
	return nil
}
func (m *mockRepoClient) ListRepositoryCollaborators(ctx context.Context, owner, repo string) ([]*clients.RepositoryCollaborator, error) {
	return nil, nil
}
func (m *mockRepoClient) GetOrganizationSettings(ctx context.Context, org string) (*clients.OrganizationSettings, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateOrganizationSettings(ctx context.Context, org string, req *clients.UpdateOrganizationSettingsRequest) (*clients.OrganizationSettings, error) {
	return nil, nil
}
func (m *mockRepoClient) GetGitHook(ctx context.Context, repository, hookType string) (*clients.GitHook, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateGitHook(ctx context.Context, repository string, req *clients.CreateGitHookRequest) (*clients.GitHook, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateGitHook(ctx context.Context, repository, hookType string, req *clients.UpdateGitHookRequest) (*clients.GitHook, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteGitHook(ctx context.Context, repository, hookType string) error {
	return nil
}
func (m *mockRepoClient) GetBranchProtection(ctx context.Context, repository, branch string) (*clients.BranchProtection, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateBranchProtection(ctx context.Context, repository, branch string, req *clients.CreateBranchProtectionRequest) (*clients.BranchProtection, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateBranchProtection(ctx context.Context, repository, branch string, req *clients.UpdateBranchProtectionRequest) (*clients.BranchProtection, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteBranchProtection(ctx context.Context, repository, branch string) error {
	return nil
}
func (m *mockRepoClient) GetRepositoryKey(ctx context.Context, repository string, keyID int64) (*clients.RepositoryKey, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateRepositoryKey(ctx context.Context, repository string, req *clients.CreateRepositoryKeyRequest) (*clients.RepositoryKey, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateRepositoryKey(ctx context.Context, repository string, keyID int64, req *clients.UpdateRepositoryKeyRequest) (*clients.RepositoryKey, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteRepositoryKey(ctx context.Context, repository string, keyID int64) error {
	return nil
}
func (m *mockRepoClient) GetAccessToken(ctx context.Context, username string, tokenID int64) (*clients.AccessToken, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateAccessToken(ctx context.Context, username string, req *clients.CreateAccessTokenRequest) (*clients.AccessToken, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateAccessToken(ctx context.Context, username string, tokenID int64, req *clients.UpdateAccessTokenRequest) (*clients.AccessToken, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteAccessToken(ctx context.Context, username string, tokenID int64) error {
	return nil
}
func (m *mockRepoClient) GetRepositorySecret(ctx context.Context, repository, secretName string) (*clients.RepositorySecret, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateRepositorySecret(ctx context.Context, repository, secretName string, req *clients.CreateRepositorySecretRequest) error {
	return nil
}
func (m *mockRepoClient) UpdateRepositorySecret(ctx context.Context, repository, secretName string, req *clients.UpdateRepositorySecretRequest) error {
	return nil
}
func (m *mockRepoClient) DeleteRepositorySecret(ctx context.Context, repository, secretName string) error {
	return nil
}
func (m *mockRepoClient) GetUserKey(ctx context.Context, username string, keyID int64) (*clients.UserKey, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateUserKey(ctx context.Context, username string, req *clients.CreateUserKeyRequest) (*clients.UserKey, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateUserKey(ctx context.Context, username string, keyID int64, req *clients.UpdateUserKeyRequest) (*clients.UserKey, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteUserKey(ctx context.Context, username string, keyID int64) error {
	return nil
}
func (m *mockRepoClient) GetIssue(ctx context.Context, owner, repo string, number int64) (*clients.Issue, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateIssue(ctx context.Context, owner, repo string, req *clients.CreateIssueOptions) (*clients.Issue, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateIssue(ctx context.Context, owner, repo string, number int64, req *clients.UpdateIssueOptions) (*clients.Issue, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteIssue(ctx context.Context, owner, repo string, number int64) error {
	return nil
}
func (m *mockRepoClient) GetPullRequest(ctx context.Context, owner, repo string, number int64) (*clients.PullRequest, error) {
	return nil, nil
}
func (m *mockRepoClient) CreatePullRequest(ctx context.Context, owner, repo string, req *clients.CreatePullRequestOptions) (*clients.PullRequest, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdatePullRequest(ctx context.Context, owner, repo string, number int64, req *clients.UpdatePullRequestOptions) (*clients.PullRequest, error) {
	return nil, nil
}
func (m *mockRepoClient) DeletePullRequest(ctx context.Context, owner, repo string, number int64) error {
	return nil
}
func (m *mockRepoClient) MergePullRequest(ctx context.Context, owner, repo string, number int64, req *clients.MergePullRequestOptions) (*clients.PullRequest, error) {
	return nil, nil
}
func (m *mockRepoClient) GetRelease(ctx context.Context, owner, repo string, id int64) (*clients.Release, error) {
	return nil, nil
}
func (m *mockRepoClient) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*clients.Release, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateRelease(ctx context.Context, owner, repo string, req *clients.CreateReleaseOptions) (*clients.Release, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateRelease(ctx context.Context, owner, repo string, id int64, req *clients.UpdateReleaseOptions) (*clients.Release, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteRelease(ctx context.Context, owner, repo string, id int64) error {
	return nil
}
func (m *mockRepoClient) CreateReleaseAttachment(ctx context.Context, owner, repo string, releaseID int64, filename, contentType string, content []byte) (*clients.ReleaseAttachment, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteReleaseAttachment(ctx context.Context, owner, repo string, releaseID, attachmentID int64) error {
	return nil
}
func (m *mockRepoClient) GetOrganizationMember(ctx context.Context, org, username string) (*clients.OrganizationMember, error) {
	return nil, nil
}
func (m *mockRepoClient) AddOrganizationMember(ctx context.Context, org, username string, req *clients.AddOrganizationMemberRequest) (*clients.OrganizationMember, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateOrganizationMember(ctx context.Context, org, username string, req *clients.UpdateOrganizationMemberRequest) (*clients.OrganizationMember, error) {
	return nil, nil
}
func (m *mockRepoClient) RemoveOrganizationMember(ctx context.Context, org, username string) error {
	return nil
}
func (m *mockRepoClient) GetAction(ctx context.Context, repository, workflowName string) (*clients.Action, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateAction(ctx context.Context, repository string, req *clients.CreateActionRequest) (*clients.Action, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateAction(ctx context.Context, repository, workflowName string, req *clients.UpdateActionRequest) (*clients.Action, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteAction(ctx context.Context, repository, workflowName string) error {
	return nil
}
func (m *mockRepoClient) EnableAction(ctx context.Context, repository, workflowName string) error {
	return nil
}
func (m *mockRepoClient) DisableAction(ctx context.Context, repository, workflowName string) error {
	return nil
}
func (m *mockRepoClient) GetRunner(ctx context.Context, scope, scopeValue string, runnerID int64) (*clients.Runner, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateRunner(ctx context.Context, scope, scopeValue string, req *clients.CreateRunnerRequest) (*clients.Runner, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateRunner(ctx context.Context, scope, scopeValue string, runnerID int64, req *clients.UpdateRunnerRequest) (*clients.Runner, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteRunner(ctx context.Context, scope, scopeValue string, runnerID int64) error {
	return nil
}
func (m *mockRepoClient) ListRunners(ctx context.Context, scope, scopeValue string) ([]*clients.Runner, error) {
	return nil, nil
}
func (m *mockRepoClient) GetAdminUser(ctx context.Context, username string) (*clients.AdminUser, error) {
	return nil, nil
}
func (m *mockRepoClient) CreateAdminUser(ctx context.Context, req *clients.CreateAdminUserRequest) (*clients.AdminUser, error) {
	return nil, nil
}
func (m *mockRepoClient) UpdateAdminUser(ctx context.Context, username string, req *clients.UpdateAdminUserRequest) (*clients.AdminUser, error) {
	return nil, nil
}
func (m *mockRepoClient) DeleteAdminUser(ctx context.Context, username string) error { return nil }

func TestObserve(t *testing.T) {
	t.Run("resource not found returns not exists", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				getRepoFn: func(ctx context.Context, owner, name string) (*clients.Repository, error) {
					return nil, fmt.Errorf("API request failed with status 404: not found")
				},
			},
		}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "owner/test-repo")

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("resource exists updates status", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				getRepoFn: func(ctx context.Context, owner, name string) (*clients.Repository, error) {
					return &clients.Repository{
						ID:       123,
						FullName: "owner/test-repo",
						HTMLURL:  "https://gitea.example.com/owner/test-repo",
						SSHURL:   "ssh://gitea@example.com/owner/test-repo.git",
						CloneURL: "https://gitea.example.com/owner/test-repo.git",
						Language: "Go",
					}, nil
				},
			},
		}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "owner/test-repo")

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
		assert.Equal(t, int64(123), *cr.Status.AtProvider.ID)
	})

	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockRepoClient{}}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("invalid external name format returns error", func(t *testing.T) {
		ec := &externalClient{client: &mockRepoClient{}}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "invalid")

		_, err := ec.Observe(context.Background(), cr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid external-id format")
	})
}

func TestCreate(t *testing.T) {
	t.Run("creates user repository", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				createRepoFn: func(ctx context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
					assert.Equal(t, "test-repo", req.Name)
					assert.Equal(t, "A test repo", req.Description)
					return &clients.Repository{
						Owner: &clients.User{Username: "testuser"},
						Name:  "test-repo",
					}, nil
				},
			},
		}

		desc := "A test repo"
		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Name:        "test-repo",
					Description: &desc,
				},
			},
		}

		_, err := ec.Create(context.Background(), cr)
		require.NoError(t, err)
		assert.Equal(t, "testuser/test-repo", meta.GetExternalName(cr))
	})

	t.Run("creates organization repository", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				createOrgFn: func(ctx context.Context, org string, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
					assert.Equal(t, "testorg", org)
					return &clients.Repository{
						Owner: &clients.User{Username: "testorg"},
						Name:  "test-repo",
					}, nil
				},
			},
		}

		owner := "testorg"
		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Name:  "test-repo",
					Owner: &owner,
				},
			},
		}

		_, err := ec.Create(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("create failure returns error", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				createRepoFn: func(ctx context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
					return nil, fmt.Errorf("API request failed with status 500: internal error")
				},
			},
		}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Name: "test-repo",
				},
			},
		}

		_, err := ec.Create(context.Background(), cr)
		require.Error(t, err)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("updates repository", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				updateRepoFn: func(ctx context.Context, owner, name string, req *clients.UpdateRepositoryRequest) (*clients.Repository, error) {
					assert.Equal(t, "owner", owner)
					assert.Equal(t, "test-repo", name)
					return &clients.Repository{}, nil
				},
			},
		}

		desc := "Updated description"
		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Description: &desc,
				},
			},
		}
		meta.SetExternalName(cr, "owner/test-repo")

		_, err := ec.Update(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("invalid external name returns error", func(t *testing.T) {
		ec := &externalClient{client: &mockRepoClient{}}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "invalid")

		_, err := ec.Update(context.Background(), cr)
		require.Error(t, err)
	})
}

func TestDelete(t *testing.T) {
	t.Run("deletes repository", func(t *testing.T) {
		deleted := false
		ec := &externalClient{
			client: &mockRepoClient{
				deleteRepoFn: func(ctx context.Context, owner, name string) error {
					assert.Equal(t, "owner", owner)
					assert.Equal(t, "test-repo", name)
					deleted = true
					return nil
				},
			},
		}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "owner/test-repo")

		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})

	t.Run("delete failure returns error", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				deleteRepoFn: func(ctx context.Context, owner, name string) error {
					return fmt.Errorf("API request failed with status 500: internal error")
				},
			},
		}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "owner/test-repo")

		_, err := ec.Delete(context.Background(), cr)
		require.Error(t, err)
	})

	t.Run("invalid external name returns error", func(t *testing.T) {
		ec := &externalClient{client: &mockRepoClient{}}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "invalid")

		_, err := ec.Delete(context.Background(), cr)
		require.Error(t, err)
	})
}

func TestConnector(t *testing.T) {
	t.Run("missing provider config returns error", func(t *testing.T) {
		c := &connector{kube: fake.NewClientBuilder().Build()}
		_, err := c.Connect(context.Background(), &v2.Repository{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "providerConfigRef is required")
	})
}
