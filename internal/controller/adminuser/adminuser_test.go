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

package adminuser

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v2 "github.com/rossigee/provider-gitea/apis/adminuser/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type mockAdminClient struct {
	testutil.NoopClient
	getFn    func(ctx context.Context, username string) (*clients.AdminUser, error)
	createFn func(ctx context.Context, req *clients.CreateAdminUserRequest) (*clients.AdminUser, error)
	updateFn func(ctx context.Context, username string, req *clients.UpdateAdminUserRequest) (*clients.AdminUser, error)
	deleteFn func(ctx context.Context, username string) error
}

func (m *mockAdminClient) GetAdminUser(ctx context.Context, username string) (*clients.AdminUser, error) {
	if m.getFn != nil {
		return m.getFn(ctx, username)
	}
	return nil, nil
}
func (m *mockAdminClient) CreateAdminUser(ctx context.Context, req *clients.CreateAdminUserRequest) (*clients.AdminUser, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return nil, nil
}
func (m *mockAdminClient) UpdateAdminUser(ctx context.Context, username string, req *clients.UpdateAdminUserRequest) (*clients.AdminUser, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, username, req)
	}
	return nil, nil
}
func (m *mockAdminClient) DeleteAdminUser(ctx context.Context, username string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, username)
	}
	return nil
}

func testExternal(username string) *externalClient {
	scheme := fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pw"},
		Data:       map[string][]byte{"password": []byte("s3cret")},
	}).Build()
	return &externalClient{kube: scheme, client: &mockAdminClient{}}
}

func TestObserve(t *testing.T) {
	t.Run("404 returns not exists", func(t *testing.T) {
		ec := testExternal("svc")
		ec.client = &mockAdminClient{
			getFn: func(ctx context.Context, username string) (*clients.AdminUser, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}
		cr := &v2.AdminUser{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "svc"}}
		cr.Spec.ForProvider.Username = "svc"
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("matching user is up to date", func(t *testing.T) {
		ec := testExternal("svc")
		ec.client = &mockAdminClient{
			getFn: func(ctx context.Context, username string) (*clients.AdminUser, error) {
				assert.Equal(t, "svc", username)
				return &clients.AdminUser{ID: 9, Username: "svc", Email: "svc@example.com"}, nil
			},
		}
		cr := &v2.AdminUser{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "svc"}}
		cr.Spec.ForProvider.Username = "svc"
		cr.Spec.ForProvider.Email = "svc@example.com"
		meta.SetExternalName(cr, "svc")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("email drift is not up to date", func(t *testing.T) {
		ec := testExternal("svc")
		ec.client = &mockAdminClient{
			getFn: func(ctx context.Context, username string) (*clients.AdminUser, error) {
				return &clients.AdminUser{ID: 9, Username: "svc", Email: "old@example.com"}, nil
			},
		}
		cr := &v2.AdminUser{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "svc"}}
		cr.Spec.ForProvider.Username = "svc"
		cr.Spec.ForProvider.Email = "new@example.com"
		meta.SetExternalName(cr, "svc")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	ec := testExternal("svc")
	ec.client = &mockAdminClient{
		createFn: func(ctx context.Context, req *clients.CreateAdminUserRequest) (*clients.AdminUser, error) {
			assert.Equal(t, "svc", req.Username)
			assert.Equal(t, "s3cret", req.Password)
			return &clients.AdminUser{ID: 9, Username: "svc"}, nil
		},
	}
	cr := &v2.AdminUser{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "svc"}}
	cr.Spec.ForProvider.Username = "svc"
	cr.Spec.ForProvider.Email = "svc@example.com"
	cr.Spec.ForProvider.PasswordSecretRef.Name = "pw"
	cr.Spec.ForProvider.PasswordSecretRef.Key = "password"
	_, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "svc", meta.GetExternalName(cr))
}

func TestDelete(t *testing.T) {
	t.Run("missing user is success", func(t *testing.T) {
		ec := testExternal("svc")
		ec.client = &mockAdminClient{
			getFn: func(ctx context.Context, username string) (*clients.AdminUser, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}
		cr := &v2.AdminUser{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "svc"}}
		cr.Spec.ForProvider.Username = "svc"
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("existing user is deleted", func(t *testing.T) {
		var deleted bool
		ec := testExternal("svc")
		ec.client = &mockAdminClient{
			getFn: func(ctx context.Context, username string) (*clients.AdminUser, error) {
				return &clients.AdminUser{ID: 9, Username: "svc"}, nil
			},
			deleteFn: func(ctx context.Context, username string) error {
				deleted = true
				return nil
			},
		}
		cr := &v2.AdminUser{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "svc"}}
		cr.Spec.ForProvider.Username = "svc"
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}
