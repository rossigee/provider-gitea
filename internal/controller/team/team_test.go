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

package team_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
	"github.com/rossigee/provider-gitea/internal/clients"
)

func TestTeamTestFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.NotEmpty(t, fixtures.TestNamespace)
}

func TestTeamMockClientExpectations(t *testing.T) {
	teamResponse := &clients.Team{
		ID:          101,
		Name:        "Development",
		Description: "Development team",
		Permission:  "write",
	}
	teamResponse.Organization.ID = 123
	teamResponse.Organization.Username = "testorg"

	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateTeam", teamResponse, nil).
		ExpectGet("GetTeam", teamResponse, nil).
		ExpectUpdate("UpdateTeam", teamResponse, nil).
		ExpectDelete("DeleteTeam", nil)

	assert.NotNil(t, builder.GetGiteaClient())
}

func TestTeamIDHandling(t *testing.T) {
	teamResponse := &clients.Team{
		ID:   999,
		Name: "Test Team",
	}
	teamResponse.Organization.ID = 456

	builder := ctesting.NewExternalClient().
		ExpectGet("GetTeam", teamResponse, nil)

	assert.NotNil(t, builder.GetGiteaClient())
}
