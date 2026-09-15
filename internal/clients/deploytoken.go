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
	"context"
	"fmt"
)

// DeployToken represents a Gitea repo-scoped HTTPS deploy token
type DeployToken struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	KeyType     string `json:"key_type"`
	Token       string `json:"token,omitempty"` // plaintext, only present on create
	URL         string `json:"url"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"created_at"`
	ReadOnly    bool   `json:"read_only"`
}

// CreateDeployTokenRequest represents the request body for creating a deploy token
type CreateDeployTokenRequest struct {
	Title    string `json:"title"`
	ReadOnly bool   `json:"read_only"`
}

// CreateDeployToken creates a new deploy token for a repository
func (c *giteaClient) CreateDeployToken(ctx context.Context, owner, repo string, req *CreateDeployTokenRequest) (*DeployToken, error) {
	path := fmt.Sprintf("/repos/%s/%s/keys/tokens", owner, repo)

	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var token DeployToken
	if err := handleResponse(resp, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// GetDeployToken retrieves a deploy token by repository and token ID.
// This hits the same endpoint as GetDeployKey since tokens and SSH keys share
// the same underlying endpoints; the response is decoded as DeployToken.
func (c *giteaClient) GetDeployToken(ctx context.Context, owner, repo string, id int64) (*DeployToken, error) {
	path := fmt.Sprintf("/repos/%s/%s/keys/%d", owner, repo, id)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var token DeployToken
	if err := handleResponse(resp, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// DeleteDeployToken deletes a deploy token.
// This hits the same endpoint as DeleteDeployKey since tokens and SSH keys share
// the same underlying delete endpoint.
func (c *giteaClient) DeleteDeployToken(ctx context.Context, owner, repo string, id int64) error {
	path := fmt.Sprintf("/repos/%s/%s/keys/%d", owner, repo, id)

	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}
