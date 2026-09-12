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

package userkey

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v2 "github.com/rossigee/provider-gitea/apis/userkey/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockUserKeyClient struct {
	testutil.NoopClient
	getFn    func(ctx context.Context, username string, id int64) (*clients.UserKey, error)
	createFn func(ctx context.Context, username string, req *clients.CreateUserKeyRequest) (*clients.UserKey, error)
	updateFn func(ctx context.Context, username string, id int64, req *clients.UpdateUserKeyRequest) (*clients.UserKey, error)
	deleteFn func(ctx context.Context, username string, id int64) error
}

func (m *mockUserKeyClient) GetUserKey(ctx context.Context, username string, id int64) (*clients.UserKey, error) {
	if m.getFn != nil {
		return m.getFn(ctx, username, id)
	}
	return nil, nil
}
func (m *mockUserKeyClient) CreateUserKey(ctx context.Context, username string, req *clients.CreateUserKeyRequest) (*clients.UserKey, error) {
	if m.createFn != nil {
		return m.createFn(ctx, username, req)
	}
	return nil, nil
}
func (m *mockUserKeyClient) UpdateUserKey(ctx context.Context, username string, id int64, req *clients.UpdateUserKeyRequest) (*clients.UserKey, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, username, id, req)
	}
	return nil, nil
}
func (m *mockUserKeyClient) DeleteUserKey(ctx context.Context, username string, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, username, id)
	}
	return nil
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockUserKeyClient{}}
		cr := &v2.UserKey{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "k"}}
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockUserKeyClient{
			getFn: func(ctx context.Context, username string, id int64) (*clients.UserKey, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.UserKey{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "k"}}
		meta.SetExternalName(cr, "alice/42")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("matching key is up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockUserKeyClient{
			getFn: func(ctx context.Context, username string, id int64) (*clients.UserKey, error) {
				assert.Equal(t, "alice", username)
				assert.Equal(t, int64(42), id)
				return &clients.UserKey{ID: 42, Title: "laptop", Key: "ssh-ed25519 AAAA"}, nil
			},
		}}
		cr := &v2.UserKey{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "k"}}
		cr.Spec.ForProvider.Title = "laptop"
		cr.Spec.ForProvider.Key = "ssh-ed25519 AAAA"
		meta.SetExternalName(cr, "alice/42")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("key material drift is not up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockUserKeyClient{
			getFn: func(ctx context.Context, username string, id int64) (*clients.UserKey, error) {
				return &clients.UserKey{ID: 42, Title: "laptop", Key: "ssh-rsa AAAA"}, nil
			},
		}}
		cr := &v2.UserKey{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "k"}}
		cr.Spec.ForProvider.Title = "laptop"
		cr.Spec.ForProvider.Key = "ssh-ed25519 AAAA"
		meta.SetExternalName(cr, "alice/42")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	ec := &externalClient{client: &mockUserKeyClient{
		createFn: func(ctx context.Context, username string, req *clients.CreateUserKeyRequest) (*clients.UserKey, error) {
			assert.Equal(t, "alice", username)
			assert.Equal(t, "laptop", req.Title)
			return &clients.UserKey{ID: 42, Title: "laptop", Key: "ssh-ed25519 AAAA"}, nil
		},
	}}
	cr := &v2.UserKey{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "k"}}
	cr.Spec.ForProvider.Username = "alice"
	cr.Spec.ForProvider.Title = "laptop"
	cr.Spec.ForProvider.Key = "ssh-ed25519 AAAA"
	_, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "alice/42", meta.GetExternalName(cr))
}

func TestDelete(t *testing.T) {
	t.Run("missing key is success", func(t *testing.T) {
		ec := &externalClient{client: &mockUserKeyClient{
			getFn: func(ctx context.Context, username string, id int64) (*clients.UserKey, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.UserKey{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "k"}}
		meta.SetExternalName(cr, "alice/42")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("existing key is deleted", func(t *testing.T) {
		var deleted bool
		ec := &externalClient{client: &mockUserKeyClient{
			getFn: func(ctx context.Context, username string, id int64) (*clients.UserKey, error) {
				return &clients.UserKey{ID: 42}, nil
			},
			deleteFn: func(ctx context.Context, username string, id int64) error {
				deleted = true
				return nil
			},
		}}
		cr := &v2.UserKey{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "k"}}
		meta.SetExternalName(cr, "alice/42")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}
