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
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/rossigee/provider-gitea/internal/clients"
	giteamock "github.com/rossigee/provider-gitea/test/mock"
)

// TestFixtures provides common test data used across all controller tests.
// It includes consistent user, organization, repository, and credential data
// to ensure predictable test behavior across the entire test suite.
type TestFixtures struct {
	TestUser      string // Default test username: "testuser"
	TestEmail     string // Default test email: "testuser@example.com" 
	TestOrg       string // Default test organization: "testorg"
	TestRepo      string // Default test repository: "testrepo"
	TestNamespace string // Default Kubernetes namespace: "default"
	TestSSHKey    string // Valid SSH ED25519 public key for testing
}

// NewTestFixtures creates a new set of test fixtures with sensible defaults.
// Returns a TestFixtures struct populated with consistent test data that can be
// used across all controller tests to ensure predictable behavior.
func NewTestFixtures() *TestFixtures {
	return &TestFixtures{
		TestUser:      "testuser",
		TestEmail:     "testuser@example.com",
		TestOrg:       "testorg",
		TestRepo:      "testrepo",
		TestNamespace: "default",
		TestSSHKey:    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG4rT3vTt99Ox5kndS4HmgTrKBT8F0E6tpHkEF/ULo5U test@example.com",
	}
}

// MockClientBuilder provides a fluent interface for creating Gitea mock clients.
// This builder simplifies the process of setting up mock expectations for Gitea API calls,
// allowing tests to specify expected method calls and return values in a readable way.
//
// Example usage:
//   mockClient := NewMockClient().
//       ExpectMethod("CreateRepository", expectedResponse, nil).
//       ExpectMethod("GetRepository", existingResponse, nil).
//       Build()
type MockClientBuilder struct {
	client *giteamock.Client
}

// NewMockClient creates a new mock client builder.
// Returns a MockClientBuilder that can be used to configure mock expectations
// for Gitea API calls using a fluent interface.
func NewMockClient() *MockClientBuilder {
	return &MockClientBuilder{
		client: &giteamock.Client{},
	}
}

// ExpectMethod adds a mock expectation for a Gitea API method call.
// This method configures the mock client to expect a specific method call and return
// the provided values. It supports up to 5 arguments using mock.Anything for flexibility.
//
// Parameters:
//   method: The name of the Gitea API method to mock (e.g., "CreateRepository")
//   returnValues: Variable number of values to return when the method is called
//
// Example:
//   builder.ExpectMethod("CreateRepository", repositoryResponse, nil)
//   builder.ExpectMethod("GetRepository", repositoryResponse, nil)
func (b *MockClientBuilder) ExpectMethod(method string, returnValues ...interface{}) *MockClientBuilder {
	call := b.client.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	
	if len(returnValues) > 0 {
		call.Return(returnValues...)
	} else {
		call.Return(nil)
	}
	return b
}

// Build returns the configured mock client
func (b *MockClientBuilder) Build() *giteamock.Client {
	return b.client
}

// K8sSecretBuilder provides a fluent interface for creating Kubernetes secrets
type K8sSecretBuilder struct {
	secret *corev1.Secret
}

// NewSecret creates a new secret builder
func NewSecret(name, namespace string) *K8sSecretBuilder {
	return &K8sSecretBuilder{
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: make(map[string][]byte),
		},
	}
}

// WithPasswordData adds password data to the secret
func (b *K8sSecretBuilder) WithPasswordData(password string) *K8sSecretBuilder {
	b.secret.Data["password"] = []byte(password)
	return b
}

// WithValueData adds value data to the secret
func (b *K8sSecretBuilder) WithValueData(value string) *K8sSecretBuilder {
	b.secret.Data["value"] = []byte(value)
	return b
}

// WithData adds custom data to the secret
func (b *K8sSecretBuilder) WithData(key, value string) *K8sSecretBuilder {
	b.secret.Data[key] = []byte(value)
	return b
}

// Build returns the configured secret
func (b *K8sSecretBuilder) Build() *corev1.Secret {
	return b.secret
}

// K8sClientBuilder provides a fluent interface for creating Kubernetes clients
type K8sClientBuilder struct {
	scheme  *runtime.Scheme
	objects []client.Object
}

// NewK8sClient creates a new Kubernetes client builder
func NewK8sClient() *K8sClientBuilder {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return &K8sClientBuilder{
		scheme:  scheme,
		objects: []client.Object{},
	}
}

// WithSecret adds a secret to the client
func (b *K8sClientBuilder) WithSecret(secret *corev1.Secret) *K8sClientBuilder {
	b.objects = append(b.objects, secret)
	return b
}

// WithSecrets adds multiple secrets to the client
func (b *K8sClientBuilder) WithSecrets(secrets ...*corev1.Secret) *K8sClientBuilder {
	for _, secret := range secrets {
		b.objects = append(b.objects, secret)
	}
	return b
}

// Build returns the configured Kubernetes client
func (b *K8sClientBuilder) Build() client.Client {
	return fake.NewClientBuilder().
		WithScheme(b.scheme).
		WithObjects(b.objects...).
		Build()
}

// Response builders for Gitea client types
//
// These methods generate consistent response objects that match the structure
// expected by controller tests. They use the fixture's test data to create
// realistic responses that can be used in mock expectations.

// RepositoryResponse creates a repository response object for testing.
// Returns a *clients.Repository populated with test data from the fixtures.
// This is commonly used in mock expectations for repository-related API calls.
//
// Example usage:
//   response := fixtures.RepositoryResponse()
//   mockClient.ExpectMethod("GetRepository", response, nil)
func (f *TestFixtures) RepositoryResponse() *clients.Repository {
	return &clients.Repository{
		ID:       123,
		Name:     f.TestRepo,
		FullName: f.TestOrg + "/" + f.TestRepo,
		Private:  true,
		Owner: &clients.User{
			ID:       456,
			Username: f.TestOrg,
			FullName: "Test Organization",
			Email:    "org@example.com",
		},
		Description: "Test repository description",
	}
}

// UserResponse creates a basic user response  
func (f *TestFixtures) UserResponse() *clients.User {
	return &clients.User{
		ID:       789,
		Username: f.TestUser,
		FullName: "Test User",
		Email:    f.TestEmail,
		Active:   true,
	}
}

// OrganizationResponse creates a basic organization response
func (f *TestFixtures) OrganizationResponse() *clients.Organization {
	return &clients.Organization{
		ID:          101,
		Username:    f.TestOrg,
		FullName:    "Test Organization",
		Description: "Test organization description",
		Website:     "https://example.com",
		Location:    "Test Location",
	}
}