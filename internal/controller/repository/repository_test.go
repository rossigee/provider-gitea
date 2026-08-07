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

package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/rossigee/provider-gitea/apis/repository/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type mockRepoClient struct {
	testutil.NoopClient
	getRepoFn    func(ctx context.Context, owner, name string) (*clients.Repository, error)
	createRepoFn func(ctx context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error)
	createOrgFn  func(ctx context.Context, org string, req *clients.CreateRepositoryRequest) (*clients.Repository, error)
	updateRepoFn func(ctx context.Context, owner, name string, req *clients.UpdateRepositoryRequest) (*clients.Repository, error)
	deleteRepoFn func(ctx context.Context, owner, name string) error
}

func (m *mockRepoClient) GetRepository(ctx context.Context, owner, name string) (*clients.Repository, error) {
	if m.getRepoFn != nil {
		return m.getRepoFn(ctx, owner, name)
	}
	return nil, nil
}
func (m *mockRepoClient) CreateRepository(ctx context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
	if m.createRepoFn != nil {
		return m.createRepoFn(ctx, req)
	}
	return nil, nil
}
func (m *mockRepoClient) CreateOrganizationRepository(ctx context.Context, org string, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
	if m.createOrgFn != nil {
		return m.createOrgFn(ctx, org, req)
	}
	return nil, nil
}
func (m *mockRepoClient) UpdateRepository(ctx context.Context, owner, name string, req *clients.UpdateRepositoryRequest) (*clients.Repository, error) {
	if m.updateRepoFn != nil {
		return m.updateRepoFn(ctx, owner, name, req)
	}
	return nil, nil
}
func (m *mockRepoClient) DeleteRepository(ctx context.Context, owner, name string) error {
	if m.deleteRepoFn != nil {
		return m.deleteRepoFn(ctx, owner, name)
	}
	return nil
}

func TestObserve(t *testing.T) {
	t.Run("resource not found returns not exists", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				getRepoFn: func(ctx context.Context, owner, name string) (*clients.Repository, error) {
					return nil, fmt.Errorf("API request failed with status 404: not found")
				},
			},
		}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "owner/test-repo")

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("resource exists updates status", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				getRepoFn: func(ctx context.Context, owner, name string) (*clients.Repository, error) {
					return &clients.Repository{
						ID:       123,
						FullName: "owner/test-repo",
						HTMLURL:  "https://gitea.example.com/owner/test-repo",
						SSHURL:   "ssh://gitea@example.com/owner/test-repo.git",
						CloneURL: "https://gitea.example.com/owner/test-repo.git",
						Language: "Go",
					}, nil
				},
			},
		}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "owner/test-repo")

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
		assert.Equal(t, int64(123), *cr.Status.AtProvider.ID)
	})

	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockRepoClient{}}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("invalid external name format treated as not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockRepoClient{}}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "invalid")

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})
}

func TestCreate(t *testing.T) {
	t.Run("creates user repository", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				createRepoFn: func(ctx context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
					assert.Equal(t, "test-repo", req.Name)
					assert.Equal(t, "A test repo", req.Description)
					return &clients.Repository{
						Owner: &clients.User{Username: "testuser"},
						Name:  "test-repo",
					}, nil
				},
			},
		}

		desc := "A test repo"
		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Name:        "test-repo",
					Description: &desc,
				},
			},
		}

		_, err := ec.Create(context.Background(), cr)
		require.NoError(t, err)
		assert.Equal(t, "testuser/test-repo", meta.GetExternalName(cr))
	})

	t.Run("creates organization repository", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				createOrgFn: func(ctx context.Context, org string, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
					assert.Equal(t, "testorg", org)
					return &clients.Repository{
						Owner: &clients.User{Username: "testorg"},
						Name:  "test-repo",
					}, nil
				},
			},
		}

		owner := "testorg"
		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Name:  "test-repo",
					Owner: &owner,
				},
			},
		}

		_, err := ec.Create(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("create failure returns error", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				createRepoFn: func(ctx context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
					return nil, fmt.Errorf("API request failed with status 500: internal error")
				},
			},
		}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Name: "test-repo",
				},
			},
		}

		_, err := ec.Create(context.Background(), cr)
		require.Error(t, err)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("updates repository", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				updateRepoFn: func(ctx context.Context, owner, name string, req *clients.UpdateRepositoryRequest) (*clients.Repository, error) {
					assert.Equal(t, "owner", owner)
					assert.Equal(t, "test-repo", name)
					return &clients.Repository{}, nil
				},
			},
		}

		desc := "Updated description"
		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Description: &desc,
				},
			},
		}
		meta.SetExternalName(cr, "owner/test-repo")

		_, err := ec.Update(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("invalid external name returns error", func(t *testing.T) {
		ec := &externalClient{client: &mockRepoClient{}}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "invalid")

		_, err := ec.Update(context.Background(), cr)
		require.Error(t, err)
	})
}

