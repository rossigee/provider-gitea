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

// GetRepositoryWebhook retrieves a webhook by repository and webhook ID
func (c *giteaClient) GetRepositoryWebhook(ctx context.Context, owner, repo string, id int64) (*Webhook, error) {
	path := fmt.Sprintf("/repos/%s/%s/hooks/%d", owner, repo, id)
	
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := handleResponse(resp, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// CreateRepositoryWebhook creates a new webhook for a repository
func (c *giteaClient) CreateRepositoryWebhook(ctx context.Context, owner, repo string, req *CreateWebhookRequest) (*Webhook, error) {
	path := fmt.Sprintf("/repos/%s/%s/hooks", owner, repo)
	
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := handleResponse(resp, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// UpdateRepositoryWebhook updates an existing webhook
func (c *giteaClient) UpdateRepositoryWebhook(ctx context.Context, owner, repo string, id int64, req *UpdateWebhookRequest) (*Webhook, error) {
	path := fmt.Sprintf("/repos/%s/%s/hooks/%d", owner, repo, id)
	
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := handleResponse(resp, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// DeleteRepositoryWebhook deletes a webhook
func (c *giteaClient) DeleteRepositoryWebhook(ctx context.Context, owner, repo string, id int64) error {
	path := fmt.Sprintf("/repos/%s/%s/hooks/%d", owner, repo, id)
	
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// GetOrganizationWebhook retrieves a webhook by organization and webhook ID
func (c *giteaClient) GetOrganizationWebhook(ctx context.Context, org string, id int64) (*Webhook, error) {
	path := fmt.Sprintf("/orgs/%s/hooks/%d", org, id)
	
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := handleResponse(resp, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// CreateOrganizationWebhook creates a new webhook for an organization
func (c *giteaClient) CreateOrganizationWebhook(ctx context.Context, org string, req *CreateWebhookRequest) (*Webhook, error) {
	path := fmt.Sprintf("/orgs/%s/hooks", org)
	
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := handleResponse(resp, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// UpdateOrganizationWebhook updates an existing organization webhook
func (c *giteaClient) UpdateOrganizationWebhook(ctx context.Context, org string, id int64, req *UpdateWebhookRequest) (*Webhook, error) {
	path := fmt.Sprintf("/orgs/%s/hooks/%d", org, id)
	
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := handleResponse(resp, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// DeleteOrganizationWebhook deletes an organization webhook
func (c *giteaClient) DeleteOrganizationWebhook(ctx context.Context, org string, id int64) error {
	path := fmt.Sprintf("/orgs/%s/hooks/%d", org, id)
	
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}