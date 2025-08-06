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

// Package mock provides comprehensive mock implementations for testing
package mock

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/rossigee/provider-gitea/internal/clients"
)

// Client provides a comprehensive mock implementation of the Gitea client interface
type Client struct {
	mock.Mock
}

// Verify that Client implements the clients.Client interface
var _ clients.Client = (*Client)(nil)

// Repository operations
func (m *Client) GetRepository(ctx context.Context, owner, name string) (*clients.Repository, error) {
	args := m.Called(ctx, owner, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Repository), args.Error(1)
}

func (m *Client) CreateRepository(ctx context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Repository), args.Error(1)
}

func (m *Client) CreateOrganizationRepository(ctx context.Context, org string, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
	args := m.Called(ctx, org, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Repository), args.Error(1)
}

func (m *Client) UpdateRepository(ctx context.Context, owner, name string, req *clients.UpdateRepositoryRequest) (*clients.Repository, error) {
	args := m.Called(ctx, owner, name, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Repository), args.Error(1)
}

func (m *Client) DeleteRepository(ctx context.Context, owner, name string) error {
	args := m.Called(ctx, owner, name)
	return args.Error(0)
}

// Organization operations
func (m *Client) GetOrganization(ctx context.Context, name string) (*clients.Organization, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Organization), args.Error(1)
}

func (m *Client) CreateOrganization(ctx context.Context, req *clients.CreateOrganizationRequest) (*clients.Organization, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Organization), args.Error(1)
}

func (m *Client) UpdateOrganization(ctx context.Context, name string, req *clients.UpdateOrganizationRequest) (*clients.Organization, error) {
	args := m.Called(ctx, name, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Organization), args.Error(1)
}

func (m *Client) DeleteOrganization(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// User operations
func (m *Client) GetUser(ctx context.Context, username string) (*clients.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.User), args.Error(1)
}

func (m *Client) CreateUser(ctx context.Context, req *clients.CreateUserRequest) (*clients.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.User), args.Error(1)
}

func (m *Client) UpdateUser(ctx context.Context, username string, req *clients.UpdateUserRequest) (*clients.User, error) {
	args := m.Called(ctx, username, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.User), args.Error(1)
}

func (m *Client) DeleteUser(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
	return args.Error(0)
}

// Webhook operations
func (m *Client) GetRepositoryWebhook(ctx context.Context, owner, repo string, id int64) (*clients.Webhook, error) {
	args := m.Called(ctx, owner, repo, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Webhook), args.Error(1)
}

func (m *Client) CreateRepositoryWebhook(ctx context.Context, owner, repo string, req *clients.CreateWebhookRequest) (*clients.Webhook, error) {
	args := m.Called(ctx, owner, repo, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Webhook), args.Error(1)
}

func (m *Client) UpdateRepositoryWebhook(ctx context.Context, owner, repo string, id int64, req *clients.UpdateWebhookRequest) (*clients.Webhook, error) {
	args := m.Called(ctx, owner, repo, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Webhook), args.Error(1)
}

func (m *Client) DeleteRepositoryWebhook(ctx context.Context, owner, repo string, id int64) error {
	args := m.Called(ctx, owner, repo, id)
	return args.Error(0)
}

func (m *Client) GetOrganizationWebhook(ctx context.Context, org string, id int64) (*clients.Webhook, error) {
	args := m.Called(ctx, org, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Webhook), args.Error(1)
}

func (m *Client) CreateOrganizationWebhook(ctx context.Context, org string, req *clients.CreateWebhookRequest) (*clients.Webhook, error) {
	args := m.Called(ctx, org, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Webhook), args.Error(1)
}

func (m *Client) UpdateOrganizationWebhook(ctx context.Context, org string, id int64, req *clients.UpdateWebhookRequest) (*clients.Webhook, error) {
	args := m.Called(ctx, org, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Webhook), args.Error(1)
}

func (m *Client) DeleteOrganizationWebhook(ctx context.Context, org string, id int64) error {
	args := m.Called(ctx, org, id)
	return args.Error(0)
}

// Deploy Key operations
func (m *Client) GetDeployKey(ctx context.Context, owner, repo string, id int64) (*clients.DeployKey, error) {
	args := m.Called(ctx, owner, repo, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.DeployKey), args.Error(1)
}

