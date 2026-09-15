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

package deploytoken

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v2 "github.com/rossigee/provider-gitea/apis/deploytoken/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockDeployTokenClient struct {
	testutil.NoopClient
	getFn    func(ctx context.Context, owner, repo string, id int64) (*clients.DeployToken, error)
	createFn func(ctx context.Context, owner, repo string, req *clients.CreateDeployTokenRequest) (*clients.DeployToken, error)
	deleteFn func(ctx context.Context, owner, repo string, id int64) error
}

func (m *mockDeployTokenClient) GetDeployToken(ctx context.Context, owner, repo string, id int64) (*clients.DeployToken, error) {
	if m.getFn != nil {
		return m.getFn(ctx, owner, repo, id)
	}
	return nil, nil
}
func (m *mockDeployTokenClient) CreateDeployToken(ctx context.Context, owner, repo string, req *clients.CreateDeployTokenRequest) (*clients.DeployToken, error) {
	if m.createFn != nil {
		return m.createFn(ctx, owner, repo, req)
	}
	return nil, nil
}
func (m *mockDeployTokenClient) DeleteDeployToken(ctx context.Context, owner, repo string, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, owner, repo, id)
	}
	return nil
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockDeployTokenClient{}}
		cr := &v2.DeployToken{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "token"}}
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockDeployTokenClient{
			getFn: func(ctx context.Context, owner, repo string, id int64) (*clients.DeployToken, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.DeployToken{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "token"}}
		meta.SetExternalName(cr, "acme/app/5")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("matching token is up to date", func(t *testing.T) {
		readOnlyTrue := true
		ec := &externalClient{client: &mockDeployTokenClient{
			getFn: func(ctx context.Context, owner, repo string, id int64) (*clients.DeployToken, error) {
				assert.Equal(t, int64(5), id)
				return &clients.DeployToken{ID: 5, Title: "ci", ReadOnly: true}, nil
			},
		}}
		cr := &v2.DeployToken{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "token"}}
		cr.Spec.ForProvider.Title = "ci"
		cr.Spec.ForProvider.ReadOnly = &readOnlyTrue
		meta.SetExternalName(cr, "acme/app/5")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("title drift makes not up to date", func(t *testing.T) {
		readOnlyTrue := true
		ec := &externalClient{client: &mockDeployTokenClient{
			getFn: func(ctx context.Context, owner, repo string, id int64) (*clients.DeployToken, error) {
				return &clients.DeployToken{ID: 5, Title: "old-ci", ReadOnly: true}, nil
			},
		}}
		cr := &v2.DeployToken{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "token"}}
		cr.Spec.ForProvider.Title = "new-ci"
		cr.Spec.ForProvider.ReadOnly = &readOnlyTrue
		meta.SetExternalName(cr, "acme/app/5")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})

	t.Run("readOnly drift makes not up to date", func(t *testing.T) {
		readOnlyFalse := false
		ec := &externalClient{client: &mockDeployTokenClient{
			getFn: func(ctx context.Context, owner, repo string, id int64) (*clients.DeployToken, error) {
				return &clients.DeployToken{ID: 5, Title: "ci", ReadOnly: true}, nil
			},
		}}
		cr := &v2.DeployToken{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "token"}}
		cr.Spec.ForProvider.Title = "ci"
		cr.Spec.ForProvider.ReadOnly = &readOnlyFalse
		meta.SetExternalName(cr, "acme/app/5")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	readOnlyTrue := true
	ec := &externalClient{client: &mockDeployTokenClient{
		createFn: func(ctx context.Context, owner, repo string, req *clients.CreateDeployTokenRequest) (*clients.DeployToken, error) {
			assert.Equal(t, "acme", owner)
			assert.Equal(t, "app", repo)
			assert.Equal(t, "ci", req.Title)
			assert.Equal(t, true, req.ReadOnly)
			return &clients.DeployToken{ID: 5, Title: "ci", Token: "gdt_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, nil
		},
	}}
	cr := &v2.DeployToken{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "token"}}
	cr.Spec.ForProvider.Owner = "acme"
	cr.Spec.ForProvider.Repository = "app"
	cr.Spec.ForProvider.Title = "ci"
	cr.Spec.ForProvider.ReadOnly = &readOnlyTrue
	creation, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "acme/app/5", meta.GetExternalName(cr))

	// Verify ConnectionDetails contains the plaintext token
	assert.NotNil(t, creation.ConnectionDetails)
	assert.Equal(t, []byte("gdt_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), creation.ConnectionDetails["token"])
}

func TestDelete(t *testing.T) {
	t.Run("missing token is success", func(t *testing.T) {
		ec := &externalClient{client: &mockDeployTokenClient{
			getFn: func(ctx context.Context, owner, repo string, id int64) (*clients.DeployToken, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.DeployToken{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "token"}}
		meta.SetExternalName(cr, "acme/app/5")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("existing token is deleted", func(t *testing.T) {
		var deleted bool
		ec := &externalClient{client: &mockDeployTokenClient{
			getFn: func(ctx context.Context, owner, repo string, id int64) (*clients.DeployToken, error) {
				return &clients.DeployToken{ID: 5}, nil
			},
			deleteFn: func(ctx context.Context, owner, repo string, id int64) error {
				deleted = true
				return nil
			},
		}}
		cr := &v2.DeployToken{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "token"}}
		meta.SetExternalName(cr, "acme/app/5")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}
