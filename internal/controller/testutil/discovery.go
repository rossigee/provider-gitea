/*
Copyright 2026 The Crossplane Authors.

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

package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// RepoNames builds n owner/name pairs for tests.
func RepoNames(owner string, n int) []string {
	names := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		names = append(names, fmt.Sprintf("%s/repo-%d", owner, i))
	}

	return names
}

// DiscoveryServer emulates the Gitea endpoints used by ExternalLister tests:
// GET /api/v1/user/repos (paginated owner/name pairs) and per-repository item
// endpoints dispatched to itemHandler with the parsed owner and repo.
// itemHandler reports whether it handled the request; unhandled requests get
// 404. The test Authorization header is asserted centrally.
func DiscoveryServer(t *testing.T, repos []string, itemHandler func(w http.ResponseWriter, r *http.Request, owner, repo string) bool) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v1/user/repos" {
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

			if page < 1 {
				page = 1
			}

			start := (page - 1) * limit
			if start > len(repos) {
				start = len(repos)
			}

			end := start + limit
			if end > len(repos) {
				end = len(repos)
			}

			_, _ = fmt.Fprint(w, "[")

			for i, n := range repos[start:end] {
				if i > 0 {
					_, _ = fmt.Fprint(w, ",")
				}

				_, _ = fmt.Fprintf(w, `{"full_name":%q}`, n)
			}

			_, _ = fmt.Fprint(w, "]")

			return
		}

		rest, ok := strings.CutPrefix(r.URL.Path, "/api/v1/repos/")
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		parts := strings.SplitN(rest, "/", 3)
		if len(parts) != 3 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if !itemHandler(w, r, parts[0], parts[1]) {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	return srv
}

// DiscoveryKube returns a fake kube client holding a token secret plus a
// ProviderConfig pointing at baseURL, for ExternalLister tests.
func DiscoveryKube(t *testing.T, baseURL string) (client.Client, *v1beta1.ProviderConfig) {
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

	return fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(secret).Build(), pc
}
