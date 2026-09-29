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

		_, err := ec.List(ctx, pc, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a Gitea ProviderConfig")
	})

	t.Run("rejects invalid page tokens", func(t *testing.T) {
		ec := &externalClient{client: testutil.NoopClient{}}
		pc := &v1beta1.ProviderConfig{}

		for _, token := range []string{"abc", "0"} {
			_, err := ec.List(ctx, pc, token)
			require.Error(t, err, "token %q", token)
		}
	})

	t.Run("discovers rule names across repositories", func(t *testing.T) {
		srv := testutil.DiscoveryServer(t, []string{"acme/r1", "acme/r2"},
			func(w http.ResponseWriter, r *http.Request, owner, repo string) bool {
				if !strings.HasSuffix(r.URL.Path, "/branch_protections") {
					return false
				}

				_, _ = fmt.Fprint(w, `[{"rule_name":"main"}]`)
				return true
			})
		kube, pc := testutil.DiscoveryKube(t, srv.URL)
		ec := &externalClient{client: testutil.NoopClient{}, kube: kube}

		result, err := ec.List(ctx, pc, "")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"acme/r1:main", "acme/r2:main"}, result.ExternalNames)
		assert.Empty(t, result.NextPageToken)
	})

	t.Run("propagates item listing errors", func(t *testing.T) {
		srv := testutil.DiscoveryServer(t, []string{"acme/r1"},
			func(w http.ResponseWriter, r *http.Request, owner, repo string) bool {
				w.WriteHeader(http.StatusInternalServerError)
				return true
			})
		kube, pc := testutil.DiscoveryKube(t, srv.URL)
		ec := &externalClient{client: testutil.NoopClient{}, kube: kube}

		_, err := ec.List(ctx, pc, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list branch protections")
	})
}
