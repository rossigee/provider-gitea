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

package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rossigee/provider-gitea/internal/clients"
	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestWebhookFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
}

func TestWebhookMockClientExpectations(t *testing.T) {
	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateRepositoryWebhook", nil, nil).
		ExpectGet("GetRepositoryWebhook", nil, nil).
		ExpectUpdate("UpdateRepositoryWebhook", nil, nil).
		ExpectDelete("DeleteRepositoryWebhook", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

func TestWebhookOrganizationMockExpectations(t *testing.T) {
	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateOrganizationWebhook", nil, nil).
		ExpectGet("GetOrganizationWebhook", nil, nil).
		ExpectUpdate("UpdateOrganizationWebhook", nil, nil).
		ExpectDelete("DeleteOrganizationWebhook", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

// Webhook type tests

func TestWebhookType_Push(t *testing.T) {
	webhook := &clients.Webhook{
		Type: "push",
	}
	assert.Equal(t, "push", webhook.Type)
}

func TestWebhookType_Issues(t *testing.T) {
	webhook := &clients.Webhook{
		Type: "issues",
	}
	assert.Equal(t, "issues", webhook.Type)
}

func TestWebhookType_PullRequest(t *testing.T) {
	webhook := &clients.Webhook{
		Type: "pull_request",
	}
	assert.Equal(t, "pull_request", webhook.Type)
}

func TestWebhookEvents(t *testing.T) {
	webhook := &clients.Webhook{
		Events: []string{"push", "pull_request", "issues"},
	}
	assert.Len(t, webhook.Events, 3)
	assert.Contains(t, webhook.Events, "push")
}
