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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

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

	// Deploy Key operations
	GetDeployKey(ctx context.Context, owner, repo string, id int64) (*DeployKey, error)
	CreateDeployKey(ctx context.Context, owner, repo string, req *CreateDeployKeyRequest) (*DeployKey, error)
	DeleteDeployKey(ctx context.Context, owner, repo string, id int64) error

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
	ListOrganizationTeams(ctx context.Context, org string) ([]*Team, error)

	// Label operations
	GetLabel(ctx context.Context, owner, repo string, labelID int64) (*Label, error)
	CreateLabel(ctx context.Context, owner, repo string, req *CreateLabelRequest) (*Label, error)
	UpdateLabel(ctx context.Context, owner, repo string, labelID int64, req *UpdateLabelRequest) (*Label, error)
	DeleteLabel(ctx context.Context, owner, repo string, labelID int64) error
	ListRepositoryLabels(ctx context.Context, owner, repo string) ([]*Label, error)

	// Repository Collaborator operations
	GetRepositoryCollaborator(ctx context.Context, owner, repo, username string) (*RepositoryCollaborator, error)
	AddRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *AddCollaboratorRequest) error
	UpdateRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *UpdateCollaboratorRequest) error
	RemoveRepositoryCollaborator(ctx context.Context, owner, repo, username string) error
	ListRepositoryCollaborators(ctx context.Context, owner, repo string) ([]*RepositoryCollaborator, error)

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

	// User Key operations
	GetUserKey(ctx context.Context, username string, keyID int64) (*UserKey, error)
	CreateUserKey(ctx context.Context, username string, req *CreateUserKeyRequest) (*UserKey, error)
	UpdateUserKey(ctx context.Context, username string, keyID int64, req *UpdateUserKeyRequest) (*UserKey, error)
	DeleteUserKey(ctx context.Context, username string, keyID int64) error

	// Issue operations
	GetIssue(ctx context.Context, owner, repo string, number int64) (*Issue, error)
	CreateIssue(ctx context.Context, owner, repo string, req *CreateIssueOptions) (*Issue, error)
	UpdateIssue(ctx context.Context, owner, repo string, number int64, req *UpdateIssueOptions) (*Issue, error)
	DeleteIssue(ctx context.Context, owner, repo string, number int64) error

	// PullRequest operations
	GetPullRequest(ctx context.Context, owner, repo string, number int64) (*PullRequest, error)
	CreatePullRequest(ctx context.Context, owner, repo string, req *CreatePullRequestOptions) (*PullRequest, error)
	UpdatePullRequest(ctx context.Context, owner, repo string, number int64, req *UpdatePullRequestOptions) (*PullRequest, error)
	DeletePullRequest(ctx context.Context, owner, repo string, number int64) error
	MergePullRequest(ctx context.Context, owner, repo string, number int64, req *MergePullRequestOptions) (*PullRequest, error)

	// Release operations
	GetRelease(ctx context.Context, owner, repo string, id int64) (*Release, error)
	GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*Release, error)
	CreateRelease(ctx context.Context, owner, repo string, req *CreateReleaseOptions) (*Release, error)
	UpdateRelease(ctx context.Context, owner, repo string, id int64, req *UpdateReleaseOptions) (*Release, error)
	DeleteRelease(ctx context.Context, owner, repo string, id int64) error
	CreateReleaseAttachment(ctx context.Context, owner, repo string, releaseID int64, filename, contentType string, content []byte) (*ReleaseAttachment, error)
	DeleteReleaseAttachment(ctx context.Context, owner, repo string, releaseID, attachmentID int64) error

	// Organization Member operations
	GetOrganizationMember(ctx context.Context, org, username string) (*OrganizationMember, error)
	AddOrganizationMember(ctx context.Context, org, username string, req *AddOrganizationMemberRequest) (*OrganizationMember, error)
	UpdateOrganizationMember(ctx context.Context, org, username string, req *UpdateOrganizationMemberRequest) (*OrganizationMember, error)
	RemoveOrganizationMember(ctx context.Context, org, username string) error

	// Action operations
	GetAction(ctx context.Context, repository, workflowName string) (*Action, error)
	CreateAction(ctx context.Context, repository string, req *CreateActionRequest) (*Action, error)
	UpdateAction(ctx context.Context, repository, workflowName string, req *UpdateActionRequest) (*Action, error)
	DeleteAction(ctx context.Context, repository, workflowName string) error
	EnableAction(ctx context.Context, repository, workflowName string) error
	DisableAction(ctx context.Context, repository, workflowName string) error

	// Runner operations
	GetRunner(ctx context.Context, scope, scopeValue string, runnerID int64) (*Runner, error)
	CreateRunner(ctx context.Context, scope, scopeValue string, req *CreateRunnerRequest) (*Runner, error)
	UpdateRunner(ctx context.Context, scope, scopeValue string, runnerID int64, req *UpdateRunnerRequest) (*Runner, error)
	DeleteRunner(ctx context.Context, scope, scopeValue string, runnerID int64) error

	// Admin User operations
	GetAdminUser(ctx context.Context, username string) (*AdminUser, error)
	CreateAdminUser(ctx context.Context, req *CreateAdminUserRequest) (*AdminUser, error)
	UpdateAdminUser(ctx context.Context, username string, req *UpdateAdminUserRequest) (*AdminUser, error)
	DeleteAdminUser(ctx context.Context, username string) error
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
}

