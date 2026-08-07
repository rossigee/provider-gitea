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

// Package testutil provides a NoopClient that implements the full
// clients.Client interface with zero-value returns. Embedding it in
// test mocks lets each test override only the methods it exercises.
package testutil

import (
	"context"

	"github.com/rossigee/provider-gitea/internal/clients"
)

// NoopClient implements clients.Client with no-op return values for every method.
type NoopClient struct{}

// Repository
func (NoopClient) GetRepository(ctx context.Context, owner, name string) (*clients.Repository, error) {
	return nil, nil
}
func (NoopClient) CreateRepository(ctx context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
	return nil, nil
}
func (NoopClient) CreateOrganizationRepository(ctx context.Context, org string, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
	return nil, nil
}
func (NoopClient) UpdateRepository(ctx context.Context, owner, name string, req *clients.UpdateRepositoryRequest) (*clients.Repository, error) {
	return nil, nil
}
func (NoopClient) DeleteRepository(ctx context.Context, owner, name string) error { return nil }

// Organization
func (NoopClient) GetOrganization(ctx context.Context, name string) (*clients.Organization, error) {
	return nil, nil
}
func (NoopClient) CreateOrganization(ctx context.Context, req *clients.CreateOrganizationRequest) (*clients.Organization, error) {
	return nil, nil
}
func (NoopClient) UpdateOrganization(ctx context.Context, name string, req *clients.UpdateOrganizationRequest) (*clients.Organization, error) {
	return nil, nil
}
func (NoopClient) DeleteOrganization(ctx context.Context, name string) error { return nil }

// User
func (NoopClient) GetUser(ctx context.Context, username string) (*clients.User, error) {
	return nil, nil
}
func (NoopClient) CreateUser(ctx context.Context, req *clients.CreateUserRequest) (*clients.User, error) {
	return nil, nil
}
func (NoopClient) UpdateUser(ctx context.Context, username string, req *clients.UpdateUserRequest) (*clients.User, error) {
	return nil, nil
}
func (NoopClient) DeleteUser(ctx context.Context, username string) error { return nil }

// Webhooks
func (NoopClient) GetRepositoryWebhook(ctx context.Context, owner, repo string, id int64) (*clients.Webhook, error) {
	return nil, nil
}
func (NoopClient) CreateRepositoryWebhook(ctx context.Context, owner, repo string, req *clients.CreateWebhookRequest) (*clients.Webhook, error) {
	return nil, nil
}
func (NoopClient) UpdateRepositoryWebhook(ctx context.Context, owner, repo string, id int64, req *clients.UpdateWebhookRequest) (*clients.Webhook, error) {
	return nil, nil
}
func (NoopClient) DeleteRepositoryWebhook(ctx context.Context, owner, repo string, id int64) error {
	return nil
}
func (NoopClient) GetOrganizationWebhook(ctx context.Context, org string, id int64) (*clients.Webhook, error) {
	return nil, nil
}
func (NoopClient) CreateOrganizationWebhook(ctx context.Context, org string, req *clients.CreateWebhookRequest) (*clients.Webhook, error) {
	return nil, nil
}
func (NoopClient) UpdateOrganizationWebhook(ctx context.Context, org string, id int64, req *clients.UpdateWebhookRequest) (*clients.Webhook, error) {
	return nil, nil
}
func (NoopClient) DeleteOrganizationWebhook(ctx context.Context, org string, id int64) error {
	return nil
}

// Deploy keys
func (NoopClient) GetDeployKey(ctx context.Context, owner, repo string, id int64) (*clients.DeployKey, error) {
	return nil, nil
}
func (NoopClient) CreateDeployKey(ctx context.Context, owner, repo string, req *clients.CreateDeployKeyRequest) (*clients.DeployKey, error) {
	return nil, nil
}
func (NoopClient) DeleteDeployKey(ctx context.Context, owner, repo string, id int64) error { return nil }

