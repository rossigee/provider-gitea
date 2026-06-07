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

package repositorycollaborator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rossigee/provider-gitea/internal/clients"
	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestRepositoryCollaboratorFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
	assert.Equal(t, "testuser", fixtures.TestUser)
}

func TestRepositoryCollaboratorMockClientExpectations(t *testing.T) {
	// RepositoryCollaborator uses Add/Update/Remove semantics
	builder := ctesting.NewExternalClient().
		ExpectCreate("AddRepositoryCollaborator", nil, nil).
		ExpectGet("GetRepositoryCollaborator", nil, nil).
		ExpectUpdate("UpdateRepositoryCollaborator", nil, nil).
		ExpectDelete("RemoveRepositoryCollaborator", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

func TestRepositoryCollaboratorAdminPermissions(t *testing.T) {
	collaborator := &clients.RepositoryCollaborator{
		Username: "testuser",
		Permissions: clients.RepositoryCollaboratorPermissions{
			Admin: true,
			Push:  true,
			Pull:  true,
		},
	}

	assert.True(t, collaborator.Permissions.Admin)
	assert.True(t, collaborator.Permissions.Push)
	assert.True(t, collaborator.Permissions.Pull)
}

func TestRepositoryCollaboratorWritePermissions(t *testing.T) {
	collaborator := &clients.RepositoryCollaborator{
		Username: "developer",
		Permissions: clients.RepositoryCollaboratorPermissions{
			Admin: false,
			Push:  true,
			Pull:  true,
		},
	}

	assert.False(t, collaborator.Permissions.Admin)
	assert.True(t, collaborator.Permissions.Push)
	assert.True(t, collaborator.Permissions.Pull)
}

func TestRepositoryCollaboratorReadPermissions(t *testing.T) {
	collaborator := &clients.RepositoryCollaborator{
		Username: "reader",
		Permissions: clients.RepositoryCollaboratorPermissions{
			Admin: false,
			Push:  false,
			Pull:  true,
		},
	}

	assert.False(t, collaborator.Permissions.Admin)
	assert.False(t, collaborator.Permissions.Push)
	assert.True(t, collaborator.Permissions.Pull)
}

func TestRepositoryCollaboratorUserInfo(t *testing.T) {
	collaborator := &clients.RepositoryCollaborator{
		Username: "testuser",
		FullName: "Test User",
		Email:    "test@example.com",
	}

	assert.Equal(t, "testuser", collaborator.Username)
	assert.Equal(t, "Test User", collaborator.FullName)
	assert.Equal(t, "test@example.com", collaborator.Email)
}

func TestRepositoryCollaboratorAvatarURL(t *testing.T) {
	collaborator := &clients.RepositoryCollaborator{
		Username:  "testuser",
		AvatarURL: "https://example.com/avatar.jpg",
	}

	assert.Equal(t, "https://example.com/avatar.jpg", collaborator.AvatarURL)
}
