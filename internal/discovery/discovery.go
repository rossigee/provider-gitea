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

// Package discovery holds shared helpers for ExternalLister implementations
// backing Crossplane resource discovery (DiscoveryReport). The page token
// addresses the outer (repository or organization) page; inner per-parent
// listings are unpaginated.
package discovery

import (
	"context"
	"strconv"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PageSize is the page size used for outer paginated listings.
const PageSize = 50

// ParsePageToken parses a discovery page token. Empty means the first page.
func ParsePageToken(token string) (int, error) {
	if token == "" {
		return 1, nil
	}

	page, err := strconv.Atoi(token)
	if err != nil || page < 1 {
		return 0, errors.Errorf("invalid page token %q: must be a positive page number", token)
	}

	return page, nil
}

// NextPageToken returns the token for the following page when a full page of
// parents was seen, or empty when the current page is the last one.
func NextPageToken(parentCount, page int) string {
	if parentCount == PageSize {
		return strconv.Itoa(page + 1)
	}

	return ""
}

// NewClient builds an API client scoped to the given ProviderConfig.
func NewClient(ctx context.Context, kube client.Client, pc resource.ProviderConfig) (clients.Client, error) {
	cfg, ok := pc.(*v1beta1.ProviderConfig)
	if !ok {
		return nil, errors.New("provider config is not a Gitea ProviderConfig")
	}

	conn, err := clients.NewClient(ctx, cfg, kube)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Gitea client for discovery")
	}

	return conn, nil
}

// ForEachRepository pages the repository list once and invokes fn for every
// repository with a usable owner/name. It returns the number of repositories
// on the page, for NextPageToken computation.
func ForEachRepository(ctx context.Context, conn clients.Client, page int, fn func(owner, repo string) error) (int, error) {
	repos, err := conn.ListRepositories(ctx, page, PageSize)
	if err != nil {
		return 0, errors.Wrap(err, "failed to list repositories")
	}

	for _, r := range repos {
		if r.FullName == "" {
			continue
		}

		parts := splitFullName(r.FullName)
		if len(parts) != 2 {
			continue
		}

		if err := fn(parts[0], parts[1]); err != nil {
			return 0, err
		}
	}

	return len(repos), nil
}

func splitFullName(full string) []string {
	for i := 0; i < len(full); i++ {
		if full[i] == '/' {
			return []string{full[:i], full[i+1:]}
		}
	}

	return []string{full}
}
