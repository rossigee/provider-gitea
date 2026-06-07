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

package team

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rossigee/provider-gitea/internal/clients"
	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestTeamFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.NotEmpty(t, fixtures.TestNamespace)
}

func TestTeamMockClientExpectations(t *testing.T) {
	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateTeam", nil, nil).
		ExpectGet("GetTeam", nil, nil).
		ExpectUpdate("UpdateTeam", nil, nil).
		ExpectDelete("DeleteTeam", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

func TestTeamPermissions(t *testing.T) {
	team := &clients.Team{
		Permission: "write",
	}
	assert.Equal(t, "write", team.Permission)
}

func TestTeamPermissionAdmin(t *testing.T) {
	team := &clients.Team{
		Permission: "admin",
	}
	assert.Equal(t, "admin", team.Permission)
}

func TestTeamPermissionRead(t *testing.T) {
	team := &clients.Team{
		Permission: "read",
	}
	assert.Equal(t, "read", team.Permission)
}

func TestTeamOrganization(t *testing.T) {
	team := &clients.Team{
		ID:   101,
		Name: "Development",
		Organization: struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Name     string `json:"name"`
		}{
			ID:       123,
			Username: "testorg",
			Name:     "Test Organization",
		},
	}

	assert.Equal(t, int64(123), team.Organization.ID)
	assert.Equal(t, "testorg", team.Organization.Username)
}
