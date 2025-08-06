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
	"sigs.k8s.io/controller-runtime/pkg/client"

	giteamock "github.com/rossigee/provider-gitea/test/mock"
)

// ExternalClientBuilder provides a fluent interface for creating test external clients
type ExternalClientBuilder struct {
	giteaClient *giteamock.Client
	kubeClient  client.Client
	fixtures    *TestFixtures
}

// NewExternalClient creates a new external client builder
func NewExternalClient() *ExternalClientBuilder {
	return &ExternalClientBuilder{
		giteaClient: &giteamock.Client{},
		fixtures:    NewTestFixtures(),
	}
}

// WithFixtures sets custom test fixtures
func (b *ExternalClientBuilder) WithFixtures(fixtures *TestFixtures) *ExternalClientBuilder {
	b.fixtures = fixtures
	return b
}

// WithGiteaClient sets a custom Gitea client
func (b *ExternalClientBuilder) WithGiteaClient(client *giteamock.Client) *ExternalClientBuilder {
	b.giteaClient = client
	return b
}

// WithKubernetesClient sets a custom Kubernetes client
func (b *ExternalClientBuilder) WithKubernetesClient(client client.Client) *ExternalClientBuilder {
	b.kubeClient = client
	return b
}

// WithSecrets creates a Kubernetes client with the specified secrets
func (b *ExternalClientBuilder) WithSecrets(secrets ...*corev1.Secret) *ExternalClientBuilder {
	b.kubeClient = NewK8sClient().WithSecrets(secrets...).Build()
	return b
}

// WithPasswordSecret creates a Kubernetes client with a password secret
func (b *ExternalClientBuilder) WithPasswordSecret(name, namespace, password string) *ExternalClientBuilder {
	secret := NewSecret(name, namespace).WithPasswordData(password).Build()
	return b.WithSecrets(secret)
}

// WithValueSecret creates a Kubernetes client with a value secret
func (b *ExternalClientBuilder) WithValueSecret(name, namespace, value string) *ExternalClientBuilder {
	secret := NewSecret(name, namespace).WithValueData(value).Build()
	return b.WithSecrets(secret)
}

// Common mock expectations for standard CRUD operations

// ExpectCreate adds create operation expectations
func (b *ExternalClientBuilder) ExpectCreate(method string, response interface{}, err error) *ExternalClientBuilder {
	var call *mock.Call
	
	switch method {
	case "CreateRepository":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	case "CreateUser":
		call = b.giteaClient.On(method, mock.Anything)
	case "CreateOrganization":
		call = b.giteaClient.On(method, mock.Anything)
	case "CreateTeam":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	case "CreateWebhook":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	case "CreateAccessToken":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "CreateDeployKey":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "CreateUserKey":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "CreateRepositoryKey":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "CreateGitHook":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "CreateIssue":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "CreatePullRequest":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "CreateRelease":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "CreateLabel":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "CreateBranchProtection":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "AddRepositoryCollaborator":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "AddOrganizationMember":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "CreateOrganizationSecret":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "CreateRepositorySecret":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "CreateAdminUser":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	case "CreateAction":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "CreateRunner":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	default:
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	}

	if err != nil {
		if method == "CreateOrganizationSecret" || method == "CreateRepositorySecret" {
			call.Return(err)
		} else {
			call.Return(nil, err)
		}
	} else {
		if method == "CreateOrganizationSecret" || method == "CreateRepositorySecret" {
			call.Return(nil)
		} else {
			call.Return(response, nil)
		}
	}
	return b
}

// ExpectGet adds get operation expectations
func (b *ExternalClientBuilder) ExpectGet(method string, response interface{}, err error) *ExternalClientBuilder {
	var call *mock.Call
	
	switch method {
	case "GetRepository":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetUser":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	case "GetOrganization":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	case "GetTeam":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetWebhook":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetAccessToken":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetDeployKey":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "GetUserKey":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetRepositoryKey":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetGitHook":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetIssue":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetPullRequest":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetRelease":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetLabel":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetBranchProtection":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "GetRepositoryCollaborator":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "GetOrganizationMember":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetOrganizationSettings":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	case "GetAdminUser":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	case "GetAction":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "GetRunner":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	default:
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	}

	if err != nil {
		call.Return(nil, err)
	} else {
		call.Return(response, nil)
	}
	return b
}

// ExpectUpdate adds update operation expectations
func (b *ExternalClientBuilder) ExpectUpdate(method string, response interface{}, err error) *ExternalClientBuilder {
	var call *mock.Call
	
	switch method {
	case "UpdateRepository":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateUser":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateOrganization":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateTeam":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateWebhook":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateUserKey":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateRepositoryKey":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateGitHook":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateIssue":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdatePullRequest":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateRelease":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateLabel":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateBranchProtection":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateOrganizationSettings":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateOrganizationSecret":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateRepositorySecret":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateAdminUser":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateAction":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "UpdateRunner":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	default:
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	}

	if err != nil {
		if method == "UpdateOrganizationSecret" || method == "UpdateRepositorySecret" {
			call.Return(err)
		} else {
			call.Return(nil, err)
		}
	} else {
		if method == "UpdateOrganizationSecret" || method == "UpdateRepositorySecret" {
			call.Return(nil)
		} else if response != nil {
			call.Return(response, nil)
		} else {
			call.Return(nil)
		}
	}
	return b
}

// ExpectDelete adds delete operation expectations
func (b *ExternalClientBuilder) ExpectDelete(method string, err error) *ExternalClientBuilder {
	var call *mock.Call
	
	switch method {
	case "DeleteRepository":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteUser":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	case "DeleteOrganization":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	case "DeleteTeam":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteWebhook":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteAccessToken":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteDeployKey":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteUserKey":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteRepositoryKey":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteGitHook":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteIssue":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeletePullRequest":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteRelease":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteLabel":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteBranchProtection":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "RemoveRepositoryCollaborator":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	case "RemoveOrganizationMember":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteOrganizationSecret":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteRepositorySecret":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteAdminUser":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	case "DeleteAction":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything, mock.Anything)
	case "DeleteRunner":
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	default:
		call = b.giteaClient.On(method, mock.Anything, mock.Anything)
	}

	call.Return(err)
	return b
}

// GetGiteaClient returns the Gitea mock client
func (b *ExternalClientBuilder) GetGiteaClient() *giteamock.Client {
	return b.giteaClient
}

// GetKubeClient returns the Kubernetes client
func (b *ExternalClientBuilder) GetKubeClient() client.Client {
	return b.kubeClient
}

// GetFixtures returns the test fixtures
func (b *ExternalClientBuilder) GetFixtures() *TestFixtures {
	return b.fixtures
}