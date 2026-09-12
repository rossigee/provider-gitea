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

package release

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v2 "github.com/rossigee/provider-gitea/apis/release/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockReleaseClient struct {
	testutil.NoopClient
	getFn    func(ctx context.Context, owner, repo string, id int64) (*clients.Release, error)
	createFn func(ctx context.Context, owner, repo string, req *clients.CreateReleaseOptions) (*clients.Release, error)
	updateFn func(ctx context.Context, owner, repo string, id int64, req *clients.UpdateReleaseOptions) (*clients.Release, error)
	deleteFn func(ctx context.Context, owner, repo string, id int64) error
}

func (m *mockReleaseClient) GetRelease(ctx context.Context, owner, repo string, id int64) (*clients.Release, error) {
	if m.getFn != nil {
		return m.getFn(ctx, owner, repo, id)
	}
	return nil, nil
}
func (m *mockReleaseClient) CreateRelease(ctx context.Context, owner, repo string, req *clients.CreateReleaseOptions) (*clients.Release, error) {
	if m.createFn != nil {
		return m.createFn(ctx, owner, repo, req)
	}
	return nil, nil
}
func (m *mockReleaseClient) UpdateRelease(ctx context.Context, owner, repo string, id int64, req *clients.UpdateReleaseOptions) (*clients.Release, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, owner, repo, id, req)
	}
	return nil, nil
}
func (m *mockReleaseClient) DeleteRelease(ctx context.Context, owner, repo string, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, owner, repo, id)
	}
	return nil
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockReleaseClient{}}
		cr := &v2.Release{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "rel"}}
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockReleaseClient{
			getFn: func(ctx context.Context, owner, repo string, id int64) (*clients.Release, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.Release{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "rel"}}
		meta.SetExternalName(cr, "acme/app/11")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("matching release is up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockReleaseClient{
			getFn: func(ctx context.Context, owner, repo string, id int64) (*clients.Release, error) {
				assert.Equal(t, int64(11), id)
				return &clients.Release{ID: 11, TagName: "v1.0.0", Name: "First"}, nil
			},
		}}
		cr := &v2.Release{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "rel"}}
		cr.Spec.ForProvider.TagName = "v1.0.0"
		meta.SetExternalName(cr, "acme/app/11")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("tag drift is not up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockReleaseClient{
			getFn: func(ctx context.Context, owner, repo string, id int64) (*clients.Release, error) {
				return &clients.Release{ID: 11, TagName: "v1.0.0"}, nil
			},
		}}
		cr := &v2.Release{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "rel"}}
		cr.Spec.ForProvider.TagName = "v2.0.0"
		meta.SetExternalName(cr, "acme/app/11")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	ec := &externalClient{client: &mockReleaseClient{
		createFn: func(ctx context.Context, owner, repo string, req *clients.CreateReleaseOptions) (*clients.Release, error) {
			assert.Equal(t, "acme", owner)
			assert.Equal(t, "v1.0.0", req.TagName)
			return &clients.Release{ID: 11, TagName: "v1.0.0"}, nil
		},
	}}
	cr := &v2.Release{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "rel"}}
	cr.Spec.ForProvider.Owner = "acme"
	cr.Spec.ForProvider.Repository = "app"
	cr.Spec.ForProvider.TagName = "v1.0.0"
	_, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "acme/app/11", meta.GetExternalName(cr))
}

func TestDelete(t *testing.T) {
	t.Run("missing release is success", func(t *testing.T) {
		ec := &externalClient{client: &mockReleaseClient{
			getFn: func(ctx context.Context, owner, repo string, id int64) (*clients.Release, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.Release{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "rel"}}
		meta.SetExternalName(cr, "acme/app/11")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("existing release is deleted", func(t *testing.T) {
		var deleted bool
		ec := &externalClient{client: &mockReleaseClient{
			getFn: func(ctx context.Context, owner, repo string, id int64) (*clients.Release, error) {
				return &clients.Release{ID: 11}, nil
			},
			deleteFn: func(ctx context.Context, owner, repo string, id int64) error {
				deleted = true
				return nil
			},
		}}
		cr := &v2.Release{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "rel"}}
		meta.SetExternalName(cr, "acme/app/11")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}
