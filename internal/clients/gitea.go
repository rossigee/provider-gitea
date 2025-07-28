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
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
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

	// Deploy Key operations
	GetDeployKey(ctx context.Context, owner, repo string, id int64) (*DeployKey, error)
	CreateDeployKey(ctx context.Context, owner, repo string, req *CreateDeployKeyRequest) (*DeployKey, error)
	DeleteDeployKey(ctx context.Context, owner, repo string, id int64) error

	// Organization Secret operations
	GetOrganizationSecret(ctx context.Context, org, secretName string) (*OrganizationSecret, error)
	CreateOrganizationSecret(ctx context.Context, org, secretName string, req *CreateOrganizationSecretRequest) error
	UpdateOrganizationSecret(ctx context.Context, org, secretName string, req *CreateOrganizationSecretRequest) error
	DeleteOrganizationSecret(ctx context.Context, org, secretName string) error
}

// giteaClient implements the Client interface
type giteaClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
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
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Private      bool   `json:"private,omitempty"`
	AutoInit     bool   `json:"auto_init,omitempty"`
	Template     bool   `json:"template,omitempty"`
	Gitignores   string `json:"gitignores,omitempty"`
	License      string `json:"license,omitempty"`
	Readme       string `json:"readme,omitempty"`
	IssueLabels  string `json:"issue_labels,omitempty"`
	TrustModel   string `json:"trust_model,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

// UpdateRepositoryRequest represents the request body for updating a repository
type UpdateRepositoryRequest struct {
	Name                      *string `json:"name,omitempty"`
	Description               *string `json:"description,omitempty"`
	Website                   *string `json:"website,omitempty"`
	Private                   *bool   `json:"private,omitempty"`
	Template                  *bool   `json:"template,omitempty"`
	HasIssues                 *bool   `json:"has_issues,omitempty"`
	HasWiki                   *bool   `json:"has_wiki,omitempty"`
	HasPullRequests           *bool   `json:"has_pull_requests,omitempty"`
	HasProjects               *bool   `json:"has_projects,omitempty"`
	HasReleases               *bool   `json:"has_releases,omitempty"`
	HasPackages               *bool   `json:"has_packages,omitempty"`
	HasActions                *bool   `json:"has_actions,omitempty"`
	AllowMergeCommits         *bool   `json:"allow_merge_commits,omitempty"`
	AllowRebase               *bool   `json:"allow_rebase,omitempty"`
	AllowRebaseExplicit       *bool   `json:"allow_rebase_explicit,omitempty"`
	AllowSquashMerge          *bool   `json:"allow_squash_merge,omitempty"`
	AllowRebaseUpdate         *bool   `json:"allow_rebase_update,omitempty"`
	DefaultDeleteBranchAfterMerge *bool   `json:"default_delete_branch_after_merge,omitempty"`
	DefaultMergeStyle         *string `json:"default_merge_style,omitempty"`
	DefaultBranch             *string `json:"default_branch,omitempty"`
	Archived                  *bool   `json:"archived,omitempty"`
}

// Organization represents a Gitea organization
type Organization struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Website     string `json:"website"`
	Location    string `json:"location"`
	Visibility  string `json:"visibility"`
	RepoAdminChangeTeamAccess bool   `json:"repo_admin_change_team_access"`
	Email       string `json:"email"`
	AvatarURL   string `json:"avatar_url"`
}

// CreateOrganizationRequest represents the request body for creating an organization
type CreateOrganizationRequest struct {
	Username    string `json:"username"`
	Name        string `json:"name,omitempty"`
	FullName    string `json:"full_name,omitempty"`
	Description string `json:"description,omitempty"`
	Website     string `json:"website,omitempty"`
	Location    string `json:"location,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
	RepoAdminChangeTeamAccess bool   `json:"repo_admin_change_team_access,omitempty"`
}

// UpdateOrganizationRequest represents the request body for updating an organization
type UpdateOrganizationRequest struct {
	Name        *string `json:"name,omitempty"`
	FullName    *string `json:"full_name,omitempty"`
	Description *string `json:"description,omitempty"`
	Website     *string `json:"website,omitempty"`
	Location    *string `json:"location,omitempty"`
	Visibility  *string `json:"visibility,omitempty"`
	RepoAdminChangeTeamAccess *bool   `json:"repo_admin_change_team_access,omitempty"`
}

// User represents a Gitea user
type User struct {
	ID                int64  `json:"id"`
	Username          string `json:"username"`
	Name              string `json:"name"`
	FullName          string `json:"full_name"`
	Email             string `json:"email"`
	AvatarURL         string `json:"avatar_url"`
	Website           string `json:"website"`
	Location          string `json:"location"`
	IsAdmin           bool   `json:"is_admin"`
	LastLogin         string `json:"last_login"`
	Created           string `json:"created"`
	Restricted        bool   `json:"restricted"`
	Active            bool   `json:"active"`
	ProhibitLogin     bool   `json:"prohibit_login"`
	LoginName         string `json:"login_name"`
	SourceID          int64  `json:"source_id"`
	Language          string `json:"language"`
	Description       string `json:"description"`
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Username          string `json:"username"`
	Email             string `json:"email"`
	FullName          string `json:"full_name,omitempty"`
	Password          string `json:"password"`
	LoginName         string `json:"login_name,omitempty"`
	SendNotify        bool   `json:"send_notify,omitempty"`
	SourceID          int64  `json:"source_id,omitempty"`
	MustChangePassword bool   `json:"must_change_password,omitempty"`
	Restricted        bool   `json:"restricted,omitempty"`
	Visibility        string `json:"visibility,omitempty"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Email             *string `json:"email,omitempty"`
	FullName          *string `json:"full_name,omitempty"`
	LoginName         *string `json:"login_name,omitempty"`
	SourceID          *int64  `json:"source_id,omitempty"`
	Active            *bool   `json:"active,omitempty"`
	Admin             *bool   `json:"admin,omitempty"`
	AllowGitHook      *bool   `json:"allow_git_hook,omitempty"`
	AllowImportLocal  *bool   `json:"allow_import_local,omitempty"`
	AllowCreateOrganization *bool   `json:"allow_create_organization,omitempty"`
	ProhibitLogin     *bool   `json:"prohibit_login,omitempty"`
	Restricted        *bool   `json:"restricted,omitempty"`
	Website           *string `json:"website,omitempty"`
	Location          *string `json:"location,omitempty"`
	Description       *string `json:"description,omitempty"`
	Visibility        *string `json:"visibility,omitempty"`
}

// Webhook represents a Gitea webhook
type Webhook struct {
	ID     int64             `json:"id"`
	Type   string            `json:"type"`
	URL    string            `json:"config.url"`
	Active bool              `json:"active"`
	Events []string          `json:"events"`
	Config map[string]string `json:"config"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
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
	path := "/orgs/" + org + "/actions/secrets/" + secretName
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("organization secret not found")
	}

	var secret OrganizationSecret
	if err := handleResponse(resp, &secret); err != nil {
		return nil, err
	}

	return &secret, nil
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

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform request")
	}

	return resp, nil
}

// handleResponse handles the HTTP response and unmarshals JSON if needed
func handleResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return errors.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if target != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return errors.Wrap(err, "failed to decode response")
		}
	}

	return nil
}

// IsNotFound checks if an error represents a "not found" response
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	// Check if the error message contains status 404
	return strings.Contains(err.Error(), "status 404")
}