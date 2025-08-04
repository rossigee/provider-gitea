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

// GetRepository retrieves a repository by owner and name
func (c *giteaClient) GetRepository(ctx context.Context, owner, name string) (*Repository, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, name)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var repository Repository
	if err := handleResponse(resp, &repository); err != nil {
		return nil, err
	}

	return &repository, nil
}

// CreateRepository creates a new repository
func (c *giteaClient) CreateRepository(ctx context.Context, req *CreateRepositoryRequest) (*Repository, error) {
	path := "/user/repos"

	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var repository Repository
	if err := handleResponse(resp, &repository); err != nil {
		return nil, err
	}

	return &repository, nil
}

// CreateOrganizationRepository creates a new repository in an organization
func (c *giteaClient) CreateOrganizationRepository(ctx context.Context, org string, req *CreateRepositoryRequest) (*Repository, error) {
	path := fmt.Sprintf("/orgs/%s/repos", org)

	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var repository Repository
	if err := handleResponse(resp, &repository); err != nil {
		return nil, err
	}

	return &repository, nil
}

// UpdateRepository updates an existing repository
func (c *giteaClient) UpdateRepository(ctx context.Context, owner, name string, req *UpdateRepositoryRequest) (*Repository, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, name)

	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var repository Repository
	if err := handleResponse(resp, &repository); err != nil {
		return nil, err
	}

	return &repository, nil
}

// DeleteRepository deletes a repository
func (c *giteaClient) DeleteRepository(ctx context.Context, owner, name string) error {
	path := fmt.Sprintf("/repos/%s/%s", owner, name)

	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}
