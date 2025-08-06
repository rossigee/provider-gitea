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

package testing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTestInfrastructure demonstrates basic test infrastructure usage
func TestTestInfrastructure(t *testing.T) {
	fixtures := NewTestFixtures()
	
	// Test basic fixtures
	assert.Equal(t, "testuser", fixtures.TestUser)
	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
	
	// Test repository response
	repo := fixtures.RepositoryResponse()
	assert.Equal(t, fixtures.TestRepo, repo.Name)
	assert.Equal(t, fixtures.TestOrg+"/"+fixtures.TestRepo, repo.FullName)
	assert.True(t, repo.Private)
	
	// Test user response
	user := fixtures.UserResponse()
	assert.Equal(t, fixtures.TestUser, user.Username)
	assert.Equal(t, fixtures.TestEmail, user.Email)
	assert.True(t, user.Active)
}

// TestSecretBuilder demonstrates secret builder usage
func TestSecretBuilder(t *testing.T) {
	// Test password secret
	passwordSecret := NewSecret("user-password", "default").
		WithPasswordData("supersecret123").
		Build()
	
	assert.Equal(t, "user-password", passwordSecret.Name)
	assert.Equal(t, "default", passwordSecret.Namespace)
	assert.Equal(t, "supersecret123", string(passwordSecret.Data["password"]))
	
	// Test value secret
	valueSecret := NewSecret("api-secret", "default").
		WithValueData("apikey123").
		Build()
	
	assert.Equal(t, "api-secret", valueSecret.Name)
	assert.Equal(t, "apikey123", string(valueSecret.Data["value"]))
}

// TestK8sClientBuilder demonstrates Kubernetes client builder usage
func TestK8sClientBuilder(t *testing.T) {
	secret := NewSecret("test-secret", "default").
		WithValueData("secret-value").
		Build()
	
	client := NewK8sClient().
		WithSecret(secret).
		Build()
	
	assert.NotNil(t, client)
}