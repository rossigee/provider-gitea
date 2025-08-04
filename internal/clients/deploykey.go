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

// DeployKey represents a Gitea deploy key
type DeployKey struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Key         string `json:"key"`
	URL         string `json:"url"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"created_at"`
	ReadOnly    bool   `json:"read_only"`
}

// CreateDeployKeyRequest represents the request body for creating a deploy key
type CreateDeployKeyRequest struct {
	Title    string `json:"title"`
	Key      string `json:"key"`
	ReadOnly bool   `json:"read_only"`
}

// GetDeployKey retrieves a deploy key by repository and key ID
func (c *giteaClient) GetDeployKey(ctx context.Context, owner, repo string, id int64) (*DeployKey, error) {
	path := fmt.Sprintf("/repos/%s/%s/keys/%d", owner, repo, id)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var deployKey DeployKey
	if err := handleResponse(resp, &deployKey); err != nil {
		return nil, err
	}

	return &deployKey, nil
}

// CreateDeployKey creates a new deploy key for a repository
func (c *giteaClient) CreateDeployKey(ctx context.Context, owner, repo string, req *CreateDeployKeyRequest) (*DeployKey, error) {
	path := fmt.Sprintf("/repos/%s/%s/keys", owner, repo)

	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var deployKey DeployKey
	if err := handleResponse(resp, &deployKey); err != nil {
		return nil, err
	}

	return &deployKey, nil
}

// DeleteDeployKey deletes a deploy key
func (c *giteaClient) DeleteDeployKey(ctx context.Context, owner, repo string, id int64) error {
	path := fmt.Sprintf("/repos/%s/%s/keys/%d", owner, repo, id)

	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}
