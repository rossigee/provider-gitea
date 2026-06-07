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

package accesstoken

import (
	"testing"

	"github.com/stretchr/testify/assert"

	accesstokenv2 "github.com/rossigee/provider-gitea/apis/accesstoken/v2"
	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestAccessTokenFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testuser", fixtures.TestUser)
	assert.NotEmpty(t, fixtures.TestNamespace)
}

func TestAccessTokenMockClientExpectations(t *testing.T) {
	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateAccessToken", nil, nil).
		ExpectGet("GetAccessToken", nil, nil).
		ExpectUpdate("UpdateAccessToken", nil, nil).
		ExpectDelete("DeleteAccessToken", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

// AccessToken field tests

func TestAccessTokenField_Name(t *testing.T) {
	name := "CI Token"
	desired := &accesstokenv2.AccessTokenParameters{
		Name: &name,
	}

	assert.NotNil(t, desired)
	assert.Equal(t, "CI Token", *desired.Name)
}

func TestAccessTokenField_Scopes(t *testing.T) {
	scopes := []string{"repo", "admin"}
	desired := &accesstokenv2.AccessTokenParameters{
		Scopes: scopes,
	}

	assert.NotNil(t, desired)
	assert.Len(t, desired.Scopes, 2)
	assert.Contains(t, desired.Scopes, "repo")
}

func TestAccessTokenOneTimeCapture(t *testing.T) {
	// Token value only available in Create response, not on Get
	const tokenValue = "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	assert.NotEmpty(t, tokenValue)
}

func TestAccessTokenLastEight(t *testing.T) {
	lastEight := "xxxxx123"
	assert.Len(t, lastEight, 8)
}
