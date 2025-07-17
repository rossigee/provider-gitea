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

// GetUser retrieves a user by username
func (c *giteaClient) GetUser(ctx context.Context, username string) (*User, error) {
	path := fmt.Sprintf("/users/%s", username)
	
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := handleResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateUser creates a new user (admin only)
func (c *giteaClient) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
	path := "/admin/users"
	
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var user User
	if err := handleResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUser updates an existing user (admin only)
func (c *giteaClient) UpdateUser(ctx context.Context, username string, req *UpdateUserRequest) (*User, error) {
	path := fmt.Sprintf("/admin/users/%s", username)
	
	resp, err := c.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, err
	}

	var user User
	if err := handleResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// DeleteUser deletes a user (admin only)
func (c *giteaClient) DeleteUser(ctx context.Context, username string) error {
	path := fmt.Sprintf("/admin/users/%s", username)
	
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}