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

package webhook_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
	"github.com/rossigee/provider-gitea/internal/clients"
)

func TestWebhookTestFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
	assert.NotEmpty(t, fixtures.TestNamespace)
}

func TestWebhookMockClientExpectations(t *testing.T) {
	webhookResponse := &clients.Webhook{
		ID:     123,
		URL:    "https://example.com/webhook",
		Active: true,
	}

	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateRepositoryWebhook", webhookResponse, nil).
		ExpectGet("GetRepositoryWebhook", webhookResponse, nil).
		ExpectUpdate("UpdateRepositoryWebhook", webhookResponse, nil).
		ExpectDelete("DeleteRepositoryWebhook", nil)

	assert.NotNil(t, builder.GetGiteaClient())
}

func TestWebhookOrganizationMockExpectations(t *testing.T) {
	webhookResponse := &clients.Webhook{
		ID:     456,
		URL:    "https://example.com/org-webhook",
		Active: true,
	}

	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateOrganizationWebhook", webhookResponse, nil).
		ExpectGet("GetOrganizationWebhook", webhookResponse, nil).
		ExpectUpdate("UpdateOrganizationWebhook", webhookResponse, nil).
		ExpectDelete("DeleteOrganizationWebhook", nil)

	assert.NotNil(t, builder.GetGiteaClient())
}
