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

package organization

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	v2 "github.com/rossigee/provider-gitea/apis/organization/v2"
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

type mockOrgClient struct {
	testutil.NoopClient
	getOrgFn    func(ctx context.Context, name string) (*clients.Organization, error)
	createOrgFn func(ctx context.Context, req *clients.CreateOrganizationRequest) (*clients.Organization, error)
	updateOrgFn func(ctx context.Context, name string, req *clients.UpdateOrganizationRequest) (*clients.Organization, error)
	deleteOrgFn func(ctx context.Context, name string) error
}

func (m *mockOrgClient) GetOrganization(ctx context.Context, name string) (*clients.Organization, error) {
	if m.getOrgFn != nil {
		return m.getOrgFn(ctx, name)
	}
	return nil, nil
}
func (m *mockOrgClient) CreateOrganization(ctx context.Context, req *clients.CreateOrganizationRequest) (*clients.Organization, error) {
	if m.createOrgFn != nil {
		return m.createOrgFn(ctx, req)
	}
	return nil, nil
}
func (m *mockOrgClient) UpdateOrganization(ctx context.Context, name string, req *clients.UpdateOrganizationRequest) (*clients.Organization, error) {
	if m.updateOrgFn != nil {
		return m.updateOrgFn(ctx, name, req)
	}
	return nil, nil
}
func (m *mockOrgClient) DeleteOrganization(ctx context.Context, name string) error {
	if m.deleteOrgFn != nil {
		return m.deleteOrgFn(ctx, name)
	}
	return nil
}

func TestIsOrganizationUpToDate(t *testing.T) {
	testCases := []struct {
		name   string
		crSpec v2.OrganizationParameters
		org    *clients.Organization
		want   bool
	}{
		{
			name: "up to date",
			crSpec: v2.OrganizationParameters{
				Username:    "test-org",
				Name:        strPtr("Test Org"),
				Description: strPtr("Test description"),
			},
			org: &clients.Organization{
				Username:    "test-org",
				FullName:    "Test Org",
				Description: "Test description",
			},
			want: true,
		},
		{
			name: "needs update - name",
			crSpec: v2.OrganizationParameters{
				Username: "test-org",
				Name:     strPtr("New Name"),
			},
			org: &clients.Organization{
				Username: "test-org",
				FullName: "Old Name",
			},
			want: false,
		},
		{
			name: "needs update - description",
			crSpec: v2.OrganizationParameters{
				Username:    "test-org",
				Description: strPtr("New description"),
			},
			org: &clients.Organization{
				Username:    "test-org",
				Description: "Old description",
			},
			want: false,
		},
		{
			name: "needs update - location",
			crSpec: v2.OrganizationParameters{
				Username: "test-org",
				Location: strPtr("New Location"),
			},
			org: &clients.Organization{
				Username: "test-org",
				Location: "Old Location",
			},
			want: false,
		},
		{
			name: "needs update - website",
			crSpec: v2.OrganizationParameters{
				Username: "test-org",
				Website:  strPtr("https://new-site.com"),
			},
			org: &clients.Organization{
				Username: "test-org",
				Website:  "https://old-site.com",
			},
			want: false,
		},
		{
			name: "needs update - visibility",
			crSpec: v2.OrganizationParameters{
				Username:   "test-org",
				Visibility: strPtr("private"),
			},
			org: &clients.Organization{
				Username:   "test-org",
				Visibility: "public",
			},
			want: false,
		},
		{
			name: "up to date - nil optional fields",
			crSpec: v2.OrganizationParameters{
				Username: "test-org",
			},
			org: &clients.Organization{
				Username: "test-org",
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cr := &v2.Organization{
				Spec: v2.OrganizationSpec{
					ForProvider: tc.crSpec,
				},
			}
			result := isOrganizationUpToDate(cr, tc.org)
			assert.Equal(t, tc.want, result)
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockOrgClient{}}
		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
		}

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{
			client: &mockOrgClient{
				getOrgFn: func(ctx context.Context, name string) (*clients.Organization, error) {
					return nil, fmt.Errorf("API request failed with status 404: not found")
				},
			},
		}
		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
		}
		meta.SetExternalName(cr, "test-org")

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("non-404 returns error", func(t *testing.T) {
		ec := &externalClient{
			client: &mockOrgClient{
				getOrgFn: func(ctx context.Context, name string) (*clients.Organization, error) {
					return nil, fmt.Errorf("API request failed with status 500: internal error")
				},
			},
		}
		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
		}
		meta.SetExternalName(cr, "test-org")

		_, err := ec.Observe(context.Background(), cr)
		require.Error(t, err)
	})

	t.Run("resource exists updates status", func(t *testing.T) {
		ec := &externalClient{
			client: &mockOrgClient{
				getOrgFn: func(ctx context.Context, name string) (*clients.Organization, error) {
					return &clients.Organization{
						ID:         1,
						Username:   "test-org",
						FullName:   "Test Org",
						NumRepos:   5,
						NumMembers: 3,
					}, nil
				},
			},
		}
		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
			Spec: v2.OrganizationSpec{
				ForProvider: v2.OrganizationParameters{Username: "test-org"},
			},
		}
		meta.SetExternalName(cr, "test-org")

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
		require.NotNil(t, cr.Status.AtProvider.ID)
		assert.Equal(t, int64(1), *cr.Status.AtProvider.ID)
	})

	t.Run("drift detected returns not up to date", func(t *testing.T) {
		ec := &externalClient{
			client: &mockOrgClient{
				getOrgFn: func(ctx context.Context, name string) (*clients.Organization, error) {
					return &clients.Organization{
						Username:    "test-org",
						FullName:    "Old Name",
						Description: "Old description",
					}, nil
				},
			},
		}
		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
			Spec: v2.OrganizationSpec{
				ForProvider: v2.OrganizationParameters{
					Username:    "test-org",
					Name:        strPtr("New Name"),
					Description: strPtr("New description"),
				},
			},
		}
		meta.SetExternalName(cr, "test-org")

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	t.Run("creates organization with full spec", func(t *testing.T) {
		ec := &externalClient{
			client: &mockOrgClient{
				createOrgFn: func(ctx context.Context, req *clients.CreateOrganizationRequest) (*clients.Organization, error) {
					assert.Equal(t, "test-org", req.Username)
					assert.Equal(t, "Test Org", req.FullName)
					assert.Equal(t, "A test org", req.Description)
					return &clients.Organization{
						ID:       42,
						Username: "test-org",
					}, nil
				},
			},
		}

		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
			Spec: v2.OrganizationSpec{
				ForProvider: v2.OrganizationParameters{
					Username:    "test-org",
					Name:        strPtr("Test Org"),
					Description: strPtr("A test org"),
				},
			},
		}

		_, err := ec.Create(context.Background(), cr)
		require.NoError(t, err)
		assert.Equal(t, "test-org", meta.GetExternalName(cr))
		require.NotNil(t, cr.Status.AtProvider.ID)
		assert.Equal(t, int64(42), *cr.Status.AtProvider.ID)
	})

	t.Run("create failure returns error", func(t *testing.T) {
		ec := &externalClient{
			client: &mockOrgClient{
				createOrgFn: func(ctx context.Context, req *clients.CreateOrganizationRequest) (*clients.Organization, error) {
					return nil, fmt.Errorf("API request failed with status 500: internal error")
				},
			},
		}

		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
			Spec: v2.OrganizationSpec{
				ForProvider: v2.OrganizationParameters{Username: "test-org"},
			},
		}

		_, err := ec.Create(context.Background(), cr)
		require.Error(t, err)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("updates organization", func(t *testing.T) {
		updated := false
		ec := &externalClient{
			client: &mockOrgClient{
				updateOrgFn: func(ctx context.Context, name string, req *clients.UpdateOrganizationRequest) (*clients.Organization, error) {
					assert.Equal(t, "test-org", name)
					updated = true
					return &clients.Organization{Username: "test-org"}, nil
				},
			},
		}

		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
			Spec: v2.OrganizationSpec{
				ForProvider: v2.OrganizationParameters{
					Username:    "test-org",
					Description: strPtr("updated desc"),
				},
			},
		}
		meta.SetExternalName(cr, "test-org")

		_, err := ec.Update(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, updated)
	})

	t.Run("missing external name returns error", func(t *testing.T) {
		ec := &externalClient{client: &mockOrgClient{}}
		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
		}

		_, err := ec.Update(context.Background(), cr)
		require.Error(t, err)
	})
}

func TestDelete(t *testing.T) {
	t.Run("deletes organization when exists", func(t *testing.T) {
		deleted := false
		ec := &externalClient{
			client: &mockOrgClient{
				getOrgFn: func(ctx context.Context, name string) (*clients.Organization, error) {
					return &clients.Organization{Username: name}, nil
				},
				deleteOrgFn: func(ctx context.Context, name string) error {
					assert.Equal(t, "test-org", name)
					deleted = true
					return nil
				},
			},
		}

		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
		}
		meta.SetExternalName(cr, "test-org")

		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})

	t.Run("404 returns nil (already deleted)", func(t *testing.T) {
		deleted := false
		ec := &externalClient{
			client: &mockOrgClient{
				getOrgFn: func(ctx context.Context, name string) (*clients.Organization, error) {
					return nil, fmt.Errorf("API request failed with status 404: not found")
				},
				deleteOrgFn: func(ctx context.Context, name string) error {
					deleted = true
					return nil
				},
			},
		}

		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
		}
		meta.SetExternalName(cr, "test-org")

		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, deleted, "Delete should not be called when external resource is already gone")
	})

	t.Run("missing external name returns nil", func(t *testing.T) {
		ec := &externalClient{client: &mockOrgClient{}}
		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
		}

		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("delete failure returns error", func(t *testing.T) {
		ec := &externalClient{
			client: &mockOrgClient{
				getOrgFn: func(ctx context.Context, name string) (*clients.Organization, error) {
					return &clients.Organization{Username: name}, nil
				},
				deleteOrgFn: func(ctx context.Context, name string) error {
					return fmt.Errorf("API request failed with status 500: internal error")
				},
			},
		}

		cr := &v2.Organization{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-org"},
		}
		meta.SetExternalName(cr, "test-org")

		_, err := ec.Delete(context.Background(), cr)
		require.Error(t, err)
	})
}

func orgListTestServer(t *testing.T, names []string, status int) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/user/orgs", r.URL.Path)
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

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

			_, _ = fmt.Fprintf(w, `{"username":%q}`, n)
		}

		_, _ = fmt.Fprint(w, "]")
	}))
	t.Cleanup(srv.Close)

	return srv
}

func orgListTestSetup(t *testing.T, baseURL string) (*externalClient, *v1beta1.ProviderConfig) {
	t.Helper()

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

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))

	kube := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(secret).Build()

	return &externalClient{client: &mockOrgClient{}, kube: kube}, pc
}

func TestList(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects non-Gitea ProviderConfig", func(t *testing.T) {
		ec := &externalClient{client: &mockOrgClient{}}

		var pc resource.ProviderConfig

		_, err := ec.List(ctx, pc, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a Gitea ProviderConfig")
	})

	t.Run("rejects invalid page tokens", func(t *testing.T) {
		ec := &externalClient{client: &mockOrgClient{}}
		pc := &v1beta1.ProviderConfig{}

		for _, token := range []string{"abc", "0", "-1"} {
			_, err := ec.List(ctx, pc, token)
			require.Error(t, err, "token %q", token)
		}
	})

	t.Run("returns next token on full page", func(t *testing.T) {
		names := make([]string, 0, 50)
		for i := 1; i <= 50; i++ {
			names = append(names, fmt.Sprintf("org-%d", i))
		}

		srv := orgListTestServer(t, names, http.StatusOK)
		ec, pc := orgListTestSetup(t, srv.URL)

		result, err := ec.List(ctx, pc, "")
		require.NoError(t, err)
		assert.Equal(t, names, result.ExternalNames)
		assert.Equal(t, "2", result.NextPageToken)
	})

	t.Run("omits next token on last page", func(t *testing.T) {
		srv := orgListTestServer(t, []string{"org-a", "org-b"}, http.StatusOK)
		ec, pc := orgListTestSetup(t, srv.URL)

		result, err := ec.List(ctx, pc, "")
		require.NoError(t, err)
		assert.Equal(t, []string{"org-a", "org-b"}, result.ExternalNames)
		assert.Empty(t, result.NextPageToken)
	})

	t.Run("returns API errors", func(t *testing.T) {
		srv := orgListTestServer(t, nil, http.StatusForbidden)
		ec, pc := orgListTestSetup(t, srv.URL)

		_, err := ec.List(ctx, pc, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list organizations")
	})
}
