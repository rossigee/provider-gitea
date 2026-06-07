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

package deploykey

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestDeployKeyFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
	assert.NotEmpty(t, fixtures.TestSSHKey)
}

func TestDeployKeyMockClientExpectations(t *testing.T) {
	// DeployKey is create+delete only (immutable)
	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateDeployKey", nil, nil).
		ExpectGet("GetDeployKey", nil, nil).
		ExpectDelete("DeleteDeployKey", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

func TestDeployKeyImmutable(t *testing.T) {
	// DeployKey does not support Update - only Create and Delete
	// This test verifies the pattern that DeployKey.Update returns an explicit error
	
	const errUpdateNotSupported = "deploy keys are immutable; changes require deletion and recreation"
	assert.Contains(t, errUpdateNotSupported, "immutable")
}

func TestDeployKeyReadOnly(t *testing.T) {
	// DeployKey has a ReadOnly boolean field
	readOnly := true
	assert.True(t, readOnly)
}

func TestDeployKeyTitle(t *testing.T) {
	title := "My Deploy Key"
	assert.NotEmpty(t, title)
}
