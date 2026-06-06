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

package deploykey_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
	"github.com/rossigee/provider-gitea/internal/clients"
)

func TestDeployKeyTestFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.NotEmpty(t, fixtures.TestSSHKey)
	assert.Contains(t, fixtures.TestSSHKey, "ssh-ed25519")
}

func TestDeployKeyMockClientExpectations(t *testing.T) {
	deployKeyResponse := &clients.DeployKey{
		ID:          789,
		Title:       "Deploy Key",
		Key:         "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG4rT3vTt99Ox5kndS4HmgTrKBT8F0E6tpHkEF/ULo5U",
		ReadOnly:    true,
		URL:         "https://example.com/api/repos/org/repo/keys/789",
		Fingerprint: "fingerprint123",
	}

	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateDeployKey", deployKeyResponse, nil).
		ExpectGet("GetDeployKey", deployKeyResponse, nil).
		ExpectDelete("DeleteDeployKey", nil)

	assert.NotNil(t, builder.GetGiteaClient())
}

func TestDeployKeyUpdateNotSupported(t *testing.T) {
	builder := ctesting.NewExternalClient().
		ExpectDelete("DeleteDeployKey", nil)

	assert.NotNil(t, builder.GetGiteaClient())
}
