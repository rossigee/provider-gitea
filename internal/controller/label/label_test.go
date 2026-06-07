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

package label

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rossigee/provider-gitea/internal/clients"
	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestLabelFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
}

func TestLabelMockClientExpectations(t *testing.T) {
	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateLabel", nil, nil).
		ExpectGet("GetLabel", nil, nil).
		ExpectUpdate("UpdateLabel", nil, nil).
		ExpectDelete("DeleteLabel", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

func TestLabelColor(t *testing.T) {
	label := &clients.Label{
		Name:  "bug",
		Color: "ff0000",
	}
	assert.Equal(t, "ff0000", label.Color)
}

func TestLabelName(t *testing.T) {
	label := &clients.Label{
		Name: "enhancement",
	}
	assert.Equal(t, "enhancement", label.Name)
}

func TestLabelDescription(t *testing.T) {
	label := &clients.Label{
		Name:        "documentation",
		Description: "Improvements or additions to documentation",
	}
	assert.Equal(t, "Improvements or additions to documentation", label.Description)
}

func TestLabelExclusive(t *testing.T) {
	label := &clients.Label{
		Name:      "priority",
		Exclusive: true,
	}
	assert.True(t, label.Exclusive)
}

func TestLabelID(t *testing.T) {
	label := &clients.Label{
		ID:   int64(42),
		Name: "test-label",
	}
	assert.Equal(t, int64(42), label.ID)
}