func (m *Client) CreateDeployKey(ctx context.Context, owner, repo string, req *clients.CreateDeployKeyRequest) (*clients.DeployKey, error) {
	args := m.Called(ctx, owner, repo, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.DeployKey), args.Error(1)
}

func (m *Client) DeleteDeployKey(ctx context.Context, owner, repo string, id int64) error {
	args := m.Called(ctx, owner, repo, id)
	return args.Error(0)
}

// Organization Secret operations
func (m *Client) GetOrganizationSecret(ctx context.Context, org, secretName string) (*clients.OrganizationSecret, error) {
	args := m.Called(ctx, org, secretName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.OrganizationSecret), args.Error(1)
}

func (m *Client) CreateOrganizationSecret(ctx context.Context, org, secretName string, req *clients.CreateOrganizationSecretRequest) error {
	args := m.Called(ctx, org, secretName, req)
	return args.Error(0)
}

func (m *Client) UpdateOrganizationSecret(ctx context.Context, org, secretName string, req *clients.CreateOrganizationSecretRequest) error {
	args := m.Called(ctx, org, secretName, req)
	return args.Error(0)
}

func (m *Client) DeleteOrganizationSecret(ctx context.Context, org, secretName string) error {
	args := m.Called(ctx, org, secretName)
	return args.Error(0)
}

// Team operations
func (m *Client) GetTeam(ctx context.Context, teamID int64) (*clients.Team, error) {
	args := m.Called(ctx, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Team), args.Error(1)
}

func (m *Client) CreateTeam(ctx context.Context, org string, req *clients.CreateTeamRequest) (*clients.Team, error) {
	args := m.Called(ctx, org, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Team), args.Error(1)
}

func (m *Client) UpdateTeam(ctx context.Context, teamID int64, req *clients.UpdateTeamRequest) (*clients.Team, error) {
	args := m.Called(ctx, teamID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Team), args.Error(1)
}

func (m *Client) DeleteTeam(ctx context.Context, teamID int64) error {
	args := m.Called(ctx, teamID)
	return args.Error(0)
}

func (m *Client) ListOrganizationTeams(ctx context.Context, org string) ([]*clients.Team, error) {
	args := m.Called(ctx, org)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*clients.Team), args.Error(1)
}

// Label operations
func (m *Client) GetLabel(ctx context.Context, owner, repo string, labelID int64) (*clients.Label, error) {
	args := m.Called(ctx, owner, repo, labelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Label), args.Error(1)
}

func (m *Client) CreateLabel(ctx context.Context, owner, repo string, req *clients.CreateLabelRequest) (*clients.Label, error) {
	args := m.Called(ctx, owner, repo, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Label), args.Error(1)
}

func (m *Client) UpdateLabel(ctx context.Context, owner, repo string, labelID int64, req *clients.UpdateLabelRequest) (*clients.Label, error) {
	args := m.Called(ctx, owner, repo, labelID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Label), args.Error(1)
}

func (m *Client) DeleteLabel(ctx context.Context, owner, repo string, labelID int64) error {
	args := m.Called(ctx, owner, repo, labelID)
	return args.Error(0)
}

func (m *Client) ListRepositoryLabels(ctx context.Context, owner, repo string) ([]*clients.Label, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*clients.Label), args.Error(1)
}

// Repository Collaborator operations
func (m *Client) GetRepositoryCollaborator(ctx context.Context, owner, repo, username string) (*clients.RepositoryCollaborator, error) {
	args := m.Called(ctx, owner, repo, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.RepositoryCollaborator), args.Error(1)
}

func (m *Client) AddRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *clients.AddCollaboratorRequest) error {
	args := m.Called(ctx, owner, repo, username, req)
	return args.Error(0)
}

func (m *Client) UpdateRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *clients.UpdateCollaboratorRequest) error {
	args := m.Called(ctx, owner, repo, username, req)
	return args.Error(0)
}

func (m *Client) RemoveRepositoryCollaborator(ctx context.Context, owner, repo, username string) error {
	args := m.Called(ctx, owner, repo, username)
	return args.Error(0)
}

func (m *Client) ListRepositoryCollaborators(ctx context.Context, owner, repo string) ([]*clients.RepositoryCollaborator, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*clients.RepositoryCollaborator), args.Error(1)
}

// Organization Settings operations
func (m *Client) GetOrganizationSettings(ctx context.Context, org string) (*clients.OrganizationSettings, error) {
	args := m.Called(ctx, org)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.OrganizationSettings), args.Error(1)
}

