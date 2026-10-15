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

		_, err := ec.List(ctx, pc, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a Gitea ProviderConfig")
	})

	t.Run("rejects invalid tokens and phases", func(t *testing.T) {
		ec := &externalClient{client: testutil.NoopClient{}}
		pc := &v1beta1.ProviderConfig{}

		for _, token := range []string{"abc", "bogus:1"} {
			_, err := ec.List(ctx, pc, token)
			require.Error(t, err, "token %q", token)
		}
	})

	t.Run("drains repos then orgs", func(t *testing.T) {
		srv := webhookTestServer(t)
		ec, pc := webhookTestSetup(t, srv.URL)

		first, err := ec.List(ctx, pc, "")
		require.NoError(t, err)
		assert.Equal(t, []string{"acme/r1/9"}, first.ExternalNames)
		assert.Equal(t, "orgs:1", first.NextPageToken)

		second, err := ec.List(ctx, pc, first.NextPageToken)
		require.NoError(t, err)
		assert.Equal(t, []string{"acme/4"}, second.ExternalNames)
		assert.Empty(t, second.NextPageToken)
	})
}
