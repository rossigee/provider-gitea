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

package clients

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/v1beta1"
)

const (
	// HTTP timeout for API requests
	defaultTimeout = 30 * time.Second

	// API paths
	apiPath = "/api/v1"
)

// Client interface for Gitea API operations
type Client interface {
	// Repository operations
	GetRepository(ctx context.Context, owner, name string) (*Repository, error)
	CreateRepository(ctx context.Context, req *CreateRepositoryRequest) (*Repository, error)
	CreateOrganizationRepository(ctx context.Context, org string, req *CreateRepositoryRequest) (*Repository, error)
	UpdateRepository(ctx context.Context, owner, name string, req *UpdateRepositoryRequest) (*Repository, error)
	DeleteRepository(ctx context.Context, owner, name string) error

	// Organization operations
	GetOrganization(ctx context.Context, name string) (*Organization, error)
	CreateOrganization(ctx context.Context, req *CreateOrganizationRequest) (*Organization, error)
	UpdateOrganization(ctx context.Context, name string, req *UpdateOrganizationRequest) (*Organization, error)
	DeleteOrganization(ctx context.Context, name string) error

	// User operations
	GetUser(ctx context.Context, username string) (*User, error)
	CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
	UpdateUser(ctx context.Context, username string, req *UpdateUserRequest) (*User, error)
	DeleteUser(ctx context.Context, username string) error

	// Webhook operations
	GetRepositoryWebhook(ctx context.Context, owner, repo string, id int64) (*Webhook, error)
	CreateRepositoryWebhook(ctx context.Context, owner, repo string, req *CreateWebhookRequest) (*Webhook, error)
	UpdateRepositoryWebhook(ctx context.Context, owner, repo string, id int64, req *UpdateWebhookRequest) (*Webhook, error)
	DeleteRepositoryWebhook(ctx context.Context, owner, repo string, id int64) error
	GetOrganizationWebhook(ctx context.Context, org string, id int64) (*Webhook, error)
	CreateOrganizationWebhook(ctx context.Context, org string, req *CreateWebhookRequest) (*Webhook, error)
	UpdateOrganizationWebhook(ctx context.Context, org string, id int64, req *UpdateWebhookRequest) (*Webhook, error)
	DeleteOrganizationWebhook(ctx context.Context, org string, id int64) error

	// Organization Secret operations
	GetOrganizationSecret(ctx context.Context, org, secretName string) (*OrganizationSecret, error)
	CreateOrganizationSecret(ctx context.Context, org, secretName string, req *CreateOrganizationSecretRequest) error
	UpdateOrganizationSecret(ctx context.Context, org, secretName string, req *CreateOrganizationSecretRequest) error
	DeleteOrganizationSecret(ctx context.Context, org, secretName string) error

	// Team operations
	GetTeam(ctx context.Context, teamID int64) (*Team, error)
	CreateTeam(ctx context.Context, org string, req *CreateTeamRequest) (*Team, error)
	UpdateTeam(ctx context.Context, teamID int64, req *UpdateTeamRequest) (*Team, error)
	DeleteTeam(ctx context.Context, teamID int64) error

	// Label operations
	GetLabel(ctx context.Context, owner, repo string, labelID int64) (*Label, error)
	CreateLabel(ctx context.Context, owner, repo string, req *CreateLabelRequest) (*Label, error)
	UpdateLabel(ctx context.Context, owner, repo string, labelID int64, req *UpdateLabelRequest) (*Label, error)
	DeleteLabel(ctx context.Context, owner, repo string, labelID int64) error

	// Repository Collaborator operations
	GetRepositoryCollaborator(ctx context.Context, owner, repo, username string) (*RepositoryCollaborator, error)
	AddRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *AddCollaboratorRequest) error
	UpdateRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *UpdateCollaboratorRequest) error
	RemoveRepositoryCollaborator(ctx context.Context, owner, repo, username string) error

	// Organization Settings operations
	GetOrganizationSettings(ctx context.Context, org string) (*OrganizationSettings, error)
	UpdateOrganizationSettings(ctx context.Context, org string, req *UpdateOrganizationSettingsRequest) (*OrganizationSettings, error)

	// Git Hooks operations
	GetGitHook(ctx context.Context, repository, hookType string) (*GitHook, error)
	CreateGitHook(ctx context.Context, repository string, req *CreateGitHookRequest) (*GitHook, error)
	UpdateGitHook(ctx context.Context, repository, hookType string, req *UpdateGitHookRequest) (*GitHook, error)
	DeleteGitHook(ctx context.Context, repository, hookType string) error

	// Branch Protection operations
	GetBranchProtection(ctx context.Context, repository, branch string) (*BranchProtection, error)
	CreateBranchProtection(ctx context.Context, repository, branch string, req *CreateBranchProtectionRequest) (*BranchProtection, error)
	UpdateBranchProtection(ctx context.Context, repository, branch string, req *UpdateBranchProtectionRequest) (*BranchProtection, error)
	DeleteBranchProtection(ctx context.Context, repository, branch string) error

	// Repository Key operations
	GetRepositoryKey(ctx context.Context, repository string, keyID int64) (*RepositoryKey, error)
	CreateRepositoryKey(ctx context.Context, repository string, req *CreateRepositoryKeyRequest) (*RepositoryKey, error)
	UpdateRepositoryKey(ctx context.Context, repository string, keyID int64, req *UpdateRepositoryKeyRequest) (*RepositoryKey, error)
	DeleteRepositoryKey(ctx context.Context, repository string, keyID int64) error

	// Access Token operations
	GetAccessToken(ctx context.Context, username string, tokenID int64) (*AccessToken, error)
	CreateAccessToken(ctx context.Context, username string, req *CreateAccessTokenRequest) (*AccessToken, error)
	UpdateAccessToken(ctx context.Context, username string, tokenID int64, req *UpdateAccessTokenRequest) (*AccessToken, error)
	DeleteAccessToken(ctx context.Context, username string, tokenID int64) error

	// Repository Secret operations
	GetRepositorySecret(ctx context.Context, repository, secretName string) (*RepositorySecret, error)
	CreateRepositorySecret(ctx context.Context, repository, secretName string, req *CreateRepositorySecretRequest) error
	UpdateRepositorySecret(ctx context.Context, repository, secretName string, req *UpdateRepositorySecretRequest) error
	DeleteRepositorySecret(ctx context.Context, repository, secretName string) error

	// Organization Actions Variable operations
	GetOrganizationVariable(ctx context.Context, org, name string) (*Variable, error)
	CreateOrganizationVariable(ctx context.Context, org, name string, req *VariableRequest) error
	UpdateOrganizationVariable(ctx context.Context, org, name string, req *VariableRequest) error
	DeleteOrganizationVariable(ctx context.Context, org, name string) error

	// Repository Actions Variable operations
	GetRepositoryVariable(ctx context.Context, owner, repo, name string) (*Variable, error)
	CreateRepositoryVariable(ctx context.Context, owner, repo, name string, req *VariableRequest) error
	UpdateRepositoryVariable(ctx context.Context, owner, repo, name string, req *VariableRequest) error
	DeleteRepositoryVariable(ctx context.Context, owner, repo, name string) error

	// Organization Labels operations
	GetOrganizationLabel(ctx context.Context, org string, labelID int64) (*Label, error)
	CreateOrganizationLabel(ctx context.Context, org string, req *CreateLabelRequest) (*Label, error)
	UpdateOrganizationLabel(ctx context.Context, org string, labelID int64, req *UpdateLabelRequest) (*Label, error)
	DeleteOrganizationLabel(ctx context.Context, org string, labelID int64) error
}

