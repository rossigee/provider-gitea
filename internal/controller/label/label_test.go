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

package label_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
	"github.com/rossigee/provider-gitea/internal/clients"
)

func TestLabelTestFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
}

func TestLabelMockClientExpectations(t *testing.T) {
	labelResponse := &clients.Label{
		ID:          111,
		Name:        "bug",
		Color:       "d73a4a",
		Description: "Something isn't working",
		URL:         "https://example.com/api/repos/org/repo/labels/111",
	}

	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateLabel", labelResponse, nil).
		ExpectGet("GetLabel", labelResponse, nil).
		ExpectUpdate("UpdateLabel", labelResponse, nil).
		ExpectDelete("DeleteLabel", nil)

	assert.NotNil(t, builder.GetGiteaClient())
}

func TestLabelWithoutDescription(t *testing.T) {
	labelResponse := &clients.Label{
		ID:    222,
		Name:  "enhancement",
		Color: "a2eeef",
		URL:   "https://example.com/api/repos/org/repo/labels/222",
	}

	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateLabel", labelResponse, nil)

	assert.NotNil(t, builder.GetGiteaClient())
}
