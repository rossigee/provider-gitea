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

package clients

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *v1beta1.ProviderConfig
		secret  *corev1.Secret
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful client creation",
			config: &v1beta1.ProviderConfig{
				Spec: v1beta1.ProviderConfigSpec{
					BaseURL: "https://gitea.example.com",
					Credentials: v1beta1.ProviderCredentials{
						Source: "Secret",
						SecretRef: &v1beta1.SecretReference{
							Name:      "test-secret",
							Namespace: "test-namespace",
							Key:       "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"token": []byte("test-token"),
				},
			},
			wantErr: false,
		},
		{
			name: "missing baseURL",
			config: &v1beta1.ProviderConfig{
				Spec: v1beta1.ProviderConfigSpec{
					BaseURL: "",
					Credentials: v1beta1.ProviderCredentials{
						Source: "Secret",
					},
				},
			},
			wantErr: true,
			errMsg:  "baseURL is required",
		},
		{
			name: "unsupported credential source",
			config: &v1beta1.ProviderConfig{
				Spec: v1beta1.ProviderConfigSpec{
					BaseURL: "https://gitea.example.com",
					Credentials: v1beta1.ProviderCredentials{
						Source: "Environment",
					},
				},
			},
			wantErr: true,
			errMsg:  "only Secret credential source is supported",
		},
		{
			name: "missing secret",
			config: &v1beta1.ProviderConfig{
				Spec: v1beta1.ProviderConfigSpec{
					BaseURL: "https://gitea.example.com",
					Credentials: v1beta1.ProviderCredentials{
						Source: "Secret",
						SecretRef: &v1beta1.SecretReference{
							Name:      "missing-secret",
							Namespace: "test-namespace",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "failed to get secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake kubernetes client
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			
			objects := []runtime.Object{}
			if tt.secret != nil {
				objects = append(objects, tt.secret)
			}
			
			kubeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objects...).
				Build()

			// Create client
			client, err := NewClient(context.Background(), tt.config, kubeClient)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestRepositoryOperations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/repos/testorg/testrepo":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 123,
				"name": "testrepo",
				"full_name": "testorg/testrepo",
				"description": "Test repository",
				"private": false,
				"fork": false,
				"template": false,
				"empty": false,
				"archived": false,
				"size": 1024,
				"html_url": "https://gitea.example.com/testorg/testrepo",
				"ssh_url": "git@gitea.example.com:testorg/testrepo.git",
				"clone_url": "https://gitea.example.com/testorg/testrepo.git",
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-02T00:00:00Z"
			}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/user/repos":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 124,
				"name": "newrepo",
				"full_name": "testuser/newrepo",
				"description": "New repository",
				"private": false
			}`))
		case r.Method == "PATCH" && r.URL.Path == "/api/v1/repos/testorg/testrepo":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 123,
				"name": "testrepo",
				"description": "Updated description"
			}`))
		case r.Method == "DELETE" && r.URL.Path == "/api/v1/repos/testorg/testrepo":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test client
	c := &giteaClient{
		httpClient: &http.Client{},
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
	}

	ctx := context.Background()

	t.Run("GetRepository", func(t *testing.T) {
		repo, err := c.GetRepository(ctx, "testorg", "testrepo")
		require.NoError(t, err)
		assert.Equal(t, int64(123), repo.ID)
		assert.Equal(t, "testrepo", repo.Name)
		assert.Equal(t, "testorg/testrepo", repo.FullName)
		assert.Equal(t, "Test repository", repo.Description)
	})

	t.Run("CreateRepository", func(t *testing.T) {
		req := &CreateRepositoryRequest{
			Name:        "newrepo",
			Description: "New repository",
			Private:     false,
		}
		repo, err := c.CreateRepository(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, int64(124), repo.ID)
		assert.Equal(t, "newrepo", repo.Name)
	})

	t.Run("UpdateRepository", func(t *testing.T) {
		desc := "Updated description"
		req := &UpdateRepositoryRequest{
			Description: &desc,
		}
		repo, err := c.UpdateRepository(ctx, "testorg", "testrepo", req)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", repo.Description)
	})

	t.Run("DeleteRepository", func(t *testing.T) {
		err := c.DeleteRepository(ctx, "testorg", "testrepo")
		require.NoError(t, err)
	})
}

