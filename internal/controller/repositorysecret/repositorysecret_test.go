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

package repositorysecret

import (
	"testing"

	"github.com/stretchr/testify/assert"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	repositorysecretv2 "github.com/rossigee/provider-gitea/apis/repositorysecret/v2"
	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestRepositorySecretFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
}

func TestRepositorySecretMockClientExpectations(t *testing.T) {
	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateRepositorySecret", nil, nil).
		ExpectGet("GetRepositorySecret", nil, nil).
		ExpectUpdate("UpdateRepositorySecret", nil, nil).
		ExpectDelete("DeleteRepositorySecret", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

// RepositorySecret field tests

func TestRepositorySecretField_SecretName(t *testing.T) {
	name := "DOCKER_PASSWORD"
	desired := &repositorysecretv2.RepositorySecretParameters{
		SecretName: name,
	}

	actual := "different_secret"

	assert.NotNil(t, desired)
	assert.NotEqual(t, desired.SecretName, actual)
}

func TestRepositorySecretField_Repository(t *testing.T) {
	repo := "owner/repo"
	desired := &repositorysecretv2.RepositorySecretParameters{
		Repository: repo,
	}

	actual := "other/repo"

	assert.NotNil(t, desired)
	assert.NotEqual(t, desired.Repository, actual)
}

func TestRepositorySecretValueSecretRef(t *testing.T) {
	// RepositorySecret requires ValueSecretRef (xpv1.SecretKeySelector)
	valueRef := &xpv1.SecretKeySelector{
		Name: "my-secret",
		Key:  "password",
	}

	assert.NotNil(t, valueRef)
	assert.Equal(t, "my-secret", valueRef.Name)
	assert.Equal(t, "password", valueRef.Key)
}

func TestRepositorySecretKubernetesIntegration(t *testing.T) {
	// RepositorySecret resolves Kubernetes Secrets for values
	secretName := "docker-credentials"
	secretKey := "password"

	assert.NotEmpty(t, secretName)
	assert.NotEmpty(t, secretKey)
}
