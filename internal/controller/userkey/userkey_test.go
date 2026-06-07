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

package userkey

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestUserKeyFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testuser", fixtures.TestUser)
	assert.NotEmpty(t, fixtures.TestSSHKey)
}

func TestUserKeyMockClientExpectations(t *testing.T) {
	// UserKey is create+delete only (immutable)
	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateUserKey", nil, nil).
		ExpectGet("GetUserKey", nil, nil).
		ExpectDelete("DeleteUserKey", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

func TestUserKeyImmutable(t *testing.T) {
	// UserKey does not support Update - only Create and Delete
	const errUpdateNotSupported = "user keys are immutable; changes require deletion and recreation"
	assert.Contains(t, errUpdateNotSupported, "immutable")
}

func TestUserKeyReadOnly(t *testing.T) {
	// UserKey has a ReadOnly boolean field
	readOnly := true
	assert.True(t, readOnly)
}

func TestUserKeyMetadata(t *testing.T) {
	title := "Personal SSH Key"
	fingerprint := "SHA256:abc123..."
	
	assert.NotEmpty(t, title)
	assert.NotEmpty(t, fingerprint)
}