// Org secrets
func (NoopClient) GetOrganizationSecret(ctx context.Context, org, secretName string) (*clients.OrganizationSecret, error) {
	return nil, nil
}
func (NoopClient) CreateOrganizationSecret(ctx context.Context, org, secretName string, req *clients.CreateOrganizationSecretRequest) error {
	return nil
}
func (NoopClient) UpdateOrganizationSecret(ctx context.Context, org, secretName string, req *clients.CreateOrganizationSecretRequest) error {
	return nil
}
func (NoopClient) DeleteOrganizationSecret(ctx context.Context, org, secretName string) error { return nil }

// Teams
func (NoopClient) GetTeam(ctx context.Context, teamID int64) (*clients.Team, error) { return nil, nil }
func (NoopClient) CreateTeam(ctx context.Context, org string, req *clients.CreateTeamRequest) (*clients.Team, error) {
	return nil, nil
}
func (NoopClient) UpdateTeam(ctx context.Context, teamID int64, req *clients.UpdateTeamRequest) (*clients.Team, error) {
	return nil, nil
}
func (NoopClient) DeleteTeam(ctx context.Context, teamID int64) error { return nil }
func (NoopClient) ListOrganizationTeams(ctx context.Context, org string) ([]*clients.Team, error) {
	return nil, nil
}

// Labels
func (NoopClient) GetLabel(ctx context.Context, owner, repo string, labelID int64) (*clients.Label, error) {
	return nil, nil
}
func (NoopClient) CreateLabel(ctx context.Context, owner, repo string, req *clients.CreateLabelRequest) (*clients.Label, error) {
	return nil, nil
}
func (NoopClient) UpdateLabel(ctx context.Context, owner, repo string, labelID int64, req *clients.UpdateLabelRequest) (*clients.Label, error) {
	return nil, nil
}
func (NoopClient) DeleteLabel(ctx context.Context, owner, repo string, labelID int64) error { return nil }
func (NoopClient) ListRepositoryLabels(ctx context.Context, owner, repo string) ([]*clients.Label, error) {
	return nil, nil
}

// Collaborators
func (NoopClient) GetRepositoryCollaborator(ctx context.Context, owner, repo, username string) (*clients.RepositoryCollaborator, error) {
	return nil, nil
}
func (NoopClient) AddRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *clients.AddCollaboratorRequest) error {
	return nil
}
func (NoopClient) UpdateRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *clients.UpdateCollaboratorRequest) error {
	return nil
}
func (NoopClient) RemoveRepositoryCollaborator(ctx context.Context, owner, repo, username string) error {
	return nil
}
func (NoopClient) ListRepositoryCollaborators(ctx context.Context, owner, repo string) ([]*clients.RepositoryCollaborator, error) {
	return nil, nil
}

// Org settings
func (NoopClient) GetOrganizationSettings(ctx context.Context, org string) (*clients.OrganizationSettings, error) {
	return nil, nil
}
func (NoopClient) UpdateOrganizationSettings(ctx context.Context, org string, req *clients.UpdateOrganizationSettingsRequest) (*clients.OrganizationSettings, error) {
	return nil, nil
}

// Git hooks
func (NoopClient) GetGitHook(ctx context.Context, repository, hookType string) (*clients.GitHook, error) {
	return nil, nil
}
func (NoopClient) CreateGitHook(ctx context.Context, repository string, req *clients.CreateGitHookRequest) (*clients.GitHook, error) {
	return nil, nil
}
func (NoopClient) UpdateGitHook(ctx context.Context, repository, hookType string, req *clients.UpdateGitHookRequest) (*clients.GitHook, error) {
	return nil, nil
}
func (NoopClient) DeleteGitHook(ctx context.Context, repository, hookType string) error { return nil }

// Branch protection
func (NoopClient) GetBranchProtection(ctx context.Context, repository, branch string) (*clients.BranchProtection, error) {
	return nil, nil
}
func (NoopClient) CreateBranchProtection(ctx context.Context, repository, branch string, req *clients.CreateBranchProtectionRequest) (*clients.BranchProtection, error) {
	return nil, nil
}
func (NoopClient) UpdateBranchProtection(ctx context.Context, repository, branch string, req *clients.UpdateBranchProtectionRequest) (*clients.BranchProtection, error) {
	return nil, nil
}
func (NoopClient) DeleteBranchProtection(ctx context.Context, repository, branch string) error { return nil }