func TestWebhookOperations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/repos/testorg/testrepo/hooks/1":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"type": "gitea",
				"config": {
					"url": "https://example.com/webhook",
					"content_type": "json"
				},
				"events": ["push", "pull_request"],
				"active": true,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-02T00:00:00Z"
			}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/repos/testorg/testrepo/hooks":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 2,
				"type": "gitea",
				"config": {
					"url": "https://example.com/webhook",
					"content_type": "json"
				},
				"events": ["push"],
				"active": true
			}`))
		case r.Method == "PATCH" && r.URL.Path == "/api/v1/repos/testorg/testrepo/hooks/1":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"type": "gitea",
				"active": false
			}`))
		case r.Method == "DELETE" && r.URL.Path == "/api/v1/repos/testorg/testrepo/hooks/1":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test client
	c := &giteaClient{
		httpClient: &http.Client{},
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
	}

	ctx := context.Background()

	t.Run("GetRepositoryWebhook", func(t *testing.T) {
		webhook, err := c.GetRepositoryWebhook(ctx, "testorg", "testrepo", 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), webhook.ID)
		assert.Equal(t, "gitea", webhook.Type)
		assert.True(t, webhook.Active)
	})

	t.Run("CreateRepositoryWebhook", func(t *testing.T) {
		req := &CreateWebhookRequest{
			Type: "gitea",
			Config: map[string]string{
				"url":          "https://example.com/webhook",
				"content_type": "json",
			},
			Events: []string{"push"},
			Active: true,
		}
		webhook, err := c.CreateRepositoryWebhook(ctx, "testorg", "testrepo", req)
		require.NoError(t, err)
		assert.Equal(t, int64(2), webhook.ID)
		assert.Equal(t, "gitea", webhook.Type)
	})

	t.Run("UpdateRepositoryWebhook", func(t *testing.T) {
		active := false
		req := &UpdateWebhookRequest{
			Active: &active,
		}
		webhook, err := c.UpdateRepositoryWebhook(ctx, "testorg", "testrepo", 1, req)
		require.NoError(t, err)
		assert.False(t, webhook.Active)
	})

	t.Run("DeleteRepositoryWebhook", func(t *testing.T) {
		err := c.DeleteRepositoryWebhook(ctx, "testorg", "testrepo", 1)
		require.NoError(t, err)
	})
}

func TestOrganizationOperations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/orgs/testorg":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"username": "testorg",
				"name": "Test Organization",
				"full_name": "Test Organization Inc",
				"description": "Test organization description",
				"website": "https://testorg.example.com",
				"location": "San Francisco",
				"visibility": "public",
				"repo_admin_change_team_access": true,
				"email": "contact@testorg.example.com"
			}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/orgs":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 2,
				"username": "neworg",
				"name": "New Organization",
				"description": "New organization",
				"visibility": "public"
			}`))
		case r.Method == "PATCH" && r.URL.Path == "/api/v1/orgs/testorg":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"username": "testorg",
				"description": "Updated description"
			}`))
		case r.Method == "DELETE" && r.URL.Path == "/api/v1/orgs/testorg":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test client
	c := &giteaClient{
		httpClient: &http.Client{},
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
	}

	ctx := context.Background()

	t.Run("GetOrganization", func(t *testing.T) {
		org, err := c.GetOrganization(ctx, "testorg")
		require.NoError(t, err)
		assert.Equal(t, int64(1), org.ID)
		assert.Equal(t, "testorg", org.Username)
		assert.Equal(t, "Test Organization", org.Name)
		assert.Equal(t, "Test organization description", org.Description)
		assert.Equal(t, "public", org.Visibility)
	})

	t.Run("CreateOrganization", func(t *testing.T) {
		req := &CreateOrganizationRequest{
			Username:    "neworg",
			Name:        "New Organization",
			Description: "New organization",
			Visibility:  "public",
		}
		org, err := c.CreateOrganization(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, int64(2), org.ID)
		assert.Equal(t, "neworg", org.Username)
	})

	t.Run("UpdateOrganization", func(t *testing.T) {
		desc := "Updated description"
		req := &UpdateOrganizationRequest{
			Description: &desc,
		}
		org, err := c.UpdateOrganization(ctx, "testorg", req)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", org.Description)
	})

	t.Run("DeleteOrganization", func(t *testing.T) {
		err := c.DeleteOrganization(ctx, "testorg")
		require.NoError(t, err)
	})
}

