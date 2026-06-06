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

package user_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestUserTestFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testuser", fixtures.TestUser)
	assert.Equal(t, "testuser@example.com", fixtures.TestEmail)
}

func TestUserResponseBuilder(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()
	user := fixtures.UserResponse()

	assert.Equal(t, fixtures.TestUser, user.Username)
	assert.Equal(t, fixtures.TestEmail, user.Email)
	assert.Equal(t, "Test User", user.FullName)
	assert.True(t, user.Active)
	assert.Equal(t, int64(789), user.ID)
}

func TestUserExternalClientBuilder(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	builder := ctesting.NewExternalClient().
		WithFixtures(fixtures).
		ExpectGet("GetUser", fixtures.UserResponse(), nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

func TestUserMockClientExpectations(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateUser", fixtures.UserResponse(), nil).
		ExpectGet("GetUser", fixtures.UserResponse(), nil).
		ExpectUpdate("UpdateUser", fixtures.UserResponse(), nil).
		ExpectDelete("DeleteUser", nil)

	assert.NotNil(t, builder.GetGiteaClient())
}