// Repo keys
func (NoopClient) GetRepositoryKey(ctx context.Context, repository string, keyID int64) (*clients.RepositoryKey, error) {
	return nil, nil
}
func (NoopClient) CreateRepositoryKey(ctx context.Context, repository string, req *clients.CreateRepositoryKeyRequest) (*clients.RepositoryKey, error) {
	return nil, nil
}
func (NoopClient) UpdateRepositoryKey(ctx context.Context, repository string, keyID int64, req *clients.UpdateRepositoryKeyRequest) (*clients.RepositoryKey, error) {
	return nil, nil
}
func (NoopClient) DeleteRepositoryKey(ctx context.Context, repository string, keyID int64) error { return nil }

// Access tokens
func (NoopClient) GetAccessToken(ctx context.Context, username string, tokenID int64) (*clients.AccessToken, error) {
	return nil, nil
}
func (NoopClient) CreateAccessToken(ctx context.Context, username string, req *clients.CreateAccessTokenRequest) (*clients.AccessToken, error) {
	return nil, nil
}
func (NoopClient) UpdateAccessToken(ctx context.Context, username string, tokenID int64, req *clients.UpdateAccessTokenRequest) (*clients.AccessToken, error) {
	return nil, nil
}
func (NoopClient) DeleteAccessToken(ctx context.Context, username string, tokenID int64) error { return nil }

// Repo secrets
func (NoopClient) GetRepositorySecret(ctx context.Context, repository, secretName string) (*clients.RepositorySecret, error) {
	return nil, nil
}
func (NoopClient) CreateRepositorySecret(ctx context.Context, repository, secretName string, req *clients.CreateRepositorySecretRequest) error {
	return nil
}
func (NoopClient) UpdateRepositorySecret(ctx context.Context, repository, secretName string, req *clients.UpdateRepositorySecretRequest) error {
	return nil
}
func (NoopClient) DeleteRepositorySecret(ctx context.Context, repository, secretName string) error { return nil }

// User keys
func (NoopClient) GetUserKey(ctx context.Context, username string, keyID int64) (*clients.UserKey, error) {
	return nil, nil
}
func (NoopClient) CreateUserKey(ctx context.Context, username string, req *clients.CreateUserKeyRequest) (*clients.UserKey, error) {
	return nil, nil
}
func (NoopClient) UpdateUserKey(ctx context.Context, username string, keyID int64, req *clients.UpdateUserKeyRequest) (*clients.UserKey, error) {
	return nil, nil
}
func (NoopClient) DeleteUserKey(ctx context.Context, username string, keyID int64) error { return nil }

// Issues / PRs
func (NoopClient) GetIssue(ctx context.Context, owner, repo string, number int64) (*clients.Issue, error) {
	return nil, nil
}
func (NoopClient) CreateIssue(ctx context.Context, owner, repo string, req *clients.CreateIssueOptions) (*clients.Issue, error) {
	return nil, nil
}
func (NoopClient) UpdateIssue(ctx context.Context, owner, repo string, number int64, req *clients.UpdateIssueOptions) (*clients.Issue, error) {
	return nil, nil
}
func (NoopClient) DeleteIssue(ctx context.Context, owner, repo string, number int64) error { return nil }
func (NoopClient) GetPullRequest(ctx context.Context, owner, repo string, number int64) (*clients.PullRequest, error) {
	return nil, nil
}
func (NoopClient) CreatePullRequest(ctx context.Context, owner, repo string, req *clients.CreatePullRequestOptions) (*clients.PullRequest, error) {
	return nil, nil
}
func (NoopClient) UpdatePullRequest(ctx context.Context, owner, repo string, number int64, req *clients.UpdatePullRequestOptions) (*clients.PullRequest, error) {
	return nil, nil
}
func (NoopClient) DeletePullRequest(ctx context.Context, owner, repo string, number int64) error { return nil }
func (NoopClient) MergePullRequest(ctx context.Context, owner, repo string, number int64, req *clients.MergePullRequestOptions) (*clients.PullRequest, error) {
	return nil, nil
}

