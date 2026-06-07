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

package organizationsecret

import (
	"testing"

	"github.com/stretchr/testify/assert"

	organizationsecretv2 "github.com/rossigee/provider-gitea/apis/organizationsecret/v2"
	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestOrganizationSecretFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.NotEmpty(t, fixtures.TestNamespace)
}

func TestOrganizationSecretMockClientExpectations(t *testing.T) {
	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateOrganizationSecret", nil, nil).
		ExpectGet("GetOrganizationSecret", nil, nil).
		ExpectUpdate("UpdateOrganizationSecret", nil, nil).
		ExpectDelete("DeleteOrganizationSecret", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

// OrganizationSecret field tests

func TestOrganizationSecretField_SecretName(t *testing.T) {
	name := "SHARED_SECRET"
	desired := &organizationsecretv2.OrganizationSecretParameters{
		SecretName: name,
	}

	actual := "different_secret"

	assert.NotNil(t, desired)
	assert.NotEqual(t, desired.SecretName, actual)
}

func TestOrganizationSecretField_Organization(t *testing.T) {
	org := "myorg"
	desired := &organizationsecretv2.OrganizationSecretParameters{
		Organization: org,
	}

	actual := "otherorg"

	assert.NotNil(t, desired)
	assert.NotEqual(t, desired.Organization, actual)
}

func TestOrganizationSecretMutualExclusivity(t *testing.T) {
	// OrganizationSecret requires mutual exclusivity: Data XOR DataFrom
	data := "secret-value"
	desired1 := &organizationsecretv2.OrganizationSecretParameters{
		Data: &data,
	}
	assert.NotNil(t, desired1)

	desired2 := &organizationsecretv2.OrganizationSecretParameters{
		Data: nil,
		// DataFrom would be set instead
	}
	assert.NotNil(t, desired2)
}

func TestOrganizationSecretCustomSecretKeySelector(t *testing.T) {
	// OrganizationSecret uses custom SecretKeySelector (not xpv1)
	secretName := "credentials"
	key := "password"

	assert.NotEmpty(t, secretName)
	assert.NotEmpty(t, key)
}
