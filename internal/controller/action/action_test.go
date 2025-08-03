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

package action

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	v1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/action/v1alpha1"
	"github.com/crossplane-contrib/provider-gitea/internal/clients"
)

var (
	_ = errors.New("boom") // errBoom - unused test error
)

type MockClient struct {
	MockGetAction    func(ctx context.Context, repo, workflow string) (*clients.Action, error)
	MockCreateAction func(ctx context.Context, repo string, req *clients.CreateActionRequest) (*clients.Action, error)
	MockUpdateAction func(ctx context.Context, repo, workflow string, req *clients.UpdateActionRequest) (*clients.Action, error)
	MockDeleteAction func(ctx context.Context, repo, workflow string) error
}

func (m *MockClient) GetAction(ctx context.Context, repo, workflow string) (*clients.Action, error) {
	return m.MockGetAction(ctx, repo, workflow)
}

func (m *MockClient) CreateAction(ctx context.Context, repo string, req *clients.CreateActionRequest) (*clients.Action, error) {
	return m.MockCreateAction(ctx, repo, req)
}

func (m *MockClient) UpdateAction(ctx context.Context, repo, workflow string, req *clients.UpdateActionRequest) (*clients.Action, error) {
	return m.MockUpdateAction(ctx, repo, workflow, req)
}

func (m *MockClient) DeleteAction(ctx context.Context, repo, workflow string) error {
	return m.MockDeleteAction(ctx, repo, workflow)
}

