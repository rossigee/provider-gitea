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
	"net/http"
)

// ListOrganizationTeams lists every team belonging to org. It backs
// ResolveTeamID's name->id lookup for TeamMembership/TeamRepository.
func (c *giteaClient) ListOrganizationTeams(ctx context.Context, org string) ([]Team, error) {
	path := fmt.Sprintf("/orgs/%s/teams", org)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var teams []Team
	if err := handleResponse(resp, &teams); err != nil {
		return nil, err
	}

	return teams, nil
}

// ResolveTeamID finds a team's numeric id by (org, team name), reusing
// ListOrganizationTeams rather than a dedicated get-by-name endpoint. Returns a
// typed not-found error if the org has no team with that name.
func ResolveTeamID(ctx context.Context, c Client, org, team string) (int64, error) {
	teams, err := c.ListOrganizationTeams(ctx, org)
	if err != nil {
		return 0, err
	}
	for _, t := range teams {
		if t.Name == team {
			return t.ID, nil
		}
	}
	return 0, NewNotFoundError("team", org+"/"+team)
}

// GetTeamMember reports whether username is a member of teamID.
func (c *giteaClient) GetTeamMember(ctx context.Context, teamID int64, username string) (*User, error) {
	path := fmt.Sprintf("/teams/%d/members/%s", teamID, username)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Body: "team member not found"}
	}

	var member User
	if err := handleResponse(resp, &member); err != nil {
		return nil, err
	}

	return &member, nil
}

// AddTeamMember adds username to teamID. Idempotent — Gitea returns 204 even
// if the user is already a member.
func (c *giteaClient) AddTeamMember(ctx context.Context, teamID int64, username string) error {
	path := fmt.Sprintf("/teams/%d/members/%s", teamID, username)
	resp, err := c.doRequest(ctx, "PUT", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// RemoveTeamMember removes username from teamID.
func (c *giteaClient) RemoveTeamMember(ctx context.Context, teamID int64, username string) error {
	path := fmt.Sprintf("/teams/%d/members/%s", teamID, username)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// GetTeamRepository reports whether org/repo is attached to teamID.
func (c *giteaClient) GetTeamRepository(ctx context.Context, teamID int64, org, repo string) (*Repository, error) {
	path := fmt.Sprintf("/teams/%d/repos/%s/%s", teamID, org, repo)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Body: "team repository not found"}
	}

	var repository Repository
	if err := handleResponse(resp, &repository); err != nil {
		return nil, err
	}

	return &repository, nil
}

// AddTeamRepository attaches org/repo to teamID. Idempotent.
func (c *giteaClient) AddTeamRepository(ctx context.Context, teamID int64, org, repo string) error {
	path := fmt.Sprintf("/teams/%d/repos/%s/%s", teamID, org, repo)
	resp, err := c.doRequest(ctx, "PUT", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// RemoveTeamRepository detaches org/repo from teamID.
func (c *giteaClient) RemoveTeamRepository(ctx context.Context, teamID int64, org, repo string) error {
	path := fmt.Sprintf("/teams/%d/repos/%s/%s", teamID, org, repo)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}