// giteaClient implements the Client interface
type giteaClient struct {
	httpClient *http.Client
	baseURL    string
	token      string

	// basicUser/basicPass, when set, switch authentication from the
	// ProviderConfig token to HTTP basic auth as a specific user (see doRequest).
	basicUser string
	basicPass string
}

// NewClient creates a new Gitea API client
func NewClient(ctx context.Context, cfg *v1beta1.ProviderConfig, kube client.Client) (Client, error) {
	if cfg.Spec.BaseURL == "" {
		return nil, errors.New("baseURL is required")
	}

	// Get authentication token from secret
	token, err := getTokenFromSecret(ctx, cfg, kube)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get authentication token")
	}

	// Setup HTTP client
	httpClient := &http.Client{
		Timeout: defaultTimeout,
	}

	// Handle insecure connections
	if cfg.Spec.Insecure != nil && *cfg.Spec.Insecure {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient.Transport = transport
	}

	baseURL := strings.TrimSuffix(cfg.Spec.BaseURL, "/") + apiPath

	return &giteaClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		token:      token,
	}, nil
}

// NewBasicAuthClient builds a client that authenticates with HTTP basic auth as
// a specific user instead of the ProviderConfig token. It is required for
// endpoints Gitea gates on the owning user's credentials (e.g. access-token
// CRUD) and is reusable by any future user-context resource.
func NewBasicAuthClient(baseURL, username, password string, insecure bool) (Client, error) {
	if baseURL == "" {
		return nil, errors.New("baseURL is required")
	}
	if username == "" || password == "" {
		return nil, errors.New("username and password are required for basic auth")
	}

	httpClient := &http.Client{Timeout: defaultTimeout}
	if insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return &giteaClient{
		httpClient: httpClient,
		baseURL:    strings.TrimSuffix(baseURL, "/") + apiPath,
		basicUser:  username,
		basicPass:  password,
	}, nil
}

// ResolveSecretValue reads a single value from the Kubernetes Secret referenced
// by a SecretKeySelector. It is the one canonical way every controller turns a
// `*SecretRef` field into a plaintext value — secrets are NEVER taken from the
// spec directly (see the secret-ref rule in the provider template).
func ResolveSecretValue(ctx context.Context, kube client.Client, sel *xpv1.SecretKeySelector) (string, error) {
	if sel == nil {
		return "", errors.New("secret reference is not set")
	}
	var sec corev1.Secret
	if err := kube.Get(ctx, client.ObjectKey{Namespace: sel.Namespace, Name: sel.Name}, &sec); err != nil {
		return "", errors.Wrap(err, "failed to read referenced secret")
	}
	v, ok := sec.Data[sel.Key]
	if !ok {
		return "", errors.Errorf("key %q not found in secret %s/%s", sel.Key, sel.Namespace, sel.Name)
	}
	return string(v), nil
}

// Repository represents a Gitea repository
type Repository struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	Fork        bool   `json:"fork"`
	Template    bool   `json:"template"`
	Empty       bool   `json:"empty"`
	Archived    bool   `json:"archived"`
	Size        int    `json:"size"`
	HTMLURL     string `json:"html_url"`
	SSHURL      string `json:"ssh_url"`
	CloneURL    string `json:"clone_url"`
	Website     string `json:"website"`
	Language    string `json:"language"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Owner       *User  `json:"owner"`
}

// CreateRepositoryRequest represents the request body for creating a repository
type CreateRepositoryRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Private       bool   `json:"private,omitempty"`
	AutoInit      bool   `json:"auto_init,omitempty"`
	Template      bool   `json:"template,omitempty"`
	Gitignores    string `json:"gitignores,omitempty"`
	License       string `json:"license,omitempty"`
	Readme        string `json:"readme,omitempty"`
	IssueLabels   string `json:"issue_labels,omitempty"`
	TrustModel    string `json:"trust_model,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

// UpdateRepositoryRequest represents the request body for updating a repository
type UpdateRepositoryRequest struct {
	Name                          *string `json:"name,omitempty"`
	Description                   *string `json:"description,omitempty"`
	Website                       *string `json:"website,omitempty"`
	Private                       *bool   `json:"private,omitempty"`
	Template                      *bool   `json:"template,omitempty"`
	HasIssues                     *bool   `json:"has_issues,omitempty"`
	HasWiki                       *bool   `json:"has_wiki,omitempty"`
	HasPullRequests               *bool   `json:"has_pull_requests,omitempty"`
	HasProjects                   *bool   `json:"has_projects,omitempty"`
	HasReleases                   *bool   `json:"has_releases,omitempty"`
	HasPackages                   *bool   `json:"has_packages,omitempty"`
	HasActions                    *bool   `json:"has_actions,omitempty"`
	AllowMergeCommits             *bool   `json:"allow_merge_commits,omitempty"`
	AllowRebase                   *bool   `json:"allow_rebase,omitempty"`
	AllowRebaseExplicit           *bool   `json:"allow_rebase_explicit,omitempty"`
	AllowSquashMerge              *bool   `json:"allow_squash_merge,omitempty"`
	AllowRebaseUpdate             *bool   `json:"allow_rebase_update,omitempty"`
	DefaultDeleteBranchAfterMerge *bool   `json:"default_delete_branch_after_merge,omitempty"`
	DefaultMergeStyle             *string `json:"default_merge_style,omitempty"`
	DefaultBranch                 *string `json:"default_branch,omitempty"`
	Archived                      *bool   `json:"archived,omitempty"`
}

