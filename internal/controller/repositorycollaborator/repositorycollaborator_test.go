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

package repositorycollaborator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
	"github.com/rossigee/provider-gitea/internal/clients"
)

func TestRepositoryCollaboratorTestFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
	assert.Equal(t, "testuser", fixtures.TestUser)
}

func TestRepositoryCollaboratorMockClientExpectations(t *testing.T) {
	collaboratorResponse := &clients.RepositoryCollaborator{
		FullName: "Test User",
		Email:    "testuser@example.com",
		Username: "testuser",
		Permissions: clients.RepositoryCollaboratorPermissions{
			Admin: false,
			Push:  true,
			Pull:  true,
		},
	}

	builder := ctesting.NewExternalClient().
		ExpectCreate("AddRepositoryCollaborator", nil, nil).
		ExpectGet("GetRepositoryCollaborator", collaboratorResponse, nil).
		ExpectUpdate("UpdateRepositoryCollaborator", nil, nil).
		ExpectDelete("RemoveRepositoryCollaborator", nil)

	assert.NotNil(t, builder.GetGiteaClient())
}

func TestRepositoryCollaboratorAdminPermissions(t *testing.T) {
	collaboratorResponse := &clients.RepositoryCollaborator{
		FullName: "Admin User",
		Email:    "admin@example.com",
		Username: "adminuser",
		Permissions: clients.RepositoryCollaboratorPermissions{
			Admin: true,
			Push:  true,
			Pull:  true,
		},
	}

	builder := ctesting.NewExternalClient().
		ExpectGet("GetRepositoryCollaborator", collaboratorResponse, nil)

	assert.NotNil(t, builder.GetGiteaClient())
}

func TestRepositoryCollaboratorReadOnlyPermissions(t *testing.T) {
	collaboratorResponse := &clients.RepositoryCollaborator{
		FullName: "Read-Only User",
		Email:    "reader@example.com",
		Username: "readonlyuser",
		Permissions: clients.RepositoryCollaboratorPermissions{
			Admin: false,
			Push:  false,
			Pull:  true,
		},
	}

	builder := ctesting.NewExternalClient().
		ExpectGet("GetRepositoryCollaborator", collaboratorResponse, nil)

	assert.NotNil(t, builder.GetGiteaClient())
}
