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

package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestRepositoryTestFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testuser", fixtures.TestUser)
	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
	assert.Equal(t, "testuser@example.com", fixtures.TestEmail)
}

func TestRepositoryResponseBuilder(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()
	repo := fixtures.RepositoryResponse()

	assert.Equal(t, fixtures.TestRepo, repo.Name)
	assert.Equal(t, fixtures.TestOrg+"/"+fixtures.TestRepo, repo.FullName)
	assert.True(t, repo.Private)
	assert.Equal(t, int64(123), repo.ID)
}

func TestExternalClientBuilder(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	builder := ctesting.NewExternalClient().
		WithFixtures(fixtures).
		ExpectGet("GetRepository", fixtures.RepositoryResponse(), nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetFixtures())
	assert.Equal(t, fixtures.TestOrg, builder.GetFixtures().TestOrg)
}

func TestMockClientExpectations(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateRepository", fixtures.RepositoryResponse(), nil).
		ExpectGet("GetRepository", fixtures.RepositoryResponse(), nil).
		ExpectUpdate("UpdateRepository", fixtures.RepositoryResponse(), nil).
		ExpectDelete("DeleteRepository", nil)

	assert.NotNil(t, builder.GetGiteaClient())
}