// Releases
func (NoopClient) GetRelease(ctx context.Context, owner, repo string, id int64) (*clients.Release, error) {
	return nil, nil
}
func (NoopClient) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*clients.Release, error) {
	return nil, nil
}
func (NoopClient) CreateRelease(ctx context.Context, owner, repo string, req *clients.CreateReleaseOptions) (*clients.Release, error) {
	return nil, nil
}
func (NoopClient) UpdateRelease(ctx context.Context, owner, repo string, id int64, req *clients.UpdateReleaseOptions) (*clients.Release, error) {
	return nil, nil
}
func (NoopClient) DeleteRelease(ctx context.Context, owner, repo string, id int64) error { return nil }
func (NoopClient) CreateReleaseAttachment(ctx context.Context, owner, repo string, releaseID int64, filename, contentType string, content []byte) (*clients.ReleaseAttachment, error) {
	return nil, nil
}
func (NoopClient) DeleteReleaseAttachment(ctx context.Context, owner, repo string, releaseID, attachmentID int64) error {
	return nil
}

// Org members
func (NoopClient) GetOrganizationMember(ctx context.Context, org, username string) (*clients.OrganizationMember, error) {
	return nil, nil
}
func (NoopClient) AddOrganizationMember(ctx context.Context, org, username string, req *clients.AddOrganizationMemberRequest) (*clients.OrganizationMember, error) {
	return nil, nil
}
func (NoopClient) UpdateOrganizationMember(ctx context.Context, org, username string, req *clients.UpdateOrganizationMemberRequest) (*clients.OrganizationMember, error) {
	return nil, nil
}
func (NoopClient) RemoveOrganizationMember(ctx context.Context, org, username string) error { return nil }

// Actions
func (NoopClient) GetAction(ctx context.Context, repository, workflowName string) (*clients.Action, error) {
	return nil, nil
}
func (NoopClient) CreateAction(ctx context.Context, repository string, req *clients.CreateActionRequest) (*clients.Action, error) {
	return nil, nil
}
func (NoopClient) UpdateAction(ctx context.Context, repository, workflowName string, req *clients.UpdateActionRequest) (*clients.Action, error) {
	return nil, nil
}
func (NoopClient) DeleteAction(ctx context.Context, repository, workflowName string) error { return nil }
func (NoopClient) EnableAction(ctx context.Context, repository, workflowName string) error { return nil }
func (NoopClient) DisableAction(ctx context.Context, repository, workflowName string) error { return nil }

// Runners
func (NoopClient) GetRunner(ctx context.Context, scope, scopeValue string, runnerID int64) (*clients.Runner, error) {
	return nil, nil
}
func (NoopClient) CreateRunner(ctx context.Context, scope, scopeValue string, req *clients.CreateRunnerRequest) (*clients.Runner, error) {
	return nil, nil
}
func (NoopClient) UpdateRunner(ctx context.Context, scope, scopeValue string, runnerID int64, req *clients.UpdateRunnerRequest) (*clients.Runner, error) {
	return nil, nil
}
func (NoopClient) DeleteRunner(ctx context.Context, scope, scopeValue string, runnerID int64) error { return nil }

// Admin users
func (NoopClient) GetAdminUser(ctx context.Context, username string) (*clients.AdminUser, error) {
	return nil, nil
}
func (NoopClient) CreateAdminUser(ctx context.Context, req *clients.CreateAdminUserRequest) (*clients.AdminUser, error) {
	return nil, nil
}
func (NoopClient) UpdateAdminUser(ctx context.Context, username string, req *clients.UpdateAdminUserRequest) (*clients.AdminUser, error) {
	return nil, nil
}
func (NoopClient) DeleteAdminUser(ctx context.Context, username string) error { return nil }

// Disconnect
func (NoopClient) Disconnect(ctx context.Context) error { return nil }
