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

// GetRepositoryTopics retrieves the topics assigned to a repository
func (c *giteaClient) GetRepositoryTopics(ctx context.Context, owner, name string) (*RepositoryTopics, error) {
	path := fmt.Sprintf("/repos/%s/%s/topics", owner, name)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var topics RepositoryTopics
	if err := handleResponse(resp, &topics); err != nil {
		return nil, err
	}

	return &topics, nil
}

// UpdateRepositoryTopics replaces the full set of topics on a repository.
// Gitea's API only supports full-replace for topics; there is no partial-update verb.
func (c *giteaClient) UpdateRepositoryTopics(ctx context.Context, owner, name string, req *UpdateRepositoryTopicsRequest) error {
	path := fmt.Sprintf("/repos/%s/%s/topics", owner, name)

	resp, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}
