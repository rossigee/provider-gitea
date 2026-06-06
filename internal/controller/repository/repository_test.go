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

package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"

	repositoryv2 "github.com/rossigee/provider-gitea/apis/repository/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

// Smoke tests for Repository controller

func TestRepositoryFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testuser", fixtures.TestUser)
	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
}

func TestRepositoryResponseBuilder(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()
	repo := fixtures.RepositoryResponse()

	assert.Equal(t, int64(123), repo.ID)
	assert.Equal(t, "testrepo", repo.Name)
	assert.Equal(t, "testorg/testrepo", repo.FullName)
	assert.True(t, repo.Private)
}

func TestRepositoryMockClientSetup(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()
	repo := fixtures.RepositoryResponse()

	ext := &external{
		client: ctesting.NewExternalClient().
			WithFixtures(fixtures).
			ExpectGet("GetRepository", repo, nil).
			GetGiteaClient(),
	}

	assert.NotNil(t, ext)
	assert.NotNil(t, ext.client)
}

// Helper function tests for repositoryUpToDate

func TestRepositoryUpToDate_PartialFields(t *testing.T) {
	desc := "Test description"
	private := true

	desired := &repositoryv2.RepositoryParameters{
		Description: &desc,
		Private:     &private,
	}

	actual := &clients.Repository{
		Description: desc,
		Private:     private,
		Archived:    false,
	}

	// When all specified fields match, should be up to date
	result := repositoryUpToDate(desired, actual)
	assert.True(t, result)
}

func TestRepositoryUpToDate_DescriptionMismatch(t *testing.T) {
	desc1 := "Description 1"
	desc2 := "Description 2"

	desired := &repositoryv2.RepositoryParameters{
		Description: &desc1,
	}

	actual := &clients.Repository{
		Description: desc2,
	}

	result := repositoryUpToDate(desired, actual)
	assert.False(t, result)
}

func TestRepositoryUpToDate_PrivateMismatch(t *testing.T) {
	private := true

	desired := &repositoryv2.RepositoryParameters{
		Private: &private,
	}

	actual := &clients.Repository{
		Private: false,
	}

	result := repositoryUpToDate(desired, actual)
	assert.False(t, result)
}

func TestRepositoryUpToDate_ArchivedMismatch(t *testing.T) {
	archived := true

	desired := &repositoryv2.RepositoryParameters{
		Archived: &archived,
	}

	actual := &clients.Repository{
		Archived: false,
	}

	result := repositoryUpToDate(desired, actual)
	assert.False(t, result)
}

func TestRepositoryUpToDate_AllFieldsNil(t *testing.T) {
	desired := &repositoryv2.RepositoryParameters{}

	actual := &clients.Repository{
		Description: "some desc",
		Private:     true,
		Archived:    true,
	}

	// No constraints, so should be up to date
	result := repositoryUpToDate(desired, actual)
	assert.True(t, result)
}