func (m *Client) UpdateOrganizationSettings(ctx context.Context, org string, req *clients.UpdateOrganizationSettingsRequest) (*clients.OrganizationSettings, error) {
	args := m.Called(ctx, org, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.OrganizationSettings), args.Error(1)
}

// Git Hooks operations
func (m *Client) GetGitHook(ctx context.Context, repository, hookType string) (*clients.GitHook, error) {
	args := m.Called(ctx, repository, hookType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.GitHook), args.Error(1)
}

func (m *Client) CreateGitHook(ctx context.Context, repository string, req *clients.CreateGitHookRequest) (*clients.GitHook, error) {
	args := m.Called(ctx, repository, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.GitHook), args.Error(1)
}

func (m *Client) UpdateGitHook(ctx context.Context, repository, hookType string, req *clients.UpdateGitHookRequest) (*clients.GitHook, error) {
	args := m.Called(ctx, repository, hookType, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.GitHook), args.Error(1)
}

func (m *Client) DeleteGitHook(ctx context.Context, repository, hookType string) error {
	args := m.Called(ctx, repository, hookType)
	return args.Error(0)
}

// Branch Protection operations
func (m *Client) GetBranchProtection(ctx context.Context, repository, branch string) (*clients.BranchProtection, error) {
	args := m.Called(ctx, repository, branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.BranchProtection), args.Error(1)
}

func (m *Client) CreateBranchProtection(ctx context.Context, repository, branch string, req *clients.CreateBranchProtectionRequest) (*clients.BranchProtection, error) {
	args := m.Called(ctx, repository, branch, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.BranchProtection), args.Error(1)
}

func (m *Client) UpdateBranchProtection(ctx context.Context, repository, branch string, req *clients.UpdateBranchProtectionRequest) (*clients.BranchProtection, error) {
	args := m.Called(ctx, repository, branch, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.BranchProtection), args.Error(1)
}

func (m *Client) DeleteBranchProtection(ctx context.Context, repository, branch string) error {
	args := m.Called(ctx, repository, branch)
	return args.Error(0)
}

// Repository Key operations
func (m *Client) GetRepositoryKey(ctx context.Context, repository string, keyID int64) (*clients.RepositoryKey, error) {
	args := m.Called(ctx, repository, keyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.RepositoryKey), args.Error(1)
}

func (m *Client) CreateRepositoryKey(ctx context.Context, repository string, req *clients.CreateRepositoryKeyRequest) (*clients.RepositoryKey, error) {
	args := m.Called(ctx, repository, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.RepositoryKey), args.Error(1)
}

func (m *Client) UpdateRepositoryKey(ctx context.Context, repository string, keyID int64, req *clients.UpdateRepositoryKeyRequest) (*clients.RepositoryKey, error) {
	args := m.Called(ctx, repository, keyID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.RepositoryKey), args.Error(1)
}

func (m *Client) DeleteRepositoryKey(ctx context.Context, repository string, keyID int64) error {
	args := m.Called(ctx, repository, keyID)
	return args.Error(0)
}

// Access Token operations
func (m *Client) GetAccessToken(ctx context.Context, username string, tokenID int64) (*clients.AccessToken, error) {
	args := m.Called(ctx, username, tokenID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.AccessToken), args.Error(1)
}

func (m *Client) CreateAccessToken(ctx context.Context, username string, req *clients.CreateAccessTokenRequest) (*clients.AccessToken, error) {
	args := m.Called(ctx, username, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.AccessToken), args.Error(1)
}

func (m *Client) UpdateAccessToken(ctx context.Context, username string, tokenID int64, req *clients.UpdateAccessTokenRequest) (*clients.AccessToken, error) {
	args := m.Called(ctx, username, tokenID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.AccessToken), args.Error(1)
}

func (m *Client) DeleteAccessToken(ctx context.Context, username string, tokenID int64) error {
	args := m.Called(ctx, username, tokenID)
	return args.Error(0)
}

// Repository Secret operations
func (m *Client) GetRepositorySecret(ctx context.Context, repository, secretName string) (*clients.RepositorySecret, error) {
	args := m.Called(ctx, repository, secretName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.RepositorySecret), args.Error(1)
}