func TestDelete(t *testing.T) {
	t.Run("deletes repository", func(t *testing.T) {
		deleted := false
		ec := &externalClient{
			client: &mockRepoClient{
				deleteRepoFn: func(ctx context.Context, owner, name string) error {
					assert.Equal(t, "owner", owner)
					assert.Equal(t, "test-repo", name)
					deleted = true
					return nil
				},
			},
		}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "owner/test-repo")

		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})

	t.Run("delete failure returns error", func(t *testing.T) {
		ec := &externalClient{
			client: &mockRepoClient{
				deleteRepoFn: func(ctx context.Context, owner, name string) error {
					return fmt.Errorf("API request failed with status 500: internal error")
				},
			},
		}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "owner/test-repo")

		_, err := ec.Delete(context.Background(), cr)
		require.Error(t, err)
	})

	t.Run("invalid external name returns error", func(t *testing.T) {
		ec := &externalClient{client: &mockRepoClient{}}

		cr := &v2.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-repo",
			},
		}
		meta.SetExternalName(cr, "invalid")

		_, err := ec.Delete(context.Background(), cr)
		require.Error(t, err)
	})
}

func TestIsRepositoryUpToDate(t *testing.T) {
	mkStr := func(s string) *string { return &s }
	mkBool := func(b bool) *bool { return &b }

	baseRepo := &clients.Repository{
		Description:   "desc",
		Private:       true,
		Template:      false,
		Archived:      false,
		DefaultBranch: "main",
	}

	t.Run("matching fields is up to date", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Description:   mkStr("desc"),
					Private:       mkBool(true),
					Template:      mkBool(false),
					Archived:      mkBool(false),
					DefaultBranch: mkStr("main"),
				},
			},
		}
		assert.True(t, isRepositoryUpToDate(cr, baseRepo))
	})

	t.Run("description mismatch is not up to date", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Description: mkStr("new-desc"),
				},
			},
		}
		assert.False(t, isRepositoryUpToDate(cr, baseRepo))
	})

	t.Run("privacy mismatch is not up to date", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Private: mkBool(false),
				},
			},
		}
		assert.False(t, isRepositoryUpToDate(cr, baseRepo))
	})

	t.Run("archived mismatch is not up to date", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Archived: mkBool(true),
				},
			},
		}
		assert.False(t, isRepositoryUpToDate(cr, baseRepo))
	})

	t.Run("template mismatch is not up to date", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Template: mkBool(true),
				},
			},
		}
		assert.False(t, isRepositoryUpToDate(cr, baseRepo))
	})

	t.Run("default branch mismatch is not up to date", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					DefaultBranch: mkStr("develop"),
				},
			},
		}
		assert.False(t, isRepositoryUpToDate(cr, baseRepo))
	})

	t.Run("nil fields are ignored", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{},
			},
		}
		assert.True(t, isRepositoryUpToDate(cr, baseRepo))
	})
}

func TestConnector(t *testing.T) {
	t.Run("missing provider config returns error", func(t *testing.T) {
		c := &connector{kube: fake.NewClientBuilder().Build()}
		_, err := c.Connect(context.Background(), &v2.Repository{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "providerConfigRef is required")
	})
}
