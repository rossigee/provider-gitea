package repositorysecret

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v2 "github.com/rossigee/provider-gitea/apis/repositorysecret/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestList(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects non-Gitea ProviderConfig", func(t *testing.T) {
		ec := &externalClient{client: testutil.NoopClient{}}

		var pc resource.ProviderConfig

		_, err := ec.List(ctx, pc, resource.ExternalListScope{"repository": "acme/r1"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a Gitea ProviderConfig")
	})

	t.Run("requires a repository scope", func(t *testing.T) {
		ec := &externalClient{client: testutil.NoopClient{}}
		pc := &v1beta1.ProviderConfig{}

		_, err := ec.List(ctx, pc, nil, "")
		require.ErrorIs(t, err, resource.ErrScopeRequired)
	})

	t.Run("lists members of the scoped repository", func(t *testing.T) {
		srv := testutil.DiscoveryServer(t, []string{"acme/r1", "acme/r2"},
			func(w http.ResponseWriter, r *http.Request, owner, repo string) bool {
				if owner != "acme" || repo != "r1" {
					w.WriteHeader(http.StatusNotFound)
					return true
				}

				if !strings.HasSuffix(r.URL.Path, "/actions/secrets") {
					return false
				}

				_, _ = fmt.Fprint(w, `[{"name":"TOKEN"}]`)
				return true
			})
		kube, pc := testutil.DiscoveryKube(t, srv.URL)
		ec := &externalClient{client: testutil.NoopClient{}, kube: kube}

		result, err := ec.List(ctx, pc, resource.ExternalListScope{"repository": "acme/r1"}, "")
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "TOKEN", result.Items[0].ExternalName)
		assert.Equal(t, resource.ExternalListScope{"repository": "acme/r1"}, result.Items[0].Scope)
	})

	t.Run("lists repositories as scopes", func(t *testing.T) {
		srv := testutil.DiscoveryServer(t, []string{"acme/r1", "acme/r2"},
			func(w http.ResponseWriter, r *http.Request, owner, repo string) bool {
				return false
			})
		kube, pc := testutil.DiscoveryKube(t, srv.URL)
		ec := &externalClient{client: testutil.NoopClient{}, kube: kube}

		result, err := ec.ListScopes(ctx, pc, nil, "")
		require.NoError(t, err)
		assert.ElementsMatch(t, []resource.ExternalListScope{
			{"repository": "acme/r1"},
			{"repository": "acme/r2"},
		}, result.Scopes)
		assert.Empty(t, result.NextPageToken)
	})

	t.Run("rejects invalid page tokens when listing scopes", func(t *testing.T) {
		ec := &externalClient{client: testutil.NoopClient{}}
		pc := &v1beta1.ProviderConfig{}

		for _, token := range []string{"abc", "0", "bogus:1"} {
			_, err := ec.ListScopes(ctx, pc, nil, token)
			require.Error(t, err, "token %q", token)
		}
	})

	t.Run("propagates item listing errors", func(t *testing.T) {
		srv := testutil.DiscoveryServer(t, []string{"acme/r1"},
			func(w http.ResponseWriter, r *http.Request, owner, repo string) bool {
				w.WriteHeader(http.StatusInternalServerError)
				return true
			})
		kube, pc := testutil.DiscoveryKube(t, srv.URL)
		ec := &externalClient{client: testutil.NoopClient{}, kube: kube}

		_, err := ec.List(ctx, pc, resource.ExternalListScope{"repository": "acme/r1"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list repository secrets")
	})
}

func TestObserve(t *testing.T) {
	ctx := context.Background()

	newKube := func(t *testing.T, baseURL string, extra ...runtime.Object) (client.Client, *v1beta1.ProviderConfig) {
		t.Helper()
		token := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gitea-creds"},
			Data:       map[string][]byte{"token": []byte("test-token")},
		}
		objs := append([]runtime.Object{token}, extra...)

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

		return fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).Build(), pc
	}

	cr := &v2.RepositorySecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Spec: v2.RepositorySecretSpec{
			ForProvider: v2.RepositorySecretParameters{
				Repository: "acme/r1",
				SecretName: "TOKEN",
				ValueSecretRef: xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "app-secrets", Namespace: "default"},
					Key:             "token",
				},
			},
		},
	}
	meta.SetExternalName(cr, "TOKEN")

	newCR := func() *v2.RepositorySecret {
		c := cr.DeepCopy()
		return c
	}

	newServer := func(t *testing.T, status int, body string, captured *map[string]string) *httptest.Server {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodPut && captured != nil {
				b, _ := io.ReadAll(r.Body)
				(*captured)["body"] = string(b)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{}`))
				return
			}
			if r.URL.Path != "/api/v1/repos/acme/r1/actions/secrets/TOKEN" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return srv
	}

	t.Run("404 maps to resource-does-not-exist", func(t *testing.T) {
		srv := newServer(t, http.StatusNotFound, `{"message":"not found"}`, nil)
		kube, pc := newKube(t, srv.URL)
		conn, err := clients.NewClient(ctx, pc, kube)
		require.NoError(t, err)

		ec := &externalClient{client: conn, kube: kube}
		obs, err := ec.Observe(ctx, newCR())
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	appSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "app-secrets"},
		Data:       map[string][]byte{"token": []byte("s3cr3t")},
	}

	t.Run("existing secret without a recorded fingerprint is not up to date", func(t *testing.T) {
		srv := newServer(t, http.StatusOK, `{"name":"TOKEN","created_at":"2026-11-22T00:00:00Z"}`, nil)
		kube, pc := newKube(t, srv.URL, appSecret.DeepCopy())
		conn, err := clients.NewClient(ctx, pc, kube)
		require.NoError(t, err)

		ec := &externalClient{client: conn, kube: kube}
		obj := newCR()
		obs, err := ec.Observe(ctx, obj)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
		require.NotNil(t, obj.Status.AtProvider.SecretName)
		assert.Equal(t, "TOKEN", *obj.Status.AtProvider.SecretName)
	})

	t.Run("matching fingerprint is up to date and keeps the recorded fingerprint", func(t *testing.T) {
		srv := newServer(t, http.StatusOK, `{"name":"TOKEN","created_at":"2026-11-22T00:00:00Z"}`, nil)
		kube, pc := newKube(t, srv.URL, appSecret.DeepCopy())
		conn, err := clients.NewClient(ctx, pc, kube)
		require.NoError(t, err)

		ec := &externalClient{client: conn, kube: kube}
		obj := newCR()
		_, fingerprint, err := ec.resolveSecretValue(ctx, obj)
		require.NoError(t, err)
		obj.Status.AtProvider.ValueFingerprint = &fingerprint

		obs, err := ec.Observe(ctx, obj)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
		require.NotNil(t, obj.Status.AtProvider.ValueFingerprint)
		assert.Equal(t, fingerprint, *obj.Status.AtProvider.ValueFingerprint)
	})

	t.Run("changed source secret makes the resource out of date", func(t *testing.T) {
		srv := newServer(t, http.StatusOK, `{"name":"TOKEN","created_at":"2026-11-22T00:00:00Z"}`, nil)
		kube, pc := newKube(t, srv.URL, appSecret.DeepCopy())
		conn, err := clients.NewClient(ctx, pc, kube)
		require.NoError(t, err)

		ec := &externalClient{client: conn, kube: kube}
		obj := newCR()
		stale := "default/app-secrets/token@stale-version"
		obj.Status.AtProvider.ValueFingerprint = &stale

		obs, err := ec.Observe(ctx, obj)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})

	t.Run("non-404 error mentioning not found is not treated as missing", func(t *testing.T) {
		srv := newServer(t, http.StatusInternalServerError, `repository not found on backend`, nil)
		kube, pc := newKube(t, srv.URL, appSecret.DeepCopy())
		conn, err := clients.NewClient(ctx, pc, kube)
		require.NoError(t, err)

		ec := &externalClient{client: conn, kube: kube}
		obs, err := ec.Observe(ctx, newCR())
		require.Error(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("create resolves the value from valueSecretRef and records its fingerprint", func(t *testing.T) {
		captured := map[string]string{}
		srv := newServer(t, http.StatusOK, `{}`, &captured)
		kube, pc := newKube(t, srv.URL, appSecret.DeepCopy())
		conn, err := clients.NewClient(ctx, pc, kube)
		require.NoError(t, err)

		ec := &externalClient{client: conn, kube: kube}
		obj := newCR()
		_, err = ec.Create(ctx, obj)
		require.NoError(t, err)
		assert.Equal(t, `{"data":"s3cr3t"}`, captured["body"])
		require.NotNil(t, obj.Status.AtProvider.ValueFingerprint)
		assert.True(t, strings.HasPrefix(*obj.Status.AtProvider.ValueFingerprint, "default/app-secrets/token@"))
	})

	t.Run("update records the fingerprint of the value written", func(t *testing.T) {
		captured := map[string]string{}
		srv := newServer(t, http.StatusOK, `{}`, &captured)
		kube, pc := newKube(t, srv.URL, appSecret.DeepCopy())
		conn, err := clients.NewClient(ctx, pc, kube)
		require.NoError(t, err)

		ec := &externalClient{client: conn, kube: kube}
		obj := newCR()
		_, err = ec.Update(ctx, obj)
		require.NoError(t, err)
		assert.Equal(t, `{"data":"s3cr3t"}`, captured["body"])
		require.NotNil(t, obj.Status.AtProvider.ValueFingerprint)
	})

	t.Run("deleted resource is not blocked by a missing source secret", func(t *testing.T) {
		srv := newServer(t, http.StatusOK, `{"name":"TOKEN","created_at":"2026-11-22T00:00:00Z"}`, nil)
		kube, pc := newKube(t, srv.URL)
		conn, err := clients.NewClient(ctx, pc, kube)
		require.NoError(t, err)

		ec := &externalClient{client: conn, kube: kube}
		obj := newCR()
		now := metav1.Now()
		obj.SetDeletionTimestamp(&now)

		obs, err := ec.Observe(ctx, obj)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
	})

	t.Run("secret reference in another namespace is rejected", func(t *testing.T) {
		srv := newServer(t, http.StatusOK, `{}`, nil)
		other := appSecret.DeepCopy()
		other.Namespace = "tenant-b"
		kube, pc := newKube(t, srv.URL, other)
		conn, err := clients.NewClient(ctx, pc, kube)
		require.NoError(t, err)

		ec := &externalClient{client: conn, kube: kube}
		obj := newCR()
		obj.Spec.ForProvider.ValueSecretRef.Namespace = "tenant-b"
		_, err = ec.Create(ctx, obj)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must match the resource namespace")
	})

	t.Run("create fails without the referenced secret", func(t *testing.T) {
		srv := newServer(t, http.StatusOK, `{}`, nil)
		kube, pc := newKube(t, srv.URL)
		conn, err := clients.NewClient(ctx, pc, kube)
		require.NoError(t, err)

		ec := &externalClient{client: conn, kube: kube}
		_, err = ec.Create(ctx, newCR())
		require.Error(t, err)
		assert.Contains(t, err.Error(), errResolveSecret)
	})
}