// Organization represents a Gitea organization
type Organization struct {
	ID                        int64  `json:"id"`
	Username                  string `json:"username"`
	Name                      string `json:"name"`
	FullName                  string `json:"full_name"`
	Description               string `json:"description"`
	Website                   string `json:"website"`
	Location                  string `json:"location"`
	Visibility                string `json:"visibility"`
	RepoAdminChangeTeamAccess bool   `json:"repo_admin_change_team_access"`
	Email                     string `json:"email"`
	AvatarURL                 string `json:"avatar_url"`
}

// CreateOrganizationRequest represents the request body for creating an organization
type CreateOrganizationRequest struct {
	Username                  string `json:"username"`
	Name                      string `json:"name,omitempty"`
	FullName                  string `json:"full_name,omitempty"`
	Description               string `json:"description,omitempty"`
	Website                   string `json:"website,omitempty"`
	Location                  string `json:"location,omitempty"`
	Visibility                string `json:"visibility,omitempty"`
	RepoAdminChangeTeamAccess bool   `json:"repo_admin_change_team_access,omitempty"`
}

// UpdateOrganizationRequest represents the request body for updating an organization
type UpdateOrganizationRequest struct {
	Name                      *string `json:"name,omitempty"`
	FullName                  *string `json:"full_name,omitempty"`
	Description               *string `json:"description,omitempty"`
	Website                   *string `json:"website,omitempty"`
	Location                  *string `json:"location,omitempty"`
	Visibility                *string `json:"visibility,omitempty"`
	RepoAdminChangeTeamAccess *bool   `json:"repo_admin_change_team_access,omitempty"`
}

// User represents a Gitea user
type User struct {
	ID            int64  `json:"id"`
	Username      string `json:"username"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Email         string `json:"email"`
	AvatarURL     string `json:"avatar_url"`
	Website       string `json:"website"`
	Location      string `json:"location"`
	IsAdmin       bool   `json:"is_admin"`
	LastLogin     string `json:"last_login"`
	Created       string `json:"created"`
	Restricted    bool   `json:"restricted"`
	Active        bool   `json:"active"`
	ProhibitLogin bool   `json:"prohibit_login"`
	LoginName     string `json:"login_name"`
	SourceID      int64  `json:"source_id"`
	Language      string `json:"language"`
	Description   string `json:"description"`
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Username           string `json:"username"`
	Email              string `json:"email"`
	FullName           string `json:"full_name,omitempty"`
	Password           string `json:"password"`
	LoginName          string `json:"login_name,omitempty"`
	SendNotify         bool   `json:"send_notify,omitempty"`
	SourceID           int64  `json:"source_id,omitempty"`
	MustChangePassword bool   `json:"must_change_password,omitempty"`
	Restricted         bool   `json:"restricted,omitempty"`
	Visibility         string `json:"visibility,omitempty"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	// Password rotates the user's password. The Gitea admin edit-user API
	// (PATCH /admin/users/{username}) requires login_name (and source_id) to be
	// present alongside password, so the controller always sets those when it
	// pushes a rotation.
	Password                string  `json:"password,omitempty"`
	Email                   *string `json:"email,omitempty"`
	FullName                *string `json:"full_name,omitempty"`
	LoginName               *string `json:"login_name,omitempty"`
	SourceID                *int64  `json:"source_id,omitempty"`
	Active                  *bool   `json:"active,omitempty"`
	Admin                   *bool   `json:"admin,omitempty"`
	AllowGitHook            *bool   `json:"allow_git_hook,omitempty"`
	AllowImportLocal        *bool   `json:"allow_import_local,omitempty"`
	AllowCreateOrganization *bool   `json:"allow_create_organization,omitempty"`
	ProhibitLogin           *bool   `json:"prohibit_login,omitempty"`
	Restricted              *bool   `json:"restricted,omitempty"`
	Website                 *string `json:"website,omitempty"`
	Location                *string `json:"location,omitempty"`
	Description             *string `json:"description,omitempty"`
	Visibility              *string `json:"visibility,omitempty"`
	MaxRepoCreation         *int    `json:"max_repo_creation,omitempty"`
}

