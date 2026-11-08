package branchprotection

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

				if !strings.HasSuffix(r.URL.Path, "/branch_protections") {
					return false
				}

				_, _ = fmt.Fprint(w, `[{"rule_name":"main"}]`)
				return true
			})
		kube, pc := testutil.DiscoveryKube(t, srv.URL)
		ec := &externalClient{client: testutil.NoopClient{}, kube: kube}

		result, err := ec.List(ctx, pc, resource.ExternalListScope{"repository": "acme/r1"}, "")
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "acme/r1:main", result.Items[0].ExternalName)
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
		assert.Contains(t, err.Error(), "failed to list branch protections")
	})
}