func (m *Client) CreateRepositorySecret(ctx context.Context, repository, secretName string, req *clients.CreateRepositorySecretRequest) error {
	args := m.Called(ctx, repository, secretName, req)
	return args.Error(0)
}

func (m *Client) UpdateRepositorySecret(ctx context.Context, repository, secretName string, req *clients.UpdateRepositorySecretRequest) error {
	args := m.Called(ctx, repository, secretName, req)
	return args.Error(0)
}

func (m *Client) DeleteRepositorySecret(ctx context.Context, repository, secretName string) error {
	args := m.Called(ctx, repository, secretName)
	return args.Error(0)
}

// User Key operations
func (m *Client) GetUserKey(ctx context.Context, username string, keyID int64) (*clients.UserKey, error) {
	args := m.Called(ctx, username, keyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.UserKey), args.Error(1)
}

func (m *Client) CreateUserKey(ctx context.Context, username string, req *clients.CreateUserKeyRequest) (*clients.UserKey, error) {
	args := m.Called(ctx, username, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.UserKey), args.Error(1)
}

func (m *Client) UpdateUserKey(ctx context.Context, username string, keyID int64, req *clients.UpdateUserKeyRequest) (*clients.UserKey, error) {
	args := m.Called(ctx, username, keyID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.UserKey), args.Error(1)
}

func (m *Client) DeleteUserKey(ctx context.Context, username string, keyID int64) error {
	args := m.Called(ctx, username, keyID)
	return args.Error(0)
}

// Issue operations
func (m *Client) GetIssue(ctx context.Context, owner, repo string, number int64) (*clients.Issue, error) {
	args := m.Called(ctx, owner, repo, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Issue), args.Error(1)
}

func (m *Client) CreateIssue(ctx context.Context, owner, repo string, req *clients.CreateIssueOptions) (*clients.Issue, error) {
	args := m.Called(ctx, owner, repo, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Issue), args.Error(1)
}

func (m *Client) UpdateIssue(ctx context.Context, owner, repo string, number int64, req *clients.UpdateIssueOptions) (*clients.Issue, error) {
	args := m.Called(ctx, owner, repo, number, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Issue), args.Error(1)
}

func (m *Client) DeleteIssue(ctx context.Context, owner, repo string, number int64) error {
	args := m.Called(ctx, owner, repo, number)
	return args.Error(0)
}

// PullRequest operations
func (m *Client) GetPullRequest(ctx context.Context, owner, repo string, number int64) (*clients.PullRequest, error) {
	args := m.Called(ctx, owner, repo, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.PullRequest), args.Error(1)
}

func (m *Client) CreatePullRequest(ctx context.Context, owner, repo string, req *clients.CreatePullRequestOptions) (*clients.PullRequest, error) {
	args := m.Called(ctx, owner, repo, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.PullRequest), args.Error(1)
}

func (m *Client) UpdatePullRequest(ctx context.Context, owner, repo string, number int64, req *clients.UpdatePullRequestOptions) (*clients.PullRequest, error) {
	args := m.Called(ctx, owner, repo, number, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.PullRequest), args.Error(1)
}

func (m *Client) DeletePullRequest(ctx context.Context, owner, repo string, number int64) error {
	args := m.Called(ctx, owner, repo, number)
	return args.Error(0)
}

func (m *Client) MergePullRequest(ctx context.Context, owner, repo string, number int64, req *clients.MergePullRequestOptions) (*clients.PullRequest, error) {
	args := m.Called(ctx, owner, repo, number, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.PullRequest), args.Error(1)
}

// Release operations
func (m *Client) GetRelease(ctx context.Context, owner, repo string, id int64) (*clients.Release, error) {
	args := m.Called(ctx, owner, repo, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Release), args.Error(1)
}

func (m *Client) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*clients.Release, error) {
	args := m.Called(ctx, owner, repo, tag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Release), args.Error(1)
}

func (m *Client) CreateRelease(ctx context.Context, owner, repo string, req *clients.CreateReleaseOptions) (*clients.Release, error) {
	args := m.Called(ctx, owner, repo, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Release), args.Error(1)
}

func (m *Client) UpdateRelease(ctx context.Context, owner, repo string, id int64, req *clients.UpdateReleaseOptions) (*clients.Release, error) {
	args := m.Called(ctx, owner, repo, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Release), args.Error(1)
}

func (m *Client) DeleteRelease(ctx context.Context, owner, repo string, id int64) error {
	args := m.Called(ctx, owner, repo, id)
	return args.Error(0)
}

func (m *Client) CreateReleaseAttachment(ctx context.Context, owner, repo string, releaseID int64, filename, contentType string, content []byte) (*clients.ReleaseAttachment, error) {
	args := m.Called(ctx, owner, repo, releaseID, filename, contentType, content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.ReleaseAttachment), args.Error(1)
}

func (m *Client) DeleteReleaseAttachment(ctx context.Context, owner, repo string, releaseID, attachmentID int64) error {
	args := m.Called(ctx, owner, repo, releaseID, attachmentID)
	return args.Error(0)
}

// Organization Member operations
func (m *Client) GetOrganizationMember(ctx context.Context, org, username string) (*clients.OrganizationMember, error) {
	args := m.Called(ctx, org, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.OrganizationMember), args.Error(1)
}

func (m *Client) AddOrganizationMember(ctx context.Context, org, username string, req *clients.AddOrganizationMemberRequest) (*clients.OrganizationMember, error) {
	args := m.Called(ctx, org, username, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.OrganizationMember), args.Error(1)
}

func (m *Client) UpdateOrganizationMember(ctx context.Context, org, username string, req *clients.UpdateOrganizationMemberRequest) (*clients.OrganizationMember, error) {
	args := m.Called(ctx, org, username, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.OrganizationMember), args.Error(1)
}

func (m *Client) RemoveOrganizationMember(ctx context.Context, org, username string) error {
	args := m.Called(ctx, org, username)
	return args.Error(0)
}

// Action operations
func (m *Client) GetAction(ctx context.Context, repository, workflowName string) (*clients.Action, error) {
	args := m.Called(ctx, repository, workflowName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Action), args.Error(1)
}

func (m *Client) CreateAction(ctx context.Context, repository string, req *clients.CreateActionRequest) (*clients.Action, error) {
	args := m.Called(ctx, repository, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Action), args.Error(1)
}

func (m *Client) UpdateAction(ctx context.Context, repository, workflowName string, req *clients.UpdateActionRequest) (*clients.Action, error) {
	args := m.Called(ctx, repository, workflowName, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Action), args.Error(1)
}

func (m *Client) DeleteAction(ctx context.Context, repository, workflowName string) error {
	args := m.Called(ctx, repository, workflowName)
	return args.Error(0)
}

func (m *Client) EnableAction(ctx context.Context, repository, workflowName string) error {
	args := m.Called(ctx, repository, workflowName)
	return args.Error(0)
}

func (m *Client) DisableAction(ctx context.Context, repository, workflowName string) error {
	args := m.Called(ctx, repository, workflowName)
	return args.Error(0)
}

// Runner operations
func (m *Client) GetRunner(ctx context.Context, scope, scopeValue string, runnerID int64) (*clients.Runner, error) {
	args := m.Called(ctx, scope, scopeValue, runnerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Runner), args.Error(1)
}

func (m *Client) CreateRunner(ctx context.Context, scope, scopeValue string, req *clients.CreateRunnerRequest) (*clients.Runner, error) {
	args := m.Called(ctx, scope, scopeValue, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Runner), args.Error(1)
}

func (m *Client) UpdateRunner(ctx context.Context, scope, scopeValue string, runnerID int64, req *clients.UpdateRunnerRequest) (*clients.Runner, error) {
	args := m.Called(ctx, scope, scopeValue, runnerID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.Runner), args.Error(1)
}

func (m *Client) DeleteRunner(ctx context.Context, scope, scopeValue string, runnerID int64) error {
	args := m.Called(ctx, scope, scopeValue, runnerID)
	return args.Error(0)
}

// Admin User operations
func (m *Client) GetAdminUser(ctx context.Context, username string) (*clients.AdminUser, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.AdminUser), args.Error(1)
}

func (m *Client) CreateAdminUser(ctx context.Context, req *clients.CreateAdminUserRequest) (*clients.AdminUser, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.AdminUser), args.Error(1)
}

func (m *Client) UpdateAdminUser(ctx context.Context, username string, req *clients.UpdateAdminUserRequest) (*clients.AdminUser, error) {
	args := m.Called(ctx, username, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.AdminUser), args.Error(1)
}

func (m *Client) DeleteAdminUser(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
	return args.Error(0)
}