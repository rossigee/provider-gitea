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

package pullrequest

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v2 "github.com/rossigee/provider-gitea/apis/pullrequest/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockPRClient struct {
	testutil.NoopClient
	getFn    func(ctx context.Context, owner, repo string, number int64) (*clients.PullRequest, error)
	createFn func(ctx context.Context, owner, repo string, req *clients.CreatePullRequestOptions) (*clients.PullRequest, error)
	updateFn func(ctx context.Context, owner, repo string, number int64, req *clients.UpdatePullRequestOptions) (*clients.PullRequest, error)
	deleteFn func(ctx context.Context, owner, repo string, number int64) error
}

func (m *mockPRClient) GetPullRequest(ctx context.Context, owner, repo string, number int64) (*clients.PullRequest, error) {
	if m.getFn != nil {
		return m.getFn(ctx, owner, repo, number)
	}
	return nil, nil
}
func (m *mockPRClient) CreatePullRequest(ctx context.Context, owner, repo string, req *clients.CreatePullRequestOptions) (*clients.PullRequest, error) {
	if m.createFn != nil {
		return m.createFn(ctx, owner, repo, req)
	}
	return nil, nil
}
func (m *mockPRClient) UpdatePullRequest(ctx context.Context, owner, repo string, number int64, req *clients.UpdatePullRequestOptions) (*clients.PullRequest, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, owner, repo, number, req)
	}
	return nil, nil
}
func (m *mockPRClient) DeletePullRequest(ctx context.Context, owner, repo string, number int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, owner, repo, number)
	}
	return nil
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockPRClient{}}
		cr := &v2.PullRequest{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pr"}}
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockPRClient{
			getFn: func(ctx context.Context, owner, repo string, number int64) (*clients.PullRequest, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.PullRequest{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pr"}}
		meta.SetExternalName(cr, "acme/app/7")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("matching PR is up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockPRClient{
			getFn: func(ctx context.Context, owner, repo string, number int64) (*clients.PullRequest, error) {
				assert.Equal(t, int64(7), number)
				return &clients.PullRequest{ID: 100, Number: 7, Title: "Feature", State: "open"}, nil
			},
		}}
		cr := &v2.PullRequest{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pr"}}
		cr.Spec.ForProvider.Title = "Feature"
		meta.SetExternalName(cr, "acme/app/7")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("title drift is not up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockPRClient{
			getFn: func(ctx context.Context, owner, repo string, number int64) (*clients.PullRequest, error) {
				return &clients.PullRequest{ID: 100, Number: 7, Title: "Old", State: "open"}, nil
			},
		}}
		cr := &v2.PullRequest{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pr"}}
		cr.Spec.ForProvider.Title = "New"
		meta.SetExternalName(cr, "acme/app/7")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	ec := &externalClient{client: &mockPRClient{
		createFn: func(ctx context.Context, owner, repo string, req *clients.CreatePullRequestOptions) (*clients.PullRequest, error) {
			assert.Equal(t, "acme", owner)
			assert.Equal(t, "feature", req.Head)
			return &clients.PullRequest{ID: 100, Number: 7, Title: "Feature"}, nil
		},
	}}
	cr := &v2.PullRequest{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pr"}}
	cr.Spec.ForProvider.Owner = "acme"
	cr.Spec.ForProvider.Repository = "app"
	cr.Spec.ForProvider.Title = "Feature"
	cr.Spec.ForProvider.Head = "feature"
	cr.Spec.ForProvider.Base = "master"
	_, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "acme/app/7", meta.GetExternalName(cr))
}

func TestDelete(t *testing.T) {
	t.Run("missing PR is success", func(t *testing.T) {
		ec := &externalClient{client: &mockPRClient{
			getFn: func(ctx context.Context, owner, repo string, number int64) (*clients.PullRequest, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.PullRequest{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pr"}}
		meta.SetExternalName(cr, "acme/app/7")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("existing PR is deleted", func(t *testing.T) {
		var deleted bool
		ec := &externalClient{client: &mockPRClient{
			getFn: func(ctx context.Context, owner, repo string, number int64) (*clients.PullRequest, error) {
				return &clients.PullRequest{ID: 100, Number: 7}, nil
			},
			deleteFn: func(ctx context.Context, owner, repo string, number int64) error {
				deleted = true
				return nil
			},
		}}
		cr := &v2.PullRequest{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pr"}}
		meta.SetExternalName(cr, "acme/app/7")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}
