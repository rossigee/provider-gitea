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

package organization_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestOrganizationTestFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testuser", fixtures.TestUser)
}

func TestOrganizationResponseBuilder(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()
	org := fixtures.OrganizationResponse()

	assert.Equal(t, fixtures.TestOrg, org.Username)
	assert.Equal(t, "Test Organization", org.FullName)
	assert.Equal(t, int64(101), org.ID)
}

func TestOrganizationExternalClientBuilder(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	builder := ctesting.NewExternalClient().
		WithFixtures(fixtures).
		ExpectGet("GetOrganization", fixtures.OrganizationResponse(), nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

func TestOrganizationMockClientExpectations(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateOrganization", fixtures.OrganizationResponse(), nil).
		ExpectGet("GetOrganization", fixtures.OrganizationResponse(), nil).
		ExpectUpdate("UpdateOrganization", fixtures.OrganizationResponse(), nil).
		ExpectDelete("DeleteOrganization", nil)

	assert.NotNil(t, builder.GetGiteaClient())
}