// Stub implementations for other client methods
func (m *MockClient) GetRepository(ctx context.Context, owner, name string) (*clients.Repository, error)                         { return nil, nil }
func (m *MockClient) CreateRepository(ctx context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error) { return nil, nil }
func (m *MockClient) UpdateRepository(ctx context.Context, owner, name string, req *clients.UpdateRepositoryRequest) (*clients.Repository, error) { return nil, nil }
func (m *MockClient) DeleteRepository(ctx context.Context, owner, repo string) error                                            { return nil }
func (m *MockClient) GetOrganization(ctx context.Context, name string) (*clients.Organization, error)                   { return nil, nil }
func (m *MockClient) CreateOrganization(ctx context.Context, req *clients.CreateOrganizationRequest) (*clients.Organization, error) { return nil, nil }
func (m *MockClient) UpdateOrganization(ctx context.Context, name string, req *clients.UpdateOrganizationRequest) (*clients.Organization, error) { return nil, nil }
func (m *MockClient) DeleteOrganization(ctx context.Context, name string) error                                          { return nil }
func (m *MockClient) GetUser(ctx context.Context, username string) (*clients.User, error)                               { return nil, nil }
func (m *MockClient) CreateUser(ctx context.Context, req *clients.CreateUserRequest) (*clients.User, error)             { return nil, nil }
func (m *MockClient) UpdateUser(ctx context.Context, username string, req *clients.UpdateUserRequest) (*clients.User, error) { return nil, nil }
func (m *MockClient) DeleteUser(ctx context.Context, username string) error                                              { return nil }
func (m *MockClient) GetWebhook(ctx context.Context, repoName string, webhookID int64) (*clients.Webhook, error)        { return nil, nil }
func (m *MockClient) CreateWebhook(ctx context.Context, repoName string, req *clients.CreateWebhookRequest) (*clients.Webhook, error) { return nil, nil }
func (m *MockClient) UpdateWebhook(ctx context.Context, repoName string, webhookID int64, req *clients.UpdateWebhookRequest) (*clients.Webhook, error) { return nil, nil }
func (m *MockClient) DeleteWebhook(ctx context.Context, repoName string, webhookID int64) error                          { return nil }
func (m *MockClient) GetDeployKey(ctx context.Context, owner, repo string, keyID int64) (*clients.DeployKey, error)        { return nil, nil }
func (m *MockClient) CreateDeployKey(ctx context.Context, owner, repo string, req *clients.CreateDeployKeyRequest) (*clients.DeployKey, error) { return nil, nil }
func (m *MockClient) DeleteDeployKey(ctx context.Context, owner, repo string, keyID int64) error                            { return nil }
func (m *MockClient) GetOrganizationWebhook(ctx context.Context, orgName string, webhookID int64) (*clients.Webhook, error) { return nil, nil }
func (m *MockClient) CreateOrganizationWebhook(ctx context.Context, orgName string, req *clients.CreateWebhookRequest) (*clients.Webhook, error) { return nil, nil }
func (m *MockClient) UpdateOrganizationWebhook(ctx context.Context, orgName string, webhookID int64, req *clients.UpdateWebhookRequest) (*clients.Webhook, error) { return nil, nil }
func (m *MockClient) DeleteOrganizationWebhook(ctx context.Context, orgName string, webhookID int64) error              { return nil }
func (m *MockClient) GetOrganizationSecret(ctx context.Context, orgName, secretName string) (*clients.OrganizationSecret, error) { return nil, nil }
func (m *MockClient) CreateOrganizationSecret(ctx context.Context, org, secretName string, req *clients.CreateOrganizationSecretRequest) error { return nil }
func (m *MockClient) UpdateOrganizationSecret(ctx context.Context, org, secretName string, req *clients.CreateOrganizationSecretRequest) error { return nil }
func (m *MockClient) DeleteOrganizationSecret(ctx context.Context, orgName, secretName string) error                    { return nil }
func (m *MockClient) GetBranchProtection(ctx context.Context, repo, branch string) (*clients.BranchProtection, error)  { return nil, nil }
func (m *MockClient) CreateBranchProtection(ctx context.Context, repo, branch string, req *clients.CreateBranchProtectionRequest) (*clients.BranchProtection, error) { return nil, nil }
func (m *MockClient) UpdateBranchProtection(ctx context.Context, repo, branch string, req *clients.UpdateBranchProtectionRequest) (*clients.BranchProtection, error) { return nil, nil }
func (m *MockClient) DeleteBranchProtection(ctx context.Context, repo, branch string) error                             { return nil }
func (m *MockClient) GetRepositoryKey(ctx context.Context, repo string, keyID int64) (*clients.RepositoryKey, error)   { return nil, nil }
func (m *MockClient) CreateRepositoryKey(ctx context.Context, repo string, req *clients.CreateRepositoryKeyRequest) (*clients.RepositoryKey, error) { return nil, nil }
func (m *MockClient) UpdateRepositoryKey(ctx context.Context, repo string, keyID int64, req *clients.UpdateRepositoryKeyRequest) (*clients.RepositoryKey, error) { return nil, nil }
func (m *MockClient) DeleteRepositoryKey(ctx context.Context, repo string, keyID int64) error                           { return nil }
func (m *MockClient) GetAccessToken(ctx context.Context, username string, tokenID int64) (*clients.AccessToken, error) { return nil, nil }
func (m *MockClient) CreateAccessToken(ctx context.Context, username string, req *clients.CreateAccessTokenRequest) (*clients.AccessToken, error) { return nil, nil }
func (m *MockClient) UpdateAccessToken(ctx context.Context, username string, tokenID int64, req *clients.UpdateAccessTokenRequest) (*clients.AccessToken, error) { return nil, nil }
func (m *MockClient) DeleteAccessToken(ctx context.Context, username string, tokenID int64) error { return nil }
func (m *MockClient) GetRepositorySecret(ctx context.Context, repository, secretName string) (*clients.RepositorySecret, error) { return nil, nil }
func (m *MockClient) CreateRepositorySecret(ctx context.Context, repository, secretName string, req *clients.CreateRepositorySecretRequest) error { return nil }
func (m *MockClient) UpdateRepositorySecret(ctx context.Context, repository, secretName string, req *clients.UpdateRepositorySecretRequest) error { return nil }
func (m *MockClient) DeleteRepositorySecret(ctx context.Context, repository, secretName string) error { return nil }
func (m *MockClient) GetUserKey(ctx context.Context, username string, keyID int64) (*clients.UserKey, error)           { return nil, nil }
func (m *MockClient) CreateUserKey(ctx context.Context, username string, req *clients.CreateUserKeyRequest) (*clients.UserKey, error) { return nil, nil }
func (m *MockClient) UpdateUserKey(ctx context.Context, username string, keyID int64, req *clients.UpdateUserKeyRequest) (*clients.UserKey, error) { return nil, nil }
func (m *MockClient) DeleteUserKey(ctx context.Context, username string, keyID int64) error                             { return nil }
func (m *MockClient) AddOrganizationMember(ctx context.Context, org, username string, req *clients.AddOrganizationMemberRequest) (*clients.OrganizationMember, error) { return nil, nil }
func (m *MockClient) GetOrganizationMember(ctx context.Context, org, username string) (*clients.OrganizationMember, error) { return nil, nil }
func (m *MockClient) UpdateOrganizationMember(ctx context.Context, org, username string, req *clients.UpdateOrganizationMemberRequest) (*clients.OrganizationMember, error) { return nil, nil }
func (m *MockClient) RemoveOrganizationMember(ctx context.Context, org, username string) error                         { return nil }
func (m *MockClient) GetRunner(ctx context.Context, scope, scopeValue string, runnerID int64) (*clients.Runner, error) { return nil, nil }
func (m *MockClient) CreateRunner(ctx context.Context, scope, scopeValue string, req *clients.CreateRunnerRequest) (*clients.Runner, error) { return nil, nil }
func (m *MockClient) UpdateRunner(ctx context.Context, scope, scopeValue string, runnerID int64, req *clients.UpdateRunnerRequest) (*clients.Runner, error) { return nil, nil }
func (m *MockClient) DeleteRunner(ctx context.Context, scope, scopeValue string, runnerID int64) error                 { return nil }
func (m *MockClient) GetAdminUser(ctx context.Context, username string) (*clients.AdminUser, error)                    { return nil, nil }
func (m *MockClient) CreateAdminUser(ctx context.Context, req *clients.CreateAdminUserRequest) (*clients.AdminUser, error) { return nil, nil }
func (m *MockClient) UpdateAdminUser(ctx context.Context, username string, req *clients.UpdateAdminUserRequest) (*clients.AdminUser, error) { return nil, nil }
func (m *MockClient) DeleteAdminUser(ctx context.Context, username string) error                                        { return nil }
func (m *MockClient) GetRepositoryWebhook(ctx context.Context, owner, repo string, id int64) (*clients.Webhook, error) { return nil, nil }
func (m *MockClient) CreateRepositoryWebhook(ctx context.Context, owner, repo string, req *clients.CreateWebhookRequest) (*clients.Webhook, error) { return nil, nil }
func (m *MockClient) UpdateRepositoryWebhook(ctx context.Context, owner, repo string, id int64, req *clients.UpdateWebhookRequest) (*clients.Webhook, error) { return nil, nil }
func (m *MockClient) DeleteRepositoryWebhook(ctx context.Context, owner, repo string, id int64) error { return nil }
func (m *MockClient) CreateOrganizationRepository(ctx context.Context, org string, req *clients.CreateRepositoryRequest) (*clients.Repository, error) { return nil, nil }
func (m *MockClient) GetTeam(ctx context.Context, teamID int64) (*clients.Team, error) { return nil, nil }
func (m *MockClient) CreateTeam(ctx context.Context, org string, req *clients.CreateTeamRequest) (*clients.Team, error) { return nil, nil }
func (m *MockClient) UpdateTeam(ctx context.Context, teamID int64, req *clients.UpdateTeamRequest) (*clients.Team, error) { return nil, nil }
func (m *MockClient) DeleteTeam(ctx context.Context, teamID int64) error { return nil }
func (m *MockClient) ListOrganizationTeams(ctx context.Context, org string) ([]*clients.Team, error) { return nil, nil }
func (m *MockClient) GetLabel(ctx context.Context, owner, repo string, labelID int64) (*clients.Label, error) { return nil, nil }
func (m *MockClient) CreateLabel(ctx context.Context, owner, repo string, req *clients.CreateLabelRequest) (*clients.Label, error) { return nil, nil }
func (m *MockClient) UpdateLabel(ctx context.Context, owner, repo string, labelID int64, req *clients.UpdateLabelRequest) (*clients.Label, error) { return nil, nil }
func (m *MockClient) DeleteLabel(ctx context.Context, owner, repo string, labelID int64) error { return nil }
func (m *MockClient) ListRepositoryLabels(ctx context.Context, owner, repo string) ([]*clients.Label, error) { return nil, nil }
func (m *MockClient) GetRepositoryCollaborator(ctx context.Context, owner, repo, username string) (*clients.RepositoryCollaborator, error) { return nil, nil }
func (m *MockClient) AddRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *clients.AddCollaboratorRequest) error { return nil }
func (m *MockClient) UpdateRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *clients.UpdateCollaboratorRequest) error { return nil }
func (m *MockClient) RemoveRepositoryCollaborator(ctx context.Context, owner, repo, username string) error { return nil }
func (m *MockClient) ListRepositoryCollaborators(ctx context.Context, owner, repo string) ([]*clients.RepositoryCollaborator, error) { return nil, nil }
func (m *MockClient) GetOrganizationSettings(ctx context.Context, org string) (*clients.OrganizationSettings, error) { return nil, nil }
func (m *MockClient) UpdateOrganizationSettings(ctx context.Context, org string, req *clients.UpdateOrganizationSettingsRequest) (*clients.OrganizationSettings, error) { return nil, nil }
func (m *MockClient) GetGitHook(ctx context.Context, repository, hookType string) (*clients.GitHook, error) { return nil, nil }
func (m *MockClient) CreateGitHook(ctx context.Context, repository string, req *clients.CreateGitHookRequest) (*clients.GitHook, error) { return nil, nil }
func (m *MockClient) UpdateGitHook(ctx context.Context, repository, hookType string, req *clients.UpdateGitHookRequest) (*clients.GitHook, error) { return nil, nil }
func (m *MockClient) DeleteGitHook(ctx context.Context, repository, hookType string) error { return nil }
func (m *MockClient) EnableAction(ctx context.Context, repository, workflowName string) error { return nil }
func (m *MockClient) DisableAction(ctx context.Context, repository, workflowName string) error { return nil }

// OrganizationSettings methods are already defined above

func TestObserve(t *testing.T) {
	cases := map[string]struct {
		client clients.Client
		mg     resource.Managed
		want   managed.ExternalObservation
		err    error
	}{
		"ObserveActionExists": {
			client: &MockClient{
				MockGetAction: func(ctx context.Context, repo, workflow string) (*clients.Action, error) {
					return &clients.Action{
						WorkflowName: "test.yml",
						State:        "active",
						WorkflowFile: clients.ActionWorkflowFile{
							Content: "name: Test\non: push",
						},
						LastRun: &clients.ActionLastRun{
							Status:     "success",
							Conclusion: "success",
						},
					}, nil
				},
			},
			mg: &v1alpha1.Action{
				Spec: v1alpha1.ActionSpec{
					ForProvider: v1alpha1.ActionParameters{
						Repository:   "owner/repo",
						WorkflowName: "test.yml",
						Content:      "name: Test\non: push",
					},
				},
			},
			want: managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: true,
			},
		},
		"ObserveActionNotFound": {
			client: &MockClient{
				MockGetAction: func(ctx context.Context, repo, workflow string) (*clients.Action, error) {
					return nil, clients.NewNotFoundError("action", workflow)
				},
			},
			mg: &v1alpha1.Action{
				Spec: v1alpha1.ActionSpec{
					ForProvider: v1alpha1.ActionParameters{
						Repository:   "owner/repo",
						WorkflowName: "test.yml",
					},
				},
			},
			want: managed.ExternalObservation{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Set external name for parsing
			meta.SetExternalName(tc.mg, "owner/repo/test.yml")
			
			e := external{client: tc.client}
			got, err := e.Observe(context.Background(), tc.mg)

			if diff := cmp.Diff(tc.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Observe(...): -want error, +got error:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.ResourceExists, got.ResourceExists); diff != "" {
				t.Errorf("Observe(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestExternalNameParsing(t *testing.T) {
	cases := map[string]struct {
		externalName     string
		expectedRepo     string
		expectedWorkflow string
		expectError      bool
	}{
		"ValidExternalName": {
			externalName:     "owner/repo/workflow.yml",
			expectedRepo:     "owner/repo",
			expectedWorkflow: "workflow.yml",
			expectError:      false,
		},
		"InvalidExternalName": {
			externalName: "invalid",
			expectError:  true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo, workflow, err := parseExternalName(tc.externalName)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if repo != tc.expectedRepo {
				t.Errorf("Expected repo %s, got %s", tc.expectedRepo, repo)
			}

			if workflow != tc.expectedWorkflow {
				t.Errorf("Expected workflow %s, got %s", tc.expectedWorkflow, workflow)
			}
		})
	}
}