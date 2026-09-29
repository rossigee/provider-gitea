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
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	v2 "github.com/rossigee/provider-gitea/apis/repository/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		assert.True(t, isRepositoryUpToDate(cr, baseRepo, nil))
	})

	t.Run("description mismatch is not up to date", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Description: mkStr("new-desc"),
				},
			},
		}
		assert.False(t, isRepositoryUpToDate(cr, baseRepo, nil))
	})

	t.Run("privacy mismatch is not up to date", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Private: mkBool(false),
				},
			},
		}
		assert.False(t, isRepositoryUpToDate(cr, baseRepo, nil))
	})

	t.Run("archived mismatch is not up to date", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Archived: mkBool(true),
				},
			},
		}
		assert.False(t, isRepositoryUpToDate(cr, baseRepo, nil))
	})

	t.Run("template mismatch is not up to date", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					Template: mkBool(true),
				},
			},
		}
		assert.False(t, isRepositoryUpToDate(cr, baseRepo, nil))
	})

	t.Run("default branch mismatch is not up to date", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{
					DefaultBranch: mkStr("develop"),
				},
			},
		}
		assert.False(t, isRepositoryUpToDate(cr, baseRepo, nil))
	})

	t.Run("nil fields are ignored", func(t *testing.T) {
		cr := &v2.Repository{
			Spec: v2.RepositorySpec{
				ForProvider: v2.RepositoryParameters{},
			},
		}
		assert.True(t, isRepositoryUpToDate(cr, baseRepo, nil))
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

// listTestServer returns an httptest server emulating GET /api/v1/user/repos
// with page/limit pagination over the given full names, plus the requests seen.
func listTestServer(t *testing.T, names []string, status int) (*httptest.Server, *[][]string) {
	t.Helper()

	var seen [][]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Helper()

		assert.Equal(t, "/api/v1/user/repos", r.URL.Path)
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		seen = append(seen, []string{r.URL.Query().Get("page"), r.URL.Query().Get("limit")})

		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		if page < 1 {
			page = 1
		}

		start := (page - 1) * limit
		if start > len(names) {
			start = len(names)
		}

		end := start + limit
		if end > len(names) {
			end = len(names)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, "[")

		for i, n := range names[start:end] {
			if i > 0 {
				_, _ = fmt.Fprint(w, ",")
			}

			_, _ = fmt.Fprintf(w, `{"full_name":%q}`, n)
		}

		_, _ = fmt.Fprint(w, "]")
	}))

	t.Cleanup(srv.Close)

	return srv, &seen
}

func listTestProviderConfig(baseURL string) (*v1beta1.ProviderConfig, *corev1.Secret) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gitea-creds"},
		Data:       map[string][]byte{"token": []byte("test-token")},
	}

	pc := &v1beta1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-pc"},
		Spec: v1beta1.ProviderConfigSpec{
			BaseURL: baseURL,
			Credentials: v1beta1.ProviderCredentials{
				Source: "Secret",
				SecretRef: &v1beta1.SecretReference{
					Namespace: "default",
					Name:      "gitea-creds",
					Key:       "token",
				},
			},
		},
	}

	return pc, secret
}

func listTestKube(t *testing.T, objs ...runtime.Object) *fake.ClientBuilder {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))

	return fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...)
}

func repoNames(prefix string, n int) []string {
	names := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		names = append(names, fmt.Sprintf("%s/repo-%d", prefix, i))
	}

	return names
}

func TestList(t *testing.T) {
	ctx := context.Background()

	newExternal := func(t *testing.T, objs ...runtime.Object) *externalClient {
		t.Helper()

		return &externalClient{
			client: &mockRepoClient{},
			kube:   listTestKube(t, objs...).Build(),
		}
	}

	t.Run("rejects non-Gitea ProviderConfig", func(t *testing.T) {
		ec := newExternal(t)

		var pc resource.ProviderConfig

		_, err := ec.List(ctx, pc, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a Gitea ProviderConfig")
	})

	t.Run("rejects invalid page tokens", func(t *testing.T) {
		ec := newExternal(t)
		pc := &v1beta1.ProviderConfig{}

		for _, token := range []string{"abc", "0", "-2", "1.5"} {
			_, err := ec.List(ctx, pc, token)
			require.Error(t, err, "token %q", token)
			assert.Contains(t, err.Error(), "invalid page token", "token %q", token)
		}
	})

	t.Run("requires baseURL", func(t *testing.T) {
		pc, secret := listTestProviderConfig("")
		ec := newExternal(t, secret)

		_, err := ec.List(ctx, pc, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "baseURL")
	})

	t.Run("returns next token on full page", func(t *testing.T) {
		names := repoNames("acme", 50)
		srv, seen := listTestServer(t, names, http.StatusOK)

		pc, secret := listTestProviderConfig(srv.URL)
		ec := newExternal(t, secret)

		result, err := ec.List(ctx, pc, "")
		require.NoError(t, err)
		assert.Equal(t, names, result.ExternalNames)
		assert.Equal(t, "2", result.NextPageToken)
		require.Len(t, *seen, 1)
		assert.Equal(t, []string{"1", "50"}, (*seen)[0])
	})

	t.Run("omits next token on last page", func(t *testing.T) {
		names := repoNames("acme", 2)
		srv, _ := listTestServer(t, names, http.StatusOK)

		pc, secret := listTestProviderConfig(srv.URL)
		ec := newExternal(t, secret)

		result, err := ec.List(ctx, pc, "3")
		require.NoError(t, err)
		assert.Empty(t, result.ExternalNames)
		assert.Empty(t, result.NextPageToken)
	})

	t.Run("skips repositories without names", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `[{"full_name":"acme/ok"},{"id":7}]`)
		}))
		t.Cleanup(srv.Close)

		pc, secret := listTestProviderConfig(srv.URL)
		ec := newExternal(t, secret)

		result, err := ec.List(ctx, pc, "")
		require.NoError(t, err)
		assert.Equal(t, []string{"acme/ok"}, result.ExternalNames)
	})

	t.Run("returns API errors", func(t *testing.T) {
		srv, _ := listTestServer(t, nil, http.StatusInternalServerError)

		pc, secret := listTestProviderConfig(srv.URL)
		ec := newExternal(t, secret)

		_, err := ec.List(ctx, pc, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list repositories")
	})
}
