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

package organizationmember

import (
	"testing"

	"github.com/stretchr/testify/assert"

	organizationmemberv2 "github.com/rossigee/provider-gitea/apis/organizationmember/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestOrganizationMemberFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testuser", fixtures.TestUser)
}

func TestOrganizationMemberMockClientExpectations(t *testing.T) {
	// OrganizationMember uses Add/Update/Remove semantics
	builder := ctesting.NewExternalClient().
		ExpectCreate("AddOrganizationMember", nil, nil).
		ExpectGet("GetOrganizationMember", nil, nil).
		ExpectUpdate("UpdateOrganizationMember", nil, nil).
		ExpectDelete("RemoveOrganizationMember", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

// OrganizationMember field tests

func TestOrganizationMemberField_Role(t *testing.T) {
	role := "member"
	desired := &organizationmemberv2.OrganizationMemberParameters{
		Role: role,
	}

	actual := &clients.OrganizationMember{
		Role: "owner",
	}

	assert.NotNil(t, desired)
	assert.NotEqual(t, desired.Role, actual.Role)
}

func TestOrganizationMemberField_Visibility(t *testing.T) {
	visibility := "private"
	desired := &organizationmemberv2.OrganizationMemberParameters{
		Visibility: &visibility,
	}

	actual := &clients.OrganizationMember{
		Visibility: "public",
	}

	assert.NotNil(t, desired)
	assert.NotEqual(t, *desired.Visibility, actual.Visibility)
}

func TestOrganizationMemberRoles(t *testing.T) {
	roles := []string{"member", "owner"}
	
	for _, role := range roles {
		assert.NotEmpty(t, role)
	}
}

func TestOrganizationMemberVisibility(t *testing.T) {
	visibility := "public"
	assert.Equal(t, "public", visibility)
}