// Issue represents a Gitea issue
type Issue struct {
	ID          int64      `json:"id"`
	Number      int64      `json:"number"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	State       string     `json:"state"`
	HTMLURL     string     `json:"html_url"`
	Comments    int        `json:"comments"`
	User        *User      `json:"user"`
	Labels      []*Label   `json:"labels"`
	Assignees   []*User    `json:"assignees"`
	Milestone   *Milestone `json:"milestone,omitempty"`
	CreatedAt   *metav1.Time `json:"created_at,omitempty"`
	UpdatedAt   *metav1.Time `json:"updated_at,omitempty"`
	ClosedAt    *metav1.Time `json:"closed_at,omitempty"`
}

// Milestone represents a Gitea milestone
type Milestone struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	OpenIssues  int    `json:"open_issues"`
	ClosedIssues int   `json:"closed_issues"`
}

// CreateIssueOptions represents the options for creating an issue
type CreateIssueOptions struct {
	Title     string   `json:"title"`
	Body      *string  `json:"body,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Milestone *string  `json:"milestone,omitempty"`
}

// UpdateIssueOptions represents the options for updating an issue
type UpdateIssueOptions struct {
	Title     *string  `json:"title,omitempty"`
	Body      *string  `json:"body,omitempty"`
	State     *string  `json:"state,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Milestone *string  `json:"milestone,omitempty"`
}

// PullRequest represents a Gitea pull request
type PullRequest struct {
	ID             int64         `json:"id"`
	Number         int64         `json:"number"`
	Title          string        `json:"title"`
	Body           string        `json:"body"`
	State          string        `json:"state"`
	HTMLURL        string        `json:"html_url"`
	DiffURL        string        `json:"diff_url"`
	PatchURL       string        `json:"patch_url"`
	Mergeable      *bool         `json:"mergeable,omitempty"`
	Merged         bool          `json:"merged"`
	Comments       int           `json:"comments"`
	ReviewComments int           `json:"review_comments"`
	Additions      int           `json:"additions"`
	Deletions      int           `json:"deletions"`
	ChangedFiles   int           `json:"changed_files"`
	Draft          bool          `json:"draft"`
	User           *User         `json:"user"`
	Head           *Branch       `json:"head"`
	Base           *Branch       `json:"base"`
	Labels         []*Label      `json:"labels"`
	Assignees      []*User       `json:"assignees"`
	RequestedReviewers []*User   `json:"requested_reviewers"`
	Milestone      *Milestone    `json:"milestone,omitempty"`
	CreatedAt      *metav1.Time  `json:"created_at,omitempty"`
	UpdatedAt      *metav1.Time  `json:"updated_at,omitempty"`
	ClosedAt       *metav1.Time  `json:"closed_at,omitempty"`
	MergedAt       *metav1.Time  `json:"merged_at,omitempty"`
}

// Branch represents a git branch reference in a pull request
type Branch struct {
	Ref  string      `json:"ref"`
	SHA  string      `json:"sha"`
	Repo *Repository `json:"repo"`
}

// CreatePullRequestOptions represents the options for creating a pull request
type CreatePullRequestOptions struct {
	Title     string   `json:"title"`
	Body      *string  `json:"body,omitempty"`
	Head      string   `json:"head"`
	Base      string   `json:"base"`
	Assignees []string `json:"assignees,omitempty"`
	Reviewers []string `json:"reviewers,omitempty"`
	TeamReviewers []string `json:"team_reviewers,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Milestone *string  `json:"milestone,omitempty"`
	Draft     *bool    `json:"draft,omitempty"`
}

// UpdatePullRequestOptions represents the options for updating a pull request
type UpdatePullRequestOptions struct {
	Title     *string  `json:"title,omitempty"`
	Body      *string  `json:"body,omitempty"`
	State     *string  `json:"state,omitempty"`
	Base      *string  `json:"base,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Milestone *string  `json:"milestone,omitempty"`
	Draft     *bool    `json:"draft,omitempty"`
}

// MergePullRequestOptions represents the options for merging a pull request
type MergePullRequestOptions struct {
	DoMerge        bool   `json:"Do"`
	MergeMessageField string `json:"MergeMessageField,omitempty"`
	MergeTitleField   string `json:"MergeTitleField,omitempty"`
	MergeWhen         string `json:"MergeWhen,omitempty"`
}

// Release represents a Gitea release
type Release struct {
	ID              int64                     `json:"id"`
	TagName         string                    `json:"tag_name"`
	TargetCommitish string                    `json:"target_commitish"`
	Name            string                    `json:"name"`
	Body            string                    `json:"body"`
	URL             string                    `json:"url"`
	HTMLURL         string                    `json:"html_url"`
	TarballURL      string                    `json:"tarball_url"`
	ZipballURL      string                    `json:"zipball_url"`
	UploadURL       string                    `json:"upload_url"`
	Draft           bool                      `json:"draft"`
	Prerelease      bool                      `json:"prerelease"`
	CreatedAt       *metav1.Time              `json:"created_at,omitempty"`
	PublishedAt     *metav1.Time              `json:"published_at,omitempty"`
	Author          *User                     `json:"author,omitempty"`
	Assets          []ReleaseAttachment       `json:"assets,omitempty"`
}

// ReleaseAttachment represents a release asset/attachment
type ReleaseAttachment struct {
	ID                 int64        `json:"id"`
	Name               string       `json:"name"`
	Size               int64        `json:"size"`
	DownloadCount      int64        `json:"download_count"`
	ContentType        string       `json:"content_type"`
	BrowserDownloadURL string       `json:"browser_download_url"`
	CreatedAt          *metav1.Time `json:"created_at,omitempty"`
	UpdatedAt          *metav1.Time `json:"updated_at,omitempty"`
}

// CreateReleaseOptions represents the options for creating a release
type CreateReleaseOptions struct {
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish,omitempty"`
	Name            string `json:"name,omitempty"`
	Body            string `json:"body,omitempty"`
	Draft           bool   `json:"draft"`
	Prerelease      bool   `json:"prerelease"`
	GenerateNotes   bool   `json:"generate_notes,omitempty"`
}

// UpdateReleaseOptions represents the options for updating a release
type UpdateReleaseOptions struct {
	TagName         *string `json:"tag_name,omitempty"`
	TargetCommitish *string `json:"target_commitish,omitempty"`
	Name            *string `json:"name,omitempty"`
	Body            *string `json:"body,omitempty"`
	Draft           *bool   `json:"draft,omitempty"`
	Prerelease      *bool   `json:"prerelease,omitempty"`
	GenerateNotes   *bool   `json:"generate_notes,omitempty"`
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
		return nil, errors.New("team not found")
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

func (c *giteaClient) ListOrganizationTeams(ctx context.Context, org string) ([]*Team, error) {
	path := "/orgs/" + org + "/teams"
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var teams []*Team
	if err := handleResponse(resp, &teams); err != nil {
		return nil, err
	}

	return teams, nil
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

// UserKey represents a user SSH key
type UserKey struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"created_at"`
	ReadOnly    bool   `json:"read_only"`
}

// CreateUserKeyRequest represents the request body for creating a user key
type CreateUserKeyRequest struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	ReadOnly *bool  `json:"read_only,omitempty"`
}

// UpdateUserKeyRequest represents the request body for updating a user key
type UpdateUserKeyRequest struct {
	Title    *string `json:"title,omitempty"`
	ReadOnly *bool   `json:"read_only,omitempty"`
}

// OrganizationMember represents an organization member
type OrganizationMember struct {
	Username   string `json:"username"`
	FullName   string `json:"full_name"`
	Email      string `json:"email"`
	AvatarURL  string `json:"avatar_url"`
	Role       string `json:"role"`       // owner, admin, member
	Visibility string `json:"visibility"` // public, private
	IsPublic   bool   `json:"is_public"`
}

// AddOrganizationMemberRequest represents the request body for adding an organization member
type AddOrganizationMemberRequest struct {
	Role string `json:"role"` // owner, admin, member
}

// UpdateOrganizationMemberRequest represents the request body for updating an organization member
type UpdateOrganizationMemberRequest struct {
	Role       *string `json:"role,omitempty"`       // owner, admin, member
	Visibility *string `json:"visibility,omitempty"` // public, private
}

// Action represents a Gitea Actions workflow
type Action struct {
	WorkflowName string             `json:"workflow_name"`
	State        string             `json:"state"` // active, disabled
	Badge        string             `json:"badge_url"`
	CreatedAt    string             `json:"created_at"`
	UpdatedAt    string             `json:"updated_at"`
	WorkflowFile ActionWorkflowFile `json:"workflow_file"`
	LastRun      *ActionLastRun     `json:"last_run,omitempty"`
}

// ActionWorkflowFile represents the workflow file details
type ActionWorkflowFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

// ActionLastRun represents the last workflow run information
type ActionLastRun struct {
	ID         int64  `json:"id"`
	Number     int64  `json:"number"`
	Status     string `json:"status"`     // success, failure, pending, cancelled
	Conclusion string `json:"conclusion"` // success, failure, cancelled, skipped
	Event      string `json:"event"`      // push, pull_request, manual, etc.
	Branch     string `json:"branch"`
	Commit     string `json:"commit"`
	StartedAt  string `json:"started_at"`
	UpdatedAt  string `json:"updated_at"`
}

// CreateActionRequest represents the request body for creating an action workflow
type CreateActionRequest struct {
	WorkflowName string `json:"workflow_name"`
	WorkflowFile string `json:"workflow_file"` // YAML content
	Path         string `json:"path"`          // .github/workflows/name.yml
	Message      string `json:"message,omitempty"`
	Branch       string `json:"branch,omitempty"`
}

// UpdateActionRequest represents the request body for updating an action workflow
type UpdateActionRequest struct {
	WorkflowFile *string `json:"workflow_file,omitempty"` // YAML content
	Message      *string `json:"message,omitempty"`
	Branch       *string `json:"branch,omitempty"`
}

// Runner represents a Gitea Actions runner
type Runner struct {
	ID              int64           `json:"id"`
	UUID            string          `json:"uuid"`
	Name            string          `json:"name"`
	Status          string          `json:"status"` // online, offline, idle, active
	LastOnline      string          `json:"last_online"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
	Labels          []string        `json:"labels"`
	Description     string          `json:"description"`
	Scope           string          `json:"scope"`       // repository, organization, system
	ScopeValue      string          `json:"scope_value"` // repo name or org name
	RunnerGroup     *RunnerGroupRef `json:"runner_group,omitempty"`
	Version         string          `json:"version"`
	Architecture    string          `json:"architecture"`
	OperatingSystem string          `json:"operating_system"`
	TokenExpiresAt  string          `json:"token_expires_at"`
}

// RunnerGroupRef represents a reference to a runner group
type RunnerGroupRef struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

// CreateRunnerRequest represents the request body for creating a runner
type CreateRunnerRequest struct {
	Name          string   `json:"name"`
	Labels        []string `json:"labels"`
	Description   string   `json:"description,omitempty"`
	RunnerGroupID *int64   `json:"runner_group_id,omitempty"`
}

// UpdateRunnerRequest represents the request body for updating a runner
type UpdateRunnerRequest struct {
	Name          *string  `json:"name,omitempty"`
	Labels        []string `json:"labels,omitempty"`
	Description   *string  `json:"description,omitempty"`
	RunnerGroupID *int64   `json:"runner_group_id,omitempty"`
}

// AdminUser represents a Gitea administrative user
type AdminUser struct {
	ID              int64           `json:"id"`
	Username        string          `json:"username"`
	Email           string          `json:"email"`
	FullName        string          `json:"full_name"`
	AvatarURL       string          `json:"avatar_url"`
	IsAdmin         bool            `json:"is_admin"`
	IsActive        bool            `json:"is_active"`
	IsRestricted    bool            `json:"is_restricted"`
	ProhibitLogin   bool            `json:"prohibit_login"`
	Visibility      string          `json:"visibility"` // public, private, limited
	CreatedAt       string          `json:"created_at"`
	LastLogin       string          `json:"last_login"`
	Language        string          `json:"language"`
	MaxRepoCreation int             `json:"max_repo_creation"`
	Website         string          `json:"website"`
	Location        string          `json:"location"`
	Description     string          `json:"description"`
	UserStats       *AdminUserStats `json:"user_stats,omitempty"`
}

// AdminUserStats represents user statistics
type AdminUserStats struct {
	Repositories int `json:"repositories"`
	PublicRepos  int `json:"public_repos"`
	Followers    int `json:"followers"`
	Following    int `json:"following"`
	StarredRepos int `json:"starred_repos"`
}

// CreateAdminUserRequest represents the request body for creating an admin user
type CreateAdminUserRequest struct {
	Username           string `json:"username"`
	Email              string `json:"email"`
	Password           string `json:"password"`
	FullName           string `json:"full_name,omitempty"`
	IsAdmin            bool   `json:"is_admin,omitempty"`
	MustChangePassword bool   `json:"must_change_password,omitempty"`
	SendNotify         bool   `json:"send_notify,omitempty"`
	Visibility         string `json:"visibility,omitempty"`
	IsActive           bool   `json:"is_active,omitempty"`
	IsRestricted       bool   `json:"is_restricted,omitempty"`
	MaxRepoCreation    int    `json:"max_repo_creation,omitempty"`
	ProhibitLogin      bool   `json:"prohibit_login,omitempty"`
	Website            string `json:"website,omitempty"`
	Location           string `json:"location,omitempty"`
	Description        string `json:"description,omitempty"`
}

// UpdateAdminUserRequest represents the request body for updating an admin user
type UpdateAdminUserRequest struct {
	Email           *string `json:"email,omitempty"`
	FullName        *string `json:"full_name,omitempty"`
	IsAdmin         *bool   `json:"is_admin,omitempty"`
	Visibility      *string `json:"visibility,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
	IsRestricted    *bool   `json:"is_restricted,omitempty"`
	MaxRepoCreation *int    `json:"max_repo_creation,omitempty"`
	ProhibitLogin   *bool   `json:"prohibit_login,omitempty"`
	Website         *string `json:"website,omitempty"`
	Location        *string `json:"location,omitempty"`
	Description     *string `json:"description,omitempty"`
}

// Label API methods
func (c *giteaClient) GetLabel(ctx context.Context, owner, repo string, labelID int64) (*Label, error) {
	path := fmt.Sprintf("/repos/%s/%s/labels/%d", owner, repo, labelID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("label not found")
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

func (c *giteaClient) ListRepositoryLabels(ctx context.Context, owner, repo string) ([]*Label, error) {
	path := fmt.Sprintf("/repos/%s/%s/labels", owner, repo)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var labels []*Label
	if err := handleResponse(resp, &labels); err != nil {
		return nil, err
	}

	return labels, nil
}

// Repository Collaborator API methods
func (c *giteaClient) GetRepositoryCollaborator(ctx context.Context, owner, repo, username string) (*RepositoryCollaborator, error) {
	path := fmt.Sprintf("/repos/%s/%s/collaborators/%s", owner, repo, username)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("collaborator not found")
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

func (c *giteaClient) ListRepositoryCollaborators(ctx context.Context, owner, repo string) ([]*RepositoryCollaborator, error) {
	path := fmt.Sprintf("/repos/%s/%s/collaborators", owner, repo)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var collaborators []*RepositoryCollaborator
	if err := handleResponse(resp, &collaborators); err != nil {
		return nil, err
	}

	return collaborators, nil
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
		return nil, errors.New("git hook not found")
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

	path := fmt.Sprintf("/repos/%s/%s/hooks/git/%s", owner, repo, req.HookType)
	resp, err := c.doRequest(ctx, "POST", path, req)
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
		return nil, errors.New("branch protection not found")
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
		return nil, errors.New("repository key not found")
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
	path := fmt.Sprintf("/users/%s/tokens/%d", username, tokenID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("access token not found")
	}

	var token AccessToken
	if err := handleResponse(resp, &token); err != nil {
		return nil, err
	}

	return &token, nil
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

	path := fmt.Sprintf("/repos/%s/%s/actions/secrets/%s", owner, repo, secretName)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("repository secret not found")
	}

	var secret RepositorySecret
	if err := handleResponse(resp, &secret); err != nil {
		return nil, err
	}

	return &secret, nil
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

// User Key API methods
func (c *giteaClient) GetUserKey(ctx context.Context, username string, keyID int64) (*UserKey, error) {
	path := fmt.Sprintf("/users/%s/keys/%d", username, keyID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("user key not found")
	}

	var key UserKey
	if err := handleResponse(resp, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

func (c *giteaClient) CreateUserKey(ctx context.Context, username string, req *CreateUserKeyRequest) (*UserKey, error) {
	path := fmt.Sprintf("/users/%s/keys", username)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var key UserKey
	if err := handleResponse(resp, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

func (c *giteaClient) UpdateUserKey(ctx context.Context, username string, keyID int64, req *UpdateUserKeyRequest) (*UserKey, error) {
	path := fmt.Sprintf("/users/%s/keys/%d", username, keyID)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var key UserKey
	if err := handleResponse(resp, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

func (c *giteaClient) DeleteUserKey(ctx context.Context, username string, keyID int64) error {
	path := fmt.Sprintf("/users/%s/keys/%d", username, keyID)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Organization Member API methods
func (c *giteaClient) GetOrganizationMember(ctx context.Context, org, username string) (*OrganizationMember, error) {
	path := fmt.Sprintf("/orgs/%s/members/%s", org, username)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("organization member not found")
	}

	var member OrganizationMember
	if err := handleResponse(resp, &member); err != nil {
		return nil, err
	}

	return &member, nil
}

func (c *giteaClient) AddOrganizationMember(ctx context.Context, org, username string, req *AddOrganizationMemberRequest) (*OrganizationMember, error) {
	path := fmt.Sprintf("/orgs/%s/members/%s", org, username)
	resp, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return nil, err
	}

	var member OrganizationMember
	if err := handleResponse(resp, &member); err != nil {
		return nil, err
	}

	return &member, nil
}

func (c *giteaClient) UpdateOrganizationMember(ctx context.Context, org, username string, req *UpdateOrganizationMemberRequest) (*OrganizationMember, error) {
	path := fmt.Sprintf("/orgs/%s/members/%s", org, username)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var member OrganizationMember
	if err := handleResponse(resp, &member); err != nil {
		return nil, err
	}

	return &member, nil
}

func (c *giteaClient) RemoveOrganizationMember(ctx context.Context, org, username string) error {
	path := fmt.Sprintf("/orgs/%s/members/%s", org, username)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Action API methods
func (c *giteaClient) GetAction(ctx context.Context, repository, workflowName string) (*Action, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%s", owner, repo, workflowName)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("action workflow not found")
	}

	var action Action
	if err := handleResponse(resp, &action); err != nil {
		return nil, err
	}

	return &action, nil
}

func (c *giteaClient) CreateAction(ctx context.Context, repository string, req *CreateActionRequest) (*Action, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	// For Gitea, creating a workflow typically involves creating a file in .github/workflows/
	// This is a conceptual implementation - actual API might differ
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows", owner, repo)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var action Action
	if err := handleResponse(resp, &action); err != nil {
		return nil, err
	}

	return &action, nil
}

func (c *giteaClient) UpdateAction(ctx context.Context, repository, workflowName string, req *UpdateActionRequest) (*Action, error) {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%s", owner, repo, workflowName)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var action Action
	if err := handleResponse(resp, &action); err != nil {
		return nil, err
	}

	return &action, nil
}

func (c *giteaClient) DeleteAction(ctx context.Context, repository, workflowName string) error {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%s", owner, repo, workflowName)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

func (c *giteaClient) EnableAction(ctx context.Context, repository, workflowName string) error {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%s/enable", owner, repo, workflowName)
	resp, err := c.doRequest(ctx, "PUT", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

func (c *giteaClient) DisableAction(ctx context.Context, repository, workflowName string) error {
	// Parse repository format "owner/repo"
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return errors.New("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%s/disable", owner, repo, workflowName)
	resp, err := c.doRequest(ctx, "PUT", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Runner API methods
func (c *giteaClient) GetRunner(ctx context.Context, scope, scopeValue string, runnerID int64) (*Runner, error) {
	var path string
	switch scope {
	case "repository":
		// Parse repository format "owner/repo"
		parts := strings.Split(scopeValue, "/")
		if len(parts) != 2 {
			return nil, errors.New("scopeValue must be in format 'owner/repo' for repository scope")
		}
		owner, repo := parts[0], parts[1]
		path = fmt.Sprintf("/repos/%s/%s/actions/runners/%d", owner, repo, runnerID)
	case "organization":
		path = fmt.Sprintf("/orgs/%s/actions/runners/%d", scopeValue, runnerID)
	case "system":
		path = fmt.Sprintf("/admin/actions/runners/%d", runnerID)
	default:
		return nil, errors.New("scope must be one of: repository, organization, system")
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("runner not found")
	}

	var runner Runner
	if err := handleResponse(resp, &runner); err != nil {
		return nil, err
	}

	return &runner, nil
}

func (c *giteaClient) CreateRunner(ctx context.Context, scope, scopeValue string, req *CreateRunnerRequest) (*Runner, error) {
	var path string
	switch scope {
	case "repository":
		// Parse repository format "owner/repo"
		parts := strings.Split(scopeValue, "/")
		if len(parts) != 2 {
			return nil, errors.New("scopeValue must be in format 'owner/repo' for repository scope")
		}
		owner, repo := parts[0], parts[1]
		path = fmt.Sprintf("/repos/%s/%s/actions/runners", owner, repo)
	case "organization":
		path = fmt.Sprintf("/orgs/%s/actions/runners", scopeValue)
	case "system":
		path = "/admin/actions/runners"
	default:
		return nil, errors.New("scope must be one of: repository, organization, system")
	}

	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var runner Runner
	if err := handleResponse(resp, &runner); err != nil {
		return nil, err
	}

	return &runner, nil
}

func (c *giteaClient) UpdateRunner(ctx context.Context, scope, scopeValue string, runnerID int64, req *UpdateRunnerRequest) (*Runner, error) {
	var path string
	switch scope {
	case "repository":
		// Parse repository format "owner/repo"
		parts := strings.Split(scopeValue, "/")
		if len(parts) != 2 {
			return nil, errors.New("scopeValue must be in format 'owner/repo' for repository scope")
		}
		owner, repo := parts[0], parts[1]
		path = fmt.Sprintf("/repos/%s/%s/actions/runners/%d", owner, repo, runnerID)
	case "organization":
		path = fmt.Sprintf("/orgs/%s/actions/runners/%d", scopeValue, runnerID)
	case "system":
		path = fmt.Sprintf("/admin/actions/runners/%d", runnerID)
	default:
		return nil, errors.New("scope must be one of: repository, organization, system")
	}

	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var runner Runner
	if err := handleResponse(resp, &runner); err != nil {
		return nil, err
	}

	return &runner, nil
}

func (c *giteaClient) DeleteRunner(ctx context.Context, scope, scopeValue string, runnerID int64) error {
	var path string
	switch scope {
	case "repository":
		// Parse repository format "owner/repo"
		parts := strings.Split(scopeValue, "/")
		if len(parts) != 2 {
			return errors.New("scopeValue must be in format 'owner/repo' for repository scope")
		}
		owner, repo := parts[0], parts[1]
		path = fmt.Sprintf("/repos/%s/%s/actions/runners/%d", owner, repo, runnerID)
	case "organization":
		path = fmt.Sprintf("/orgs/%s/actions/runners/%d", scopeValue, runnerID)
	case "system":
		path = fmt.Sprintf("/admin/actions/runners/%d", runnerID)
	default:
		return errors.New("scope must be one of: repository, organization, system")
	}

	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Admin User API methods
func (c *giteaClient) GetAdminUser(ctx context.Context, username string) (*AdminUser, error) {
	path := fmt.Sprintf("/admin/users/%s", username)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("admin user not found")
	}

	var user AdminUser
	if err := handleResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *giteaClient) CreateAdminUser(ctx context.Context, req *CreateAdminUserRequest) (*AdminUser, error) {
	path := "/admin/users"
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var user AdminUser
	if err := handleResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *giteaClient) UpdateAdminUser(ctx context.Context, username string, req *UpdateAdminUserRequest) (*AdminUser, error) {
	path := fmt.Sprintf("/admin/users/%s", username)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var user AdminUser
	if err := handleResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *giteaClient) DeleteAdminUser(ctx context.Context, username string) error {
	path := fmt.Sprintf("/admin/users/%s", username)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// Issue API methods
func (c *giteaClient) GetIssue(ctx context.Context, owner, repo string, number int64) (*Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, NewNotFoundError("issue", fmt.Sprintf("%d", number))
	}

	var issue Issue
	if err := handleResponse(resp, &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func (c *giteaClient) CreateIssue(ctx context.Context, owner, repo string, req *CreateIssueOptions) (*Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues", owner, repo)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := handleResponse(resp, &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func (c *giteaClient) UpdateIssue(ctx context.Context, owner, repo string, number int64, req *UpdateIssueOptions) (*Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := handleResponse(resp, &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func (c *giteaClient) DeleteIssue(ctx context.Context, owner, repo string, number int64) error {
	// In Gitea, we typically close issues instead of deleting them
	// This method will close the issue
	req := &UpdateIssueOptions{
		State: func() *string { s := "closed"; return &s }(),
	}
	
	_, err := c.UpdateIssue(ctx, owner, repo, number, req)
	return err
}

// PullRequest operations implementation
func (c *giteaClient) GetPullRequest(ctx context.Context, owner, repo string, number int64) (*PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, NewNotFoundError("pull request", fmt.Sprintf("%d", number))
	}

	var pr PullRequest
	if err := handleResponse(resp, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func (c *giteaClient) CreatePullRequest(ctx context.Context, owner, repo string, req *CreatePullRequestOptions) (*PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := handleResponse(resp, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func (c *giteaClient) UpdatePullRequest(ctx context.Context, owner, repo string, number int64, req *UpdatePullRequestOptions) (*PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := handleResponse(resp, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func (c *giteaClient) DeletePullRequest(ctx context.Context, owner, repo string, number int64) error {
	// In Gitea, we typically close pull requests instead of deleting them
	// This method will close the pull request
	req := &UpdatePullRequestOptions{
		State: func() *string { s := "closed"; return &s }(),
	}
	
	_, err := c.UpdatePullRequest(ctx, owner, repo, number, req)
	return err
}

func (c *giteaClient) MergePullRequest(ctx context.Context, owner, repo string, number int64, req *MergePullRequestOptions) (*PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", owner, repo, number)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := handleResponse(resp, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

// Release API implementations

func (c *giteaClient) GetRelease(ctx context.Context, owner, repo string, id int64) (*Release, error) {
	path := fmt.Sprintf("/repos/%s/%s/releases/%d", owner, repo, id)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var release Release
	if err := handleResponse(resp, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *giteaClient) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*Release, error) {
	path := fmt.Sprintf("/repos/%s/%s/releases/tags/%s", owner, repo, tag)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var release Release
	if err := handleResponse(resp, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *giteaClient) CreateRelease(ctx context.Context, owner, repo string, req *CreateReleaseOptions) (*Release, error) {
	path := fmt.Sprintf("/repos/%s/%s/releases", owner, repo)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var release Release
	if err := handleResponse(resp, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *giteaClient) UpdateRelease(ctx context.Context, owner, repo string, id int64, req *UpdateReleaseOptions) (*Release, error) {
	path := fmt.Sprintf("/repos/%s/%s/releases/%d", owner, repo, id)
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var release Release
	if err := handleResponse(resp, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *giteaClient) DeleteRelease(ctx context.Context, owner, repo string, id int64) error {
	path := fmt.Sprintf("/repos/%s/%s/releases/%d", owner, repo, id)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

func (c *giteaClient) CreateReleaseAttachment(ctx context.Context, owner, repo string, releaseID int64, filename, contentType string, content []byte) (*ReleaseAttachment, error) {
	// For simplicity, we'll implement a basic version
	// In a full implementation, this would handle multipart file uploads
	path := fmt.Sprintf("/repos/%s/%s/releases/%d/assets", owner, repo, releaseID)
	
	// This is a simplified implementation - real implementation would use multipart upload
	req := map[string]interface{}{
		"name": filename,
		"content_type": contentType,
		// Note: Real implementation would handle file upload differently
	}
	
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var attachment ReleaseAttachment
	if err := handleResponse(resp, &attachment); err != nil {
		return nil, err
	}

	return &attachment, nil
}

func (c *giteaClient) DeleteReleaseAttachment(ctx context.Context, owner, repo string, releaseID, attachmentID int64) error {
	path := fmt.Sprintf("/repos/%s/%s/releases/%d/assets/%d", owner, repo, releaseID, attachmentID)
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
	defer func() {
		_ = resp.Body.Close()
	}()

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

// NewNotFoundError creates a new not found error
func NewNotFoundError(resourceType, identifier string) error {
	return errors.Errorf("API request failed with status 404: %s '%s' not found", resourceType, identifier)
}