func TestUserOperations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/users/testuser":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"username": "testuser",
				"name": "Test User",
				"full_name": "Test User Full Name",
				"email": "testuser@example.com",
				"avatar_url": "https://example.com/avatar.png",
				"is_admin": false,
				"created": "2024-01-01T00:00:00Z",
				"active": true,
				"restricted": false
			}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/admin/users":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 2,
				"username": "newuser",
				"email": "newuser@example.com",
				"full_name": "New User"
			}`))
		case r.Method == "PATCH" && r.URL.Path == "/api/v1/admin/users/testuser":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"username": "testuser",
				"full_name": "Updated Full Name"
			}`))
		case r.Method == "DELETE" && r.URL.Path == "/api/v1/admin/users/testuser":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test client
	c := &giteaClient{
		httpClient: &http.Client{},
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
	}

	ctx := context.Background()

	t.Run("GetUser", func(t *testing.T) {
		user, err := c.GetUser(ctx, "testuser")
		require.NoError(t, err)
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "testuser@example.com", user.Email)
		assert.False(t, user.IsAdmin)
		assert.True(t, user.Active)
	})

	t.Run("CreateUser", func(t *testing.T) {
		req := &CreateUserRequest{
			Username: "newuser",
			Email:    "newuser@example.com",
			FullName: "New User",
			Password: "password123",
		}
		user, err := c.CreateUser(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, int64(2), user.ID)
		assert.Equal(t, "newuser", user.Username)
		assert.Equal(t, "newuser@example.com", user.Email)
	})

	t.Run("UpdateUser", func(t *testing.T) {
		fullName := "Updated Full Name"
		req := &UpdateUserRequest{
			FullName: &fullName,
		}
		user, err := c.UpdateUser(ctx, "testuser", req)
		require.NoError(t, err)
		assert.Equal(t, "Updated Full Name", user.FullName)
	})

	t.Run("DeleteUser", func(t *testing.T) {
		err := c.DeleteUser(ctx, "testuser")
		require.NoError(t, err)
	})
}

