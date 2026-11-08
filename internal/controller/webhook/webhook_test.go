package webhook

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

func webhookTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/user/repos":
			_, _ = fmt.Fprint(w, `[{"full_name":"acme/r1"}]`)
		case "/api/v1/user/orgs":
			_, _ = fmt.Fprint(w, `[{"username":"acme"}]`)
		case "/api/v1/repos/acme/r1/hooks":
			_, _ = fmt.Fprint(w, `[{"id":9}]`)
		case "/api/v1/orgs/acme/hooks":
			_, _ = fmt.Fprint(w, `[{"id":4}]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	return srv
}

func webhookTestSetup(t *testing.T, baseURL string) (*externalClient, *v1beta1.ProviderConfig) {
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

		_, err := ec.List(ctx, pc, resource.ExternalListScope{"organization": "acme"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a Gitea ProviderConfig")
	})

	t.Run("requires a repository or organization scope", func(t *testing.T) {
		ec := &externalClient{client: testutil.NoopClient{}}
		pc := &v1beta1.ProviderConfig{}

		_, err := ec.List(ctx, pc, nil, "")
		require.ErrorIs(t, err, resource.ErrScopeRequired)
	})

	t.Run("lists repository webhooks", func(t *testing.T) {
		srv := webhookTestServer(t)
		ec, pc := webhookTestSetup(t, srv.URL)

		result, err := ec.List(ctx, pc, resource.ExternalListScope{"owner": "acme", "repository": "r1"}, "")
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "acme/r1/9", result.Items[0].ExternalName)
		assert.Equal(t, resource.ExternalListScope{"owner": "acme", "repository": "r1"}, result.Items[0].Scope)
	})

	t.Run("lists organization webhooks", func(t *testing.T) {
		srv := webhookTestServer(t)
		ec, pc := webhookTestSetup(t, srv.URL)

		result, err := ec.List(ctx, pc, resource.ExternalListScope{"organization": "acme"}, "")
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "acme/4", result.Items[0].ExternalName)
		assert.Equal(t, resource.ExternalListScope{"organization": "acme"}, result.Items[0].Scope)
	})

	t.Run("drains repository then organization scopes", func(t *testing.T) {
		srv := webhookTestServer(t)
		ec, pc := webhookTestSetup(t, srv.URL)

		first, err := ec.ListScopes(ctx, pc, nil, "")
		require.NoError(t, err)
		assert.Equal(t, []resource.ExternalListScope{{"owner": "acme", "repository": "r1"}}, first.Scopes)
		assert.Equal(t, "orgs:1", first.NextPageToken)

		second, err := ec.ListScopes(ctx, pc, nil, first.NextPageToken)
		require.NoError(t, err)
		assert.Equal(t, []resource.ExternalListScope{{"organization": "acme"}}, second.Scopes)
		assert.Empty(t, second.NextPageToken)
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
