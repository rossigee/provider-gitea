package runner

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func runnerTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/user/repos":
			_, _ = fmt.Fprint(w, `[{"full_name":"acme/r1"}]`)
		case "/api/v1/user/orgs":
			_, _ = fmt.Fprint(w, `[{"username":"acme"}]`)
		case "/api/v1/repos/acme/r1/actions/runners":
			_, _ = fmt.Fprint(w, `{"runners":[{"id":1}]}`)
		case "/api/v1/orgs/acme/actions/runners":
			_, _ = fmt.Fprint(w, `{"runners":[{"id":2}]}`)
		case "/api/v1/admin/actions/runners":
			_, _ = fmt.Fprint(w, `{"runners":[{"id":3}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	return srv
}

func runnerTestSetup(t *testing.T, baseURL string) (*externalClient, *v1beta1.ProviderConfig) {
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

	return &externalClient{client: testutil.NoopClient{}, kube: kube}, pc
}

func TestList(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects non-Gitea ProviderConfig", func(t *testing.T) {
		ec := &externalClient{client: testutil.NoopClient{}}

		var pc resource.ProviderConfig

		_, err := ec.List(ctx, pc, resource.ExternalListScope{"scope": "system"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a Gitea ProviderConfig")
	})

	t.Run("requires a scope", func(t *testing.T) {
		ec := &externalClient{client: testutil.NoopClient{}}
		pc := &v1beta1.ProviderConfig{}

		_, err := ec.List(ctx, pc, nil, "")
		require.ErrorIs(t, err, resource.ErrScopeRequired)
	})

	t.Run("lists runners of each scope", func(t *testing.T) {
		srv := runnerTestServer(t)
		ec, pc := runnerTestSetup(t, srv.URL)

		repo, err := ec.List(ctx, pc, resource.ExternalListScope{"scope": "repository", "scopeValue": "acme/r1"}, "")
		require.NoError(t, err)
		assert.Equal(t, []string{"repository/acme/r1/1"}, testutil.ItemNames(repo.Items))
		assert.Equal(t, resource.ExternalListScope{"scope": "repository", "scopeValue": "acme/r1"}, repo.Items[0].Scope)

		org, err := ec.List(ctx, pc, resource.ExternalListScope{"scope": "organization", "scopeValue": "acme"}, "")
		require.NoError(t, err)
		assert.Equal(t, []string{"organization/acme/2"}, testutil.ItemNames(org.Items))
		assert.Equal(t, resource.ExternalListScope{"scope": "organization", "scopeValue": "acme"}, org.Items[0].Scope)

		system, err := ec.List(ctx, pc, resource.ExternalListScope{"scope": "system"}, "")
		require.NoError(t, err)
		assert.Equal(t, []string{"system/3"}, testutil.ItemNames(system.Items))
		assert.Equal(t, resource.ExternalListScope{"scope": "system"}, system.Items[0].Scope)
	})

	t.Run("drains repository, organization, then system scopes", func(t *testing.T) {
		srv := runnerTestServer(t)
		ec, pc := runnerTestSetup(t, srv.URL)

		first, err := ec.ListScopes(ctx, pc, nil, "")
		require.NoError(t, err)
		assert.Equal(t, []resource.ExternalListScope{{"scope": "repository", "scopeValue": "acme/r1"}}, first.Scopes)
		assert.Equal(t, "orgs:1", first.NextPageToken)

		second, err := ec.ListScopes(ctx, pc, nil, first.NextPageToken)
		require.NoError(t, err)
		assert.Equal(t, []resource.ExternalListScope{{"scope": "organization", "scopeValue": "acme"}}, second.Scopes)
		assert.Equal(t, "system:1", second.NextPageToken)

		third, err := ec.ListScopes(ctx, pc, nil, second.NextPageToken)
		require.NoError(t, err)
		assert.Equal(t, []resource.ExternalListScope{{"scope": "system"}}, third.Scopes)
		assert.Empty(t, third.NextPageToken)
	})

	t.Run("rejects invalid tokens and phases when listing scopes", func(t *testing.T) {
		ec := &externalClient{client: testutil.NoopClient{}}
		pc := &v1beta1.ProviderConfig{}

		for _, token := range []string{"abc", "bogus:1"} {
			_, err := ec.ListScopes(ctx, pc, nil, token)
			require.Error(t, err, "token %q", token)
		}
	})
}