func TestDeployKeyOperations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/repos/testorg/testrepo/keys/1":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC...",
				"title": "test-key",
				"read_only": true,
				"created_at": "2024-01-01T00:00:00Z"
			}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/repos/testorg/testrepo/keys":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 2,
				"key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD...",
				"title": "new-key",
				"read_only": false
			}`))
		case r.Method == "DELETE" && r.URL.Path == "/api/v1/repos/testorg/testrepo/keys/1":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test client
	c := &giteaClient{
		httpClient: &http.Client{},
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
	}

	ctx := context.Background()

	t.Run("GetDeployKey", func(t *testing.T) {
		key, err := c.GetDeployKey(ctx, "testorg", "testrepo", 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), key.ID)
		assert.Equal(t, "test-key", key.Title)
		assert.True(t, key.ReadOnly)
		assert.Contains(t, key.Key, "ssh-rsa")
	})

	t.Run("CreateDeployKey", func(t *testing.T) {
		req := &CreateDeployKeyRequest{
			Title:    "new-key",
			Key:      "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD...",
			ReadOnly: false,
		}
		key, err := c.CreateDeployKey(ctx, "testorg", "testrepo", req)
		require.NoError(t, err)
		assert.Equal(t, int64(2), key.ID)
		assert.Equal(t, "new-key", key.Title)
		assert.False(t, key.ReadOnly)
	})

	t.Run("DeleteDeployKey", func(t *testing.T) {
		err := c.DeleteDeployKey(ctx, "testorg", "testrepo", 1)
		require.NoError(t, err)
	})
}

func TestOrganizationWebhookOperations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/orgs/testorg/hooks/1":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"type": "gitea",
				"config": {
					"url": "https://example.com/org-webhook",
					"content_type": "json"
				},
				"events": ["organization", "repository"],
				"active": true
			}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/orgs/testorg/hooks":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 2,
				"type": "gitea",
				"config": {
					"url": "https://example.com/new-org-webhook"
				},
				"events": ["organization"],
				"active": true
			}`))
		case r.Method == "PATCH" && r.URL.Path == "/api/v1/orgs/testorg/hooks/1":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"type": "gitea",
				"active": false
			}`))
		case r.Method == "DELETE" && r.URL.Path == "/api/v1/orgs/testorg/hooks/1":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test client
	c := &giteaClient{
		httpClient: &http.Client{},
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
	}

	ctx := context.Background()

	t.Run("GetOrganizationWebhook", func(t *testing.T) {
		webhook, err := c.GetOrganizationWebhook(ctx, "testorg", 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), webhook.ID)
		assert.Equal(t, "gitea", webhook.Type)
		assert.True(t, webhook.Active)
		assert.Contains(t, webhook.Events, "organization")
	})

	t.Run("CreateOrganizationWebhook", func(t *testing.T) {
		req := &CreateWebhookRequest{
			Type: "gitea",
			Config: map[string]string{
				"url": "https://example.com/new-org-webhook",
			},
			Events: []string{"organization"},
			Active: true,
		}
		webhook, err := c.CreateOrganizationWebhook(ctx, "testorg", req)
		require.NoError(t, err)
		assert.Equal(t, int64(2), webhook.ID)
		assert.Equal(t, "gitea", webhook.Type)
	})

	t.Run("UpdateOrganizationWebhook", func(t *testing.T) {
		active := false
		req := &UpdateWebhookRequest{
			Active: &active,
		}
		webhook, err := c.UpdateOrganizationWebhook(ctx, "testorg", 1, req)
		require.NoError(t, err)
		assert.False(t, webhook.Active)
	})

	t.Run("DeleteOrganizationWebhook", func(t *testing.T) {
		err := c.DeleteOrganizationWebhook(ctx, "testorg", 1)
		require.NoError(t, err)
	})
}

func TestOrganizationSecretOperations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/orgs/testorg/actions/secrets/testsecret"):
			// Gitea returns 405 for GET operations on organization secrets
			w.Header().Set("Allow", "PUT, DELETE")
			w.WriteHeader(http.StatusMethodNotAllowed)
			
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/orgs/testorg/actions/secrets/testsecret"):
			// Verify request body
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var req map[string]interface{}
			err = json.Unmarshal(body, &req)
			require.NoError(t, err)
			assert.Equal(t, "test-secret-value", req["data"])
			
			w.WriteHeader(http.StatusCreated)
			
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/orgs/testorg/actions/secrets/newsecret"):
			w.WriteHeader(http.StatusCreated)
			
		case r.Method == "DELETE" && strings.Contains(r.URL.Path, "/orgs/testorg/actions/secrets/testsecret"):
			w.WriteHeader(http.StatusNoContent)
			
		default:
			t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test client
	c := &giteaClient{
		httpClient: &http.Client{},
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
	}

	ctx := context.Background()

	t.Run("GetOrganizationSecret_Returns405", func(t *testing.T) {
		// Test that Gitea API returns 405 for GET operations
		secret, err := c.GetOrganizationSecret(ctx, "testorg", "testsecret")
		assert.Error(t, err)
		assert.Nil(t, secret)
		assert.Contains(t, err.Error(), "405")
	})

	t.Run("CreateOrganizationSecret", func(t *testing.T) {
		req := &CreateOrganizationSecretRequest{
			Data: "test-secret-value",
		}
		err := c.CreateOrganizationSecret(ctx, "testorg", "testsecret", req)
		require.NoError(t, err)
	})

	t.Run("UpdateOrganizationSecret", func(t *testing.T) {
		req := &CreateOrganizationSecretRequest{
			Data: "updated-secret-value",
		}
		err := c.UpdateOrganizationSecret(ctx, "testorg", "newsecret", req)
		require.NoError(t, err)
	})

	t.Run("DeleteOrganizationSecret", func(t *testing.T) {
		err := c.DeleteOrganizationSecret(ctx, "testorg", "testsecret")
		require.NoError(t, err)
	})
}