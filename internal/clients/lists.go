/*
Copyright 2026 The Crossplane Authors.

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
	"context"
	"fmt"
)

// This file collects paginated list calls backing ExternalLister discovery.
// Page is 1-based; limit caps results per page (Gitea allows at most 50).

func normalizeListRange(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}

	if limit < 1 || limit > 50 {
		limit = 50
	}

	return page, limit
}

// ListIssues lists issues of a repository, all states.
func (c *giteaClient) ListIssues(ctx context.Context, owner, repo string, page, limit int) ([]Issue, error) {
	page, limit = normalizeListRange(page, limit)
	path := fmt.Sprintf("/repos/%s/%s/issues?state=all&page=%d&limit=%d", owner, repo, page, limit)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := handleResponse(resp, &issues); err != nil {
		return nil, err
	}

	return issues, nil
}

// ListPullRequests lists pull requests of a repository, all states.
func (c *giteaClient) ListPullRequests(ctx context.Context, owner, repo string, page, limit int) ([]PullRequest, error) {
	page, limit = normalizeListRange(page, limit)
	path := fmt.Sprintf("/repos/%s/%s/pulls?state=all&page=%d&limit=%d", owner, repo, page, limit)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var pulls []PullRequest
	if err := handleResponse(resp, &pulls); err != nil {
		return nil, err
	}

	return pulls, nil
}

// ListReleases lists releases of a repository.
func (c *giteaClient) ListReleases(ctx context.Context, owner, repo string, page, limit int) ([]Release, error) {
	page, limit = normalizeListRange(page, limit)
	path := fmt.Sprintf("/repos/%s/%s/releases?page=%d&limit=%d", owner, repo, page, limit)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var releases []Release
	if err := handleResponse(resp, &releases); err != nil {
		return nil, err
	}

	return releases, nil
}

// ListDeployKeys lists deploy keys of a repository.
func (c *giteaClient) ListDeployKeys(ctx context.Context, owner, repo string) ([]DeployKey, error) {
	path := fmt.Sprintf("/repos/%s/%s/keys", owner, repo)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var keys []DeployKey
	if err := handleResponse(resp, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// ListBranchProtections lists branch protections of a repository.
func (c *giteaClient) ListBranchProtections(ctx context.Context, owner, repo string) ([]BranchProtection, error) {
	path := fmt.Sprintf("/repos/%s/%s/branch_protections", owner, repo)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var protections []BranchProtection
	if err := handleResponse(resp, &protections); err != nil {
		return nil, err
	}

	return protections, nil
}

// ListRepositorySecrets lists action secrets of a repository.
func (c *giteaClient) ListRepositorySecrets(ctx context.Context, owner, repo string) ([]RepositorySecret, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/secrets", owner, repo)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var secrets []RepositorySecret
	if err := handleResponse(resp, &secrets); err != nil {
		return nil, err
	}

	return secrets, nil
}

// ListActionWorkflows lists action workflows of a repository.
func (c *giteaClient) ListActionWorkflows(ctx context.Context, owner, repo string) ([]Action, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows", owner, repo)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var workflows []Action
	if err := handleResponse(resp, &workflows); err != nil {
		return nil, err
	}

	return workflows, nil
}

// ListRepositoryWebhooks lists webhooks of a repository.
func (c *giteaClient) ListRepositoryWebhooks(ctx context.Context, owner, repo string) ([]Webhook, error) {
	path := fmt.Sprintf("/repos/%s/%s/hooks", owner, repo)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var hooks []Webhook
	if err := handleResponse(resp, &hooks); err != nil {
		return nil, err
	}

	return hooks, nil
}

// ListOrganizationWebhooks lists webhooks of an organization.
func (c *giteaClient) ListOrganizationWebhooks(ctx context.Context, org string) ([]Webhook, error) {
	path := fmt.Sprintf("/orgs/%s/hooks", org)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var hooks []Webhook
	if err := handleResponse(resp, &hooks); err != nil {
		return nil, err
	}

	return hooks, nil
}

// RunnerList is the paged wrapper Gitea uses for runner listings.
type RunnerList struct {
	Runners []Runner `json:"runners"`
}

func listRunners(ctx context.Context, c *giteaClient, path string) ([]Runner, error) {
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var list RunnerList
	if err := handleResponse(resp, &list); err != nil {
		return nil, err
	}

	return list.Runners, nil
}

// ListRepositoryRunners lists runners of a repository.
func (c *giteaClient) ListRepositoryRunners(ctx context.Context, owner, repo string) ([]Runner, error) {
	return listRunners(ctx, c, fmt.Sprintf("/repos/%s/%s/actions/runners", owner, repo))
}

// ListOrganizationRunners lists runners of an organization.
func (c *giteaClient) ListOrganizationRunners(ctx context.Context, org string) ([]Runner, error) {
	return listRunners(ctx, c, fmt.Sprintf("/orgs/%s/actions/runners", org))
}

// ListSystemRunners lists instance-wide runners (admin only).
func (c *giteaClient) ListSystemRunners(ctx context.Context) ([]Runner, error) {
	return listRunners(ctx, c, "/admin/actions/runners")
}

// ListOrganizationMembers lists members of an organization.
func (c *giteaClient) ListOrganizationMembers(ctx context.Context, org string) ([]OrganizationMember, error) {
	path := fmt.Sprintf("/orgs/%s/members", org)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var members []OrganizationMember
	if err := handleResponse(resp, &members); err != nil {
		return nil, err
	}

	return members, nil
}

// ListOrganizationSecrets lists action secrets of an organization.
func (c *giteaClient) ListOrganizationSecrets(ctx context.Context, org string) ([]OrganizationSecret, error) {
	path := fmt.Sprintf("/orgs/%s/actions/secrets", org)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var secrets []OrganizationSecret
	if err := handleResponse(resp, &secrets); err != nil {
		return nil, err
	}

	return secrets, nil
}

// ListUserTokens lists access tokens of a user (admin only).
func (c *giteaClient) ListUserTokens(ctx context.Context, username string) ([]AccessToken, error) {
	path := fmt.Sprintf("/users/%s/tokens", username)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var tokens []AccessToken
	if err := handleResponse(resp, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

// ListUserKeys lists public keys of a user.
func (c *giteaClient) ListUserKeys(ctx context.Context, username string) ([]UserKey, error) {
	path := fmt.Sprintf("/users/%s/keys", username)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var keys []UserKey
	if err := handleResponse(resp, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}
