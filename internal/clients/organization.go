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

// GetOrganization retrieves an organization by name
func (c *giteaClient) GetOrganization(ctx context.Context, name string) (*Organization, error) {
	path := fmt.Sprintf("/orgs/%s", name)
	
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var organization Organization
	if err := handleResponse(resp, &organization); err != nil {
		return nil, err
	}

	return &organization, nil
}

// CreateOrganization creates a new organization
func (c *giteaClient) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest) (*Organization, error) {
	path := "/orgs"
	
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var organization Organization
	if err := handleResponse(resp, &organization); err != nil {
		return nil, err
	}

	return &organization, nil
}

// UpdateOrganization updates an existing organization
func (c *giteaClient) UpdateOrganization(ctx context.Context, name string, req *UpdateOrganizationRequest) (*Organization, error) {
	path := fmt.Sprintf("/orgs/%s", name)
	
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var organization Organization
	if err := handleResponse(resp, &organization); err != nil {
		return nil, err
	}

	return &organization, nil
}

// DeleteOrganization deletes an organization
func (c *giteaClient) DeleteOrganization(ctx context.Context, name string) error {
	path := fmt.Sprintf("/orgs/%s", name)
	
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}