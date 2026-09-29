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

package team

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	v2 "github.com/rossigee/provider-gitea/apis/team/v2"
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

type mockTeamClient struct {
	testutil.NoopClient
	getFn    func(ctx context.Context, id int64) (*clients.Team, error)
	createFn func(ctx context.Context, org string, req *clients.CreateTeamRequest) (*clients.Team, error)
	updateFn func(ctx context.Context, id int64, req *clients.UpdateTeamRequest) (*clients.Team, error)
	deleteFn func(ctx context.Context, id int64) error
}

func (m *mockTeamClient) GetTeam(ctx context.Context, id int64) (*clients.Team, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return nil, nil
}
func (m *mockTeamClient) CreateTeam(ctx context.Context, org string, req *clients.CreateTeamRequest) (*clients.Team, error) {
	if m.createFn != nil {
		return m.createFn(ctx, org, req)
	}
	return nil, nil
}
func (m *mockTeamClient) UpdateTeam(ctx context.Context, id int64, req *clients.UpdateTeamRequest) (*clients.Team, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return nil, nil
}
func (m *mockTeamClient) DeleteTeam(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockTeamClient{}}
		cr := &v2.Team{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "t"}}
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockTeamClient{
			getFn: func(ctx context.Context, id int64) (*clients.Team, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.Team{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "t"}}
		meta.SetExternalName(cr, "42")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("matching team is up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockTeamClient{
			getFn: func(ctx context.Context, id int64) (*clients.Team, error) {
				assert.Equal(t, int64(42), id)
				return &clients.Team{ID: 42, Name: "devs", Permission: "write"}, nil
			},
		}}
		cr := &v2.Team{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "t"}}
		cr.Spec.ForProvider.Name = "devs"
		meta.SetExternalName(cr, "42")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("name drift is not up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockTeamClient{
			getFn: func(ctx context.Context, id int64) (*clients.Team, error) {
				return &clients.Team{ID: 42, Name: "old", Permission: "write"}, nil
			},
		}}
		cr := &v2.Team{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "t"}}
		cr.Spec.ForProvider.Name = "new"
		meta.SetExternalName(cr, "42")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	ec := &externalClient{client: &mockTeamClient{
		createFn: func(ctx context.Context, org string, req *clients.CreateTeamRequest) (*clients.Team, error) {
			assert.Equal(t, "acme", org)
			assert.Equal(t, "devs", req.Name)
			return &clients.Team{ID: 42, Name: "devs"}, nil
		},
	}}
	cr := &v2.Team{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "t"}}
	cr.Spec.ForProvider.Organization = "acme"
	cr.Spec.ForProvider.Name = "devs"
	_, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "42", meta.GetExternalName(cr))
}

func TestDelete(t *testing.T) {
	t.Run("missing team is success", func(t *testing.T) {
		ec := &externalClient{client: &mockTeamClient{
			getFn: func(ctx context.Context, id int64) (*clients.Team, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.Team{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "t"}}
		meta.SetExternalName(cr, "42")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("existing team is deleted", func(t *testing.T) {
		var deleted bool
		ec := &externalClient{client: &mockTeamClient{
			getFn: func(ctx context.Context, id int64) (*clients.Team, error) {
				return &clients.Team{ID: 42}, nil
			},
			deleteFn: func(ctx context.Context, id int64) error {
				deleted = true
				return nil
			},
		}}
		cr := &v2.Team{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "t"}}
		meta.SetExternalName(cr, "42")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}

// teamListTestServer emulates GET /api/v1/user/orgs (paginated) and
// GET /api/v1/orgs/{org}/teams for the given orgs and their team IDs.
func teamListTestServer(t *testing.T, orgs map[string][]int64, orgStatus int) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v1/user/orgs" {
			if orgStatus != http.StatusOK {
				w.WriteHeader(orgStatus)
				return
			}

			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

			names := make([]string, 0, len(orgs))
			for n := range orgs {
				names = append(names, n)
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

			_, _ = fmt.Fprint(w, "[")

			for i, n := range names[start:end] {
				if i > 0 {
					_, _ = fmt.Fprint(w, ",")
				}

				_, _ = fmt.Fprintf(w, `{"username":%q}`, n)
			}

			_, _ = fmt.Fprint(w, "]")

			return
		}

		var org string

		if strings.HasPrefix(r.URL.Path, "/api/v1/orgs/") && strings.HasSuffix(r.URL.Path, "/teams") {
			org = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/orgs/"), "/teams")
		}

		ids, ok := orgs[org]
		if org == "" || !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_, _ = fmt.Fprint(w, "[")

		for i, id := range ids {
			if i > 0 {
				_, _ = fmt.Fprint(w, ",")
			}

			_, _ = fmt.Fprintf(w, `{"id":%d,"name":"team-%d"}`, id, id)
		}

		_, _ = fmt.Fprint(w, "]")
	}))
	t.Cleanup(srv.Close)

	return srv
}

func teamListTestSetup(t *testing.T, baseURL string) (*externalClient, *v1beta1.ProviderConfig) {
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

	return &externalClient{client: &mockTeamClient{}, kube: kube}, pc
}

func TestList(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects non-Gitea ProviderConfig", func(t *testing.T) {
		ec := &externalClient{client: &mockTeamClient{}}

		var pc resource.ProviderConfig

		_, err := ec.List(ctx, pc, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a Gitea ProviderConfig")
	})

	t.Run("rejects invalid page tokens", func(t *testing.T) {
		ec := &externalClient{client: &mockTeamClient{}}
		pc := &v1beta1.ProviderConfig{}

		for _, token := range []string{"abc", "0"} {
			_, err := ec.List(ctx, pc, token)
			require.Error(t, err, "token %q", token)
		}
	})

	t.Run("lists teams across organizations", func(t *testing.T) {
		srv := teamListTestServer(t, map[string][]int64{"acme": {7, 3}, "globex": {42}}, http.StatusOK)
		ec, pc := teamListTestSetup(t, srv.URL)

		result, err := ec.List(ctx, pc, "")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"7", "3", "42"}, result.ExternalNames)
		assert.Empty(t, result.NextPageToken)
	})

	t.Run("returns organization listing errors", func(t *testing.T) {
		srv := teamListTestServer(t, nil, http.StatusForbidden)
		ec, pc := teamListTestSetup(t, srv.URL)

		_, err := ec.List(ctx, pc, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list organizations")
	})
}