// Webhook represents a Gitea webhook
type Webhook struct {
	ID        int64             `json:"id"`
	Type      string            `json:"type"`
	URL       string            `json:"config.url"`
	Active    bool              `json:"active"`
	Events    []string          `json:"events"`
	Config    map[string]string `json:"config"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

// CreateWebhookRequest represents the request body for creating a webhook
type CreateWebhookRequest struct {
	Type   string            `json:"type"`
	Config map[string]string `json:"config"`
	Events []string          `json:"events"`
	Active bool              `json:"active"`
}

// UpdateWebhookRequest represents the request body for updating a webhook
type UpdateWebhookRequest struct {
	Config *map[string]string `json:"config,omitempty"`
	Events *[]string          `json:"events,omitempty"`
	Active *bool              `json:"active,omitempty"`
}

// OrganizationSecret represents a Gitea organization action secret
type OrganizationSecret struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateOrganizationSecretRequest represents the request body for creating/updating an organization secret
type CreateOrganizationSecretRequest struct {
	Data string `json:"data"`
}

// Organization Secret API methods
func (c *giteaClient) GetOrganizationSecret(ctx context.Context, org, secretName string) (*OrganizationSecret, error) {
	// Gitea has no GET single secret (405). List org secrets and match by name.
	path := "/orgs/" + org + "/actions/secrets"
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var list []OrganizationSecret
	if err := handleResponse(resp, &list); err != nil {
		return nil, err
	}
	for i := range list {
		if strings.EqualFold(list[i].Name, secretName) {
			return &list[i], nil
		}
	}
	return nil, NewNotFoundError("organization secret", secretName)
}

func (c *giteaClient) CreateOrganizationSecret(ctx context.Context, org, secretName string, req *CreateOrganizationSecretRequest) error {
	path := "/orgs/" + org + "/actions/secrets/" + secretName
	resp, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

func (c *giteaClient) UpdateOrganizationSecret(ctx context.Context, org, secretName string, req *CreateOrganizationSecretRequest) error {
	// For Gitea API, create and update use the same PUT endpoint
	return c.CreateOrganizationSecret(ctx, org, secretName, req)
}

func (c *giteaClient) DeleteOrganizationSecret(ctx context.Context, org, secretName string) error {
	path := "/orgs/" + org + "/actions/secrets/" + secretName
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Variable represents a Gitea Actions variable (org- or repo-scoped). Unlike a
// secret, the value (Data) is readable back from a GET, enabling real drift
// detection.
type Variable struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Data    string `json:"data"`
	OwnerID int64  `json:"owner_id"`
	RepoID  int64  `json:"repo_id"`
}

// VariableRequest is the request body for creating/updating an Actions variable.
type VariableRequest struct {
	Value string `json:"value"`
}

// Organization Actions Variable API methods
func (c *giteaClient) GetOrganizationVariable(ctx context.Context, org, name string) (*Variable, error) {
	path := "/orgs/" + org + "/actions/variables/" + name
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Body: "organization variable not found"}
	}

	var v Variable
	if err := handleResponse(resp, &v); err != nil {
		return nil, err
	}
	// Gitea returns the variable name in the path, not always in the body.
	if v.Name == "" {
		v.Name = name
	}
	return &v, nil
}

func (c *giteaClient) CreateOrganizationVariable(ctx context.Context, org, name string, req *VariableRequest) error {
	path := "/orgs/" + org + "/actions/variables/" + name
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return err
	}
	return handleResponse(resp, nil)
}

func (c *giteaClient) UpdateOrganizationVariable(ctx context.Context, org, name string, req *VariableRequest) error {
	path := "/orgs/" + org + "/actions/variables/" + name
	resp, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return err
	}
	return handleResponse(resp, nil)
}

func (c *giteaClient) DeleteOrganizationVariable(ctx context.Context, org, name string) error {
	path := "/orgs/" + org + "/actions/variables/" + name
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	return handleResponse(resp, nil)
}

// Repository Actions Variable API methods
func (c *giteaClient) GetRepositoryVariable(ctx context.Context, owner, repo, name string) (*Variable, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/variables/%s", owner, repo, name)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Body: "repository variable not found"}
	}

	var v Variable
	if err := handleResponse(resp, &v); err != nil {
		return nil, err
	}
	if v.Name == "" {
		v.Name = name
	}
	return &v, nil
}

func (c *giteaClient) CreateRepositoryVariable(ctx context.Context, owner, repo, name string, req *VariableRequest) error {
	path := fmt.Sprintf("/repos/%s/%s/actions/variables/%s", owner, repo, name)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return err
	}
	return handleResponse(resp, nil)
}

func (c *giteaClient) UpdateRepositoryVariable(ctx context.Context, owner, repo, name string, req *VariableRequest) error {
	path := fmt.Sprintf("/repos/%s/%s/actions/variables/%s", owner, repo, name)
	resp, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return err
	}
	return handleResponse(resp, nil)
}

func (c *giteaClient) DeleteRepositoryVariable(ctx context.Context, owner, repo, name string) error {
	path := fmt.Sprintf("/repos/%s/%s/actions/variables/%s", owner, repo, name)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	return handleResponse(resp, nil)
}

// Team represents a Gitea organization team
type Team struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Organization struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
	} `json:"organization"`
	Permission              string            `json:"permission"`
	CanCreateOrgRepo        bool              `json:"can_create_org_repo"`
	IncludesAllRepositories bool              `json:"includes_all_repositories"`
	UnitsMap                map[string]string `json:"units_map"`
}

// CreateTeamRequest represents the request body for creating a team
type CreateTeamRequest struct {
	Name                    string   `json:"name"`
	Description             string   `json:"description,omitempty"`
	Permission              string   `json:"permission,omitempty"`
	CanCreateOrgRepo        bool     `json:"can_create_org_repo,omitempty"`
	IncludesAllRepositories bool     `json:"includes_all_repositories,omitempty"`
	Units                   []string `json:"units,omitempty"`
}

// UpdateTeamRequest represents the request body for updating a team
type UpdateTeamRequest struct {
	Name                    *string  `json:"name,omitempty"`
	Description             *string  `json:"description,omitempty"`
	Permission              *string  `json:"permission,omitempty"`
	CanCreateOrgRepo        *bool    `json:"can_create_org_repo,omitempty"`
	IncludesAllRepositories *bool    `json:"includes_all_repositories,omitempty"`
	Units                   []string `json:"units,omitempty"`
}

// Team API methods
func (c *giteaClient) GetTeam(ctx context.Context, teamID int64) (*Team, error) {
	path := fmt.Sprintf("/teams/%d", teamID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Body: "team not found"}
	}

	var team Team
	if err := handleResponse(resp, &team); err != nil {
		return nil, err
	}

	return &team, nil
}

func (c *giteaClient) CreateTeam(ctx context.Context, org string, req *CreateTeamRequest) (*Team, error) {
	path := "/orgs/" + org + "/teams"
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var team Team
	if err := handleResponse(resp, &team); err != nil {
		return nil, err
	}

	return &team, nil
}

func (c *giteaClient) UpdateTeam(ctx context.Context, teamID int64, req *UpdateTeamRequest) (*Team, error) {
	path := fmt.Sprintf("/teams/%d", teamID)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var team Team
	if err := handleResponse(resp, &team); err != nil {
		return nil, err
	}

	return &team, nil
}

func (c *giteaClient) DeleteTeam(ctx context.Context, teamID int64) error {
	path := fmt.Sprintf("/teams/%d", teamID)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Label represents a Gitea repository label
type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	Exclusive   bool   `json:"exclusive"`
	URL         string `json:"url"`
}

// CreateLabelRequest represents the request body for creating a label
type CreateLabelRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
	Exclusive   bool   `json:"exclusive,omitempty"`
}

// UpdateLabelRequest represents the request body for updating a label
type UpdateLabelRequest struct {
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
	Exclusive   *bool   `json:"exclusive,omitempty"`
}

// RepositoryCollaborator represents a repository collaborator
type RepositoryCollaborator struct {
	Username    string                            `json:"login"`
	FullName    string                            `json:"full_name"`
	Email       string                            `json:"email"`
	AvatarURL   string                            `json:"avatar_url"`
	Permissions RepositoryCollaboratorPermissions `json:"permissions"`
}

// RepositoryCollaboratorPermissions represents the permissions of a collaborator
type RepositoryCollaboratorPermissions struct {
	Admin bool `json:"admin"`
	Push  bool `json:"push"`
	Pull  bool `json:"pull"`
}

// AddCollaboratorRequest represents the request body for adding a collaborator
type AddCollaboratorRequest struct {
	Permission string `json:"permission"` // read, write, admin
}

// UpdateCollaboratorRequest represents the request body for updating a collaborator
type UpdateCollaboratorRequest struct {
	Permission string `json:"permission"` // read, write, admin
}

// OrganizationSettings represents organization-wide settings and policies
type OrganizationSettings struct {
	// Default repository settings
	DefaultRepoPermission    string `json:"default_repo_permission"` // read, write, admin
	MembersCanCreateRepos    bool   `json:"members_can_create_repos"`
	MembersCanCreatePrivate  bool   `json:"members_can_create_private"`
	MembersCanCreateInternal bool   `json:"members_can_create_internal"`

	// Member management settings
	MembersCanDeleteRepos bool `json:"members_can_delete_repos"`
	MembersCanFork        bool `json:"members_can_fork"`
	MembersCanCreatePages bool `json:"members_can_create_pages"`

	// Security and visibility settings
	DefaultRepoVisibility string `json:"default_repo_visibility"` // public, private, internal
	RequireSignedCommits  bool   `json:"require_signed_commits"`
	EnableDependencyGraph bool   `json:"enable_dependency_graph"`

	// Git hooks and automation
	AllowGitHooks       bool `json:"allow_git_hooks"`
	AllowCustomGitHooks bool `json:"allow_custom_git_hooks"`
}

// UpdateOrganizationSettingsRequest represents the request body for updating organization settings
type UpdateOrganizationSettingsRequest struct {
	DefaultRepoPermission    *string `json:"default_repo_permission,omitempty"`
	MembersCanCreateRepos    *bool   `json:"members_can_create_repos,omitempty"`
	MembersCanCreatePrivate  *bool   `json:"members_can_create_private,omitempty"`
	MembersCanCreateInternal *bool   `json:"members_can_create_internal,omitempty"`
	MembersCanDeleteRepos    *bool   `json:"members_can_delete_repos,omitempty"`
	MembersCanFork           *bool   `json:"members_can_fork,omitempty"`
	MembersCanCreatePages    *bool   `json:"members_can_create_pages,omitempty"`
	DefaultRepoVisibility    *string `json:"default_repo_visibility,omitempty"`
	RequireSignedCommits     *bool   `json:"require_signed_commits,omitempty"`
	EnableDependencyGraph    *bool   `json:"enable_dependency_graph,omitempty"`
	AllowGitHooks            *bool   `json:"allow_git_hooks,omitempty"`
	AllowCustomGitHooks      *bool   `json:"allow_custom_git_hooks,omitempty"`
}

// GitHook represents a Git hook
type GitHook struct {
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
	Content  string `json:"content"`
	Type     string `json:"type"` // pre-receive, update, post-receive, pre-push, post-update
}

// CreateGitHookRequest represents the request body for creating a Git hook
type CreateGitHookRequest struct {
	HookType string `json:"hook_type"`
	Content  string `json:"content"`
	IsActive bool   `json:"is_active"`
}

// UpdateGitHookRequest represents the request body for updating a Git hook
type UpdateGitHookRequest struct {
	Content  string `json:"content"`
	IsActive bool   `json:"is_active"`
}

// BranchProtection represents a Git branch protection rule
type BranchProtection struct {
	RuleName                      string   `json:"rule_name"`
	EnablePush                    bool     `json:"enable_push"`
	EnablePushWhitelist           bool     `json:"enable_push_whitelist"`
	PushWhitelistUsernames        []string `json:"push_whitelist_usernames"`
	PushWhitelistTeams            []string `json:"push_whitelist_teams"`
	PushWhitelistDeployKeys       bool     `json:"push_whitelist_deploy_keys"`
	EnableMergeWhitelist          bool     `json:"enable_merge_whitelist"`
	MergeWhitelistUsernames       []string `json:"merge_whitelist_usernames"`
	MergeWhitelistTeams           []string `json:"merge_whitelist_teams"`
	EnableStatusCheck             bool     `json:"enable_status_check"`
	StatusCheckContexts           []string `json:"status_check_contexts"`
	RequiredApprovals             int      `json:"required_approvals"`
	EnableApprovalsWhitelist      bool     `json:"enable_approvals_whitelist"`
	ApprovalsWhitelistUsernames   []string `json:"approvals_whitelist_usernames"`
	ApprovalsWhitelistTeams       []string `json:"approvals_whitelist_teams"`
	BlockOnRejectedReviews        bool     `json:"block_on_rejected_reviews"`
	BlockOnOfficialReviewRequests bool     `json:"block_on_official_review_requests"`
	BlockOnOutdatedBranch         bool     `json:"block_on_outdated_branch"`
	DismissStaleApprovals         bool     `json:"dismiss_stale_approvals"`
	RequireSignedCommits          bool     `json:"require_signed_commits"`
	ProtectedFilePatterns         string   `json:"protected_file_patterns"`
	UnprotectedFilePatterns       string   `json:"unprotected_file_patterns"`
	CreatedAt                     string   `json:"created_at"`
	UpdatedAt                     string   `json:"updated_at"`
}

// CreateBranchProtectionRequest represents the request body for creating branch protection
type CreateBranchProtectionRequest struct {
	RuleName                      string   `json:"rule_name"`
	EnablePush                    *bool    `json:"enable_push,omitempty"`
	EnablePushWhitelist           *bool    `json:"enable_push_whitelist,omitempty"`
	PushWhitelistUsernames        []string `json:"push_whitelist_usernames,omitempty"`
	PushWhitelistTeams            []string `json:"push_whitelist_teams,omitempty"`
	PushWhitelistDeployKeys       *bool    `json:"push_whitelist_deploy_keys,omitempty"`
	EnableMergeWhitelist          *bool    `json:"enable_merge_whitelist,omitempty"`
	MergeWhitelistUsernames       []string `json:"merge_whitelist_usernames,omitempty"`
	MergeWhitelistTeams           []string `json:"merge_whitelist_teams,omitempty"`
	EnableStatusCheck             *bool    `json:"enable_status_check,omitempty"`
	StatusCheckContexts           []string `json:"status_check_contexts,omitempty"`
	RequiredApprovals             *int     `json:"required_approvals,omitempty"`
	EnableApprovalsWhitelist      *bool    `json:"enable_approvals_whitelist,omitempty"`
	ApprovalsWhitelistUsernames   []string `json:"approvals_whitelist_usernames,omitempty"`
	ApprovalsWhitelistTeams       []string `json:"approvals_whitelist_teams,omitempty"`
	BlockOnRejectedReviews        *bool    `json:"block_on_rejected_reviews,omitempty"`
	BlockOnOfficialReviewRequests *bool    `json:"block_on_official_review_requests,omitempty"`
	BlockOnOutdatedBranch         *bool    `json:"block_on_outdated_branch,omitempty"`
	DismissStaleApprovals         *bool    `json:"dismiss_stale_approvals,omitempty"`
	RequireSignedCommits          *bool    `json:"require_signed_commits,omitempty"`
	ProtectedFilePatterns         *string  `json:"protected_file_patterns,omitempty"`
	UnprotectedFilePatterns       *string  `json:"unprotected_file_patterns,omitempty"`
}

// UpdateBranchProtectionRequest represents the request body for updating branch protection
type UpdateBranchProtectionRequest struct {
	EnablePush                    *bool    `json:"enable_push,omitempty"`
	EnablePushWhitelist           *bool    `json:"enable_push_whitelist,omitempty"`
	PushWhitelistUsernames        []string `json:"push_whitelist_usernames,omitempty"`
	PushWhitelistTeams            []string `json:"push_whitelist_teams,omitempty"`
	PushWhitelistDeployKeys       *bool    `json:"push_whitelist_deploy_keys,omitempty"`
	EnableMergeWhitelist          *bool    `json:"enable_merge_whitelist,omitempty"`
	MergeWhitelistUsernames       []string `json:"merge_whitelist_usernames,omitempty"`
	MergeWhitelistTeams           []string `json:"merge_whitelist_teams,omitempty"`
	EnableStatusCheck             *bool    `json:"enable_status_check,omitempty"`
	StatusCheckContexts           []string `json:"status_check_contexts,omitempty"`
	RequiredApprovals             *int     `json:"required_approvals,omitempty"`
	EnableApprovalsWhitelist      *bool    `json:"enable_approvals_whitelist,omitempty"`
	ApprovalsWhitelistUsernames   []string `json:"approvals_whitelist_usernames,omitempty"`
	ApprovalsWhitelistTeams       []string `json:"approvals_whitelist_teams,omitempty"`
	BlockOnRejectedReviews        *bool    `json:"block_on_rejected_reviews,omitempty"`
	BlockOnOfficialReviewRequests *bool    `json:"block_on_official_review_requests,omitempty"`
	BlockOnOutdatedBranch         *bool    `json:"block_on_outdated_branch,omitempty"`
	DismissStaleApprovals         *bool    `json:"dismiss_stale_approvals,omitempty"`
	RequireSignedCommits          *bool    `json:"require_signed_commits,omitempty"`
	ProtectedFilePatterns         *string  `json:"protected_file_patterns,omitempty"`
	UnprotectedFilePatterns       *string  `json:"unprotected_file_patterns,omitempty"`
}

// RepositoryKey represents a repository SSH key
type RepositoryKey struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"created_at"`
	ReadOnly    bool   `json:"read_only"`
}

// CreateRepositoryKeyRequest represents the request body for creating a repository key
type CreateRepositoryKeyRequest struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	ReadOnly *bool  `json:"read_only,omitempty"`
}

// UpdateRepositoryKeyRequest represents the request body for updating a repository key
type UpdateRepositoryKeyRequest struct {
	Title    *string `json:"title,omitempty"`
	ReadOnly *bool   `json:"read_only,omitempty"`
}

// AccessToken represents an API access token
type AccessToken struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	Sha1           string   `json:"sha1"`
	Token          string   `json:"token,omitempty"` // Only returned on creation
	TokenLastEight string   `json:"token_last_eight"`
	Scopes         []string `json:"scopes"`
}

// CreateAccessTokenRequest represents the request body for creating an access token
type CreateAccessTokenRequest struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes,omitempty"`
}

// UpdateAccessTokenRequest represents the request body for updating an access token
type UpdateAccessTokenRequest struct {
	Name   *string  `json:"name,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
}

// RepositorySecret represents a repository action secret
type RepositorySecret struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// CreateRepositorySecretRequest represents the request body for creating a repository secret
type CreateRepositorySecretRequest struct {
	Data string `json:"data"`
}

// UpdateRepositorySecretRequest represents the request body for updating a repository secret
type UpdateRepositorySecretRequest struct {
	Data string `json:"data"`
}

// Label API methods
func (c *giteaClient) GetLabel(ctx context.Context, owner, repo string, labelID int64) (*Label, error) {
	path := fmt.Sprintf("/repos/%s/%s/labels/%d", owner, repo, labelID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Body: "label not found"}
	}

	var label Label
	if err := handleResponse(resp, &label); err != nil {
		return nil, err
	}

	return &label, nil
}

func (c *giteaClient) CreateLabel(ctx context.Context, owner, repo string, req *CreateLabelRequest) (*Label, error) {
	path := fmt.Sprintf("/repos/%s/%s/labels", owner, repo)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var label Label
	if err := handleResponse(resp, &label); err != nil {
		return nil, err
	}

	return &label, nil
}

func (c *giteaClient) UpdateLabel(ctx context.Context, owner, repo string, labelID int64, req *UpdateLabelRequest) (*Label, error) {
	path := fmt.Sprintf("/repos/%s/%s/labels/%d", owner, repo, labelID)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var label Label
	if err := handleResponse(resp, &label); err != nil {
		return nil, err
	}

	return &label, nil
}

func (c *giteaClient) DeleteLabel(ctx context.Context, owner, repo string, labelID int64) error {
	path := fmt.Sprintf("/repos/%s/%s/labels/%d", owner, repo, labelID)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Organization Label API methods. Org labels mirror repo labels but live under
// /orgs/{org}/labels; the org-label PATCH/DELETE endpoints are keyed by the
// numeric label id, exactly like the repo variants.
func (c *giteaClient) GetOrganizationLabel(ctx context.Context, org string, labelID int64) (*Label, error) {
	path := fmt.Sprintf("/orgs/%s/labels/%d", org, labelID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Body: "organization label not found"}
	}

	var label Label
	if err := handleResponse(resp, &label); err != nil {
		return nil, err
	}

	return &label, nil
}

func (c *giteaClient) CreateOrganizationLabel(ctx context.Context, org string, req *CreateLabelRequest) (*Label, error) {
	path := fmt.Sprintf("/orgs/%s/labels", org)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var label Label
	if err := handleResponse(resp, &label); err != nil {
		return nil, err
	}

	return &label, nil
}

func (c *giteaClient) UpdateOrganizationLabel(ctx context.Context, org string, labelID int64, req *UpdateLabelRequest) (*Label, error) {
	path := fmt.Sprintf("/orgs/%s/labels/%d", org, labelID)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var label Label
	if err := handleResponse(resp, &label); err != nil {
		return nil, err
	}

	return &label, nil
}

func (c *giteaClient) DeleteOrganizationLabel(ctx context.Context, org string, labelID int64) error {
	path := fmt.Sprintf("/orgs/%s/labels/%d", org, labelID)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Repository Collaborator API methods
func (c *giteaClient) GetRepositoryCollaborator(ctx context.Context, owner, repo, username string) (*RepositoryCollaborator, error) {
	path := fmt.Sprintf("/repos/%s/%s/collaborators/%s", owner, repo, username)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	// 404 = not a collaborator; 422 = the user does not exist yet (a transient
	// during parallel apply). Both mean "no such collaborator" — return typed
	// not-found so Observe reports not-created cleanly instead of erroring.
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnprocessableEntity {
		return nil, &APIError{StatusCode: http.StatusNotFound, Body: "collaborator not found"}
	}

	var collaborator RepositoryCollaborator
	if err := handleResponse(resp, &collaborator); err != nil {
		return nil, err
	}

	return &collaborator, nil
}

func (c *giteaClient) AddRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *AddCollaboratorRequest) error {
	path := fmt.Sprintf("/repos/%s/%s/collaborators/%s", owner, repo, username)
	resp, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

func (c *giteaClient) UpdateRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *UpdateCollaboratorRequest) error {
	path := fmt.Sprintf("/repos/%s/%s/collaborators/%s", owner, repo, username)
	resp, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

func (c *giteaClient) RemoveRepositoryCollaborator(ctx context.Context, owner, repo, username string) error {
	path := fmt.Sprintf("/repos/%s/%s/collaborators/%s", owner, repo, username)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Organization Settings API methods
func (c *giteaClient) GetOrganizationSettings(ctx context.Context, org string) (*OrganizationSettings, error) {
	// Note: This is a conceptual implementation - Gitea API may not have a single endpoint
	// for all organization settings. This might need to aggregate multiple API calls.
	path := fmt.Sprintf("/orgs/%s", org)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	// For now, we'll get the basic organization and map what we can
	var orgData Organization
	if err := handleResponse(resp, &orgData); err != nil {
		return nil, err
	}

	// Map Organization data to OrganizationSettings
	// Note: This is a simplified mapping - real implementation would aggregate multiple endpoints
	settings := &OrganizationSettings{
		DefaultRepoPermission:   "read",   // Default fallback
		MembersCanCreateRepos:   true,     // Default fallback
		MembersCanCreatePrivate: true,     // Default fallback
		DefaultRepoVisibility:   "public", // Default fallback
		AllowGitHooks:           false,    // Default fallback
	}

	return settings, nil
}

func (c *giteaClient) UpdateOrganizationSettings(ctx context.Context, org string, req *UpdateOrganizationSettingsRequest) (*OrganizationSettings, error) {
	// Note: This is a conceptual implementation - real implementation would update
	// organization settings through appropriate Gitea admin endpoints
	path := fmt.Sprintf("/orgs/%s", org)

	// For now, we'll use the organization update endpoint as a placeholder
	orgUpdateReq := &UpdateOrganizationRequest{}

	resp, err := c.doRequest(ctx, "PATCH", path, orgUpdateReq)
	if err != nil {
		return nil, err
	}

	var orgData Organization
	if err := handleResponse(resp, &orgData); err != nil {
		return nil, err
	}

	// Return updated settings (simplified)
	return c.GetOrganizationSettings(ctx, org)
}

// Git Hooks API methods
func (c *giteaClient) GetGitHook(ctx context.Context, repository, hookType string) (*GitHook, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/hooks/git/%s", owner, repo, hookType)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Body: "git hook not found"}
	}

	var hook GitHook
	if err := handleResponse(resp, &hook); err != nil {
		return nil, err
	}

	return &hook, nil
}

func (c *giteaClient) CreateGitHook(ctx context.Context, repository string, req *CreateGitHookRequest) (*GitHook, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	// Gitea git hooks pre-exist (pre-receive/update/post-receive); there is no
	// POST to create one (405) — you EDIT the hook's script via PATCH.
	path := fmt.Sprintf("/repos/%s/%s/hooks/git/%s", owner, repo, req.HookType)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var hook GitHook
	if err := handleResponse(resp, &hook); err != nil {
		return nil, err
	}

	return &hook, nil
}

func (c *giteaClient) UpdateGitHook(ctx context.Context, repository, hookType string, req *UpdateGitHookRequest) (*GitHook, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/hooks/git/%s", owner, repo, hookType)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var hook GitHook
	if err := handleResponse(resp, &hook); err != nil {
		return nil, err
	}

	return &hook, nil
}

func (c *giteaClient) DeleteGitHook(ctx context.Context, repository, hookType string) error {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/hooks/git/%s", owner, repo, hookType)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Branch Protection API methods
func (c *giteaClient) GetBranchProtection(ctx context.Context, repository, branch string) (*BranchProtection, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/branch_protections/%s", owner, repo, branch)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Body: "branch protection not found"}
	}

	var protection BranchProtection
	if err := handleResponse(resp, &protection); err != nil {
		return nil, err
	}

	return &protection, nil
}

func (c *giteaClient) CreateBranchProtection(ctx context.Context, repository, branch string, req *CreateBranchProtectionRequest) (*BranchProtection, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/branch_protections", owner, repo)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var protection BranchProtection
	if err := handleResponse(resp, &protection); err != nil {
		return nil, err
	}

	return &protection, nil
}

func (c *giteaClient) UpdateBranchProtection(ctx context.Context, repository, branch string, req *UpdateBranchProtectionRequest) (*BranchProtection, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/branch_protections/%s", owner, repo, branch)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var protection BranchProtection
	if err := handleResponse(resp, &protection); err != nil {
		return nil, err
	}

	return &protection, nil
}

func (c *giteaClient) DeleteBranchProtection(ctx context.Context, repository, branch string) error {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/branch_protections/%s", owner, repo, branch)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Repository Key API methods
func (c *giteaClient) GetRepositoryKey(ctx context.Context, repository string, keyID int64) (*RepositoryKey, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/keys/%d", owner, repo, keyID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Body: "repository key not found"}
	}

	var key RepositoryKey
	if err := handleResponse(resp, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

func (c *giteaClient) CreateRepositoryKey(ctx context.Context, repository string, req *CreateRepositoryKeyRequest) (*RepositoryKey, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/keys", owner, repo)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var key RepositoryKey
	if err := handleResponse(resp, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

func (c *giteaClient) UpdateRepositoryKey(ctx context.Context, repository string, keyID int64, req *UpdateRepositoryKeyRequest) (*RepositoryKey, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/keys/%d", owner, repo, keyID)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var key RepositoryKey
	if err := handleResponse(resp, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

func (c *giteaClient) DeleteRepositoryKey(ctx context.Context, repository string, keyID int64) error {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/keys/%d", owner, repo, keyID)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Access Token API methods
func (c *giteaClient) GetAccessToken(ctx context.Context, username string, tokenID int64) (*AccessToken, error) {
	// Gitea has no GET for a single token (GET /users/{u}/tokens/{id} is 404);
	// list the user's tokens and match by id (same shape as the secret GETs).
	path := fmt.Sprintf("/users/%s/tokens", username)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var tokens []AccessToken
	if err := handleResponse(resp, &tokens); err != nil {
		return nil, err
	}
	for i := range tokens {
		if tokens[i].ID == tokenID {
			return &tokens[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Body: "access token not found"}
}

func (c *giteaClient) CreateAccessToken(ctx context.Context, username string, req *CreateAccessTokenRequest) (*AccessToken, error) {
	path := fmt.Sprintf("/users/%s/tokens", username)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var token AccessToken
	if err := handleResponse(resp, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

func (c *giteaClient) UpdateAccessToken(ctx context.Context, username string, tokenID int64, req *UpdateAccessTokenRequest) (*AccessToken, error) {
	path := fmt.Sprintf("/users/%s/tokens/%d", username, tokenID)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var token AccessToken
	if err := handleResponse(resp, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

func (c *giteaClient) DeleteAccessToken(ctx context.Context, username string, tokenID int64) error {
	path := fmt.Sprintf("/users/%s/tokens/%d", username, tokenID)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Repository Secret API methods
func (c *giteaClient) GetRepositorySecret(ctx context.Context, repository, secretName string) (*RepositorySecret, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	// Gitea has no GET single secret (405) and never returns secret VALUES.
	// List the repo's secrets and match by name (Gitea upper-cases secret names).
	path := fmt.Sprintf("/repos/%s/%s/actions/secrets", owner, repo)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var list []RepositorySecret
	if err := handleResponse(resp, &list); err != nil {
		return nil, err
	}
	for i := range list {
		if strings.EqualFold(list[i].Name, secretName) {
			return &list[i], nil
		}
	}
	return nil, NewNotFoundError("repository secret", secretName)
}

func (c *giteaClient) CreateRepositorySecret(ctx context.Context, repository, secretName string, req *CreateRepositorySecretRequest) error {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/actions/secrets/%s", owner, repo, secretName)
	resp, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

func (c *giteaClient) UpdateRepositorySecret(ctx context.Context, repository, secretName string, req *UpdateRepositorySecretRequest) error {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/actions/secrets/%s", owner, repo, secretName)
	resp, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

func (c *giteaClient) DeleteRepositorySecret(ctx context.Context, repository, secretName string) error {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/actions/secrets/%s", owner, repo, secretName)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// getTokenFromSecret extracts the API token from the provider config's secret
func getTokenFromSecret(ctx context.Context, cfg *v1beta1.ProviderConfig, kube client.Client) (string, error) {
	if cfg.Spec.Credentials.Source != "Secret" {
		return "", errors.New("only Secret credential source is supported")
	}

	if cfg.Spec.Credentials.SecretRef == nil {
		return "", errors.New("secretRef is required when using Secret credential source")
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Namespace: cfg.Spec.Credentials.SecretRef.Namespace,
		Name:      cfg.Spec.Credentials.SecretRef.Name,
	}

	if err := kube.Get(ctx, key, secret); err != nil {
		return "", errors.Wrap(err, "failed to get secret")
	}

	keyName := "token"
	if cfg.Spec.Credentials.SecretRef.Key != "" {
		keyName = cfg.Spec.Credentials.SecretRef.Key
	}

	token, ok := secret.Data[keyName]
	if !ok {
		return "", errors.Errorf("key %s not found in secret", keyName)
	}

	return string(token), nil
}

// doRequest performs an HTTP request with authentication
func (c *giteaClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal request body")
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	// Some Gitea endpoints (notably user access-token CRUD under
	// /users/{user}/tokens) reject token auth and REQUIRE HTTP basic auth as the
	// owning user. A client built with NewBasicAuthClient carries those creds;
	// the default client authenticates with the ProviderConfig token.
	if c.basicUser != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	} else {
		req.Header.Set("Authorization", "token "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform request")
	}

	return resp, nil
}

// APIError is a typed error carrying the HTTP status code of a failed Gitea API
// request. Controllers must classify not-found (and other) conditions off the
// status code via IsNotFound/StatusCode, never by string-matching the message
// (which is brittle: "404" matches "40456" and misses "(status 404)").
// See crossplane-provider-template dev/docs/09-lessons-learned.md #3.
type APIError struct {
	// StatusCode is the HTTP status returned by Gitea.
	StatusCode int
	// Body is the (possibly empty) response body, retained for diagnostics.
	Body string
}

// Error preserves the historical message format so existing string assertions
// and logs keep working; classification, however, should use StatusCode.
func (e *APIError) Error() string {
	return fmt.Sprintf("API request failed with status %d: %s", e.StatusCode, e.Body)
}

// StatusCode returns the HTTP status of the first APIError in err's chain, or 0
// if err is not (or does not wrap) an APIError.
func StatusCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

// handleResponse handles the HTTP response and unmarshals JSON if needed
func handleResponse(resp *http.Response, target interface{}) error {
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	if target != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return errors.Wrap(err, "failed to decode response")
		}
	}

	return nil
}

// IsNotFound reports whether err represents an HTTP 404, classified off the
// typed status code (works through pkg/errors Wrap chains).
func IsNotFound(err error) bool {
	return StatusCode(err) == http.StatusNotFound
}

// NewNotFoundError creates a new typed not-found (404) error.
func NewNotFoundError(resourceType, identifier string) error {
	return &APIError{
		StatusCode: http.StatusNotFound,
		Body:       fmt.Sprintf("%s '%s' not found", resourceType, identifier),
	}
}
