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

package user

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/rossigee/provider-gitea/apis/user/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockUserClient struct {
	testutil.NoopClient
	getUserFn    func(ctx context.Context, name string) (*clients.User, error)
	createUserFn func(ctx context.Context, req *clients.CreateUserRequest) (*clients.User, error)
	updateUserFn func(ctx context.Context, name string, req *clients.UpdateUserRequest) (*clients.User, error)
	deleteUserFn func(ctx context.Context, name string) error
}

func (m *mockUserClient) GetUser(ctx context.Context, name string) (*clients.User, error) {
	if m.getUserFn != nil {
		return m.getUserFn(ctx, name)
	}
	return nil, nil
}
func (m *mockUserClient) CreateUser(ctx context.Context, req *clients.CreateUserRequest) (*clients.User, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, req)
	}
	return nil, nil
}
func (m *mockUserClient) UpdateUser(ctx context.Context, name string, req *clients.UpdateUserRequest) (*clients.User, error) {
	if m.updateUserFn != nil {
		return m.updateUserFn(ctx, name, req)
	}
	return nil, nil
}
func (m *mockUserClient) DeleteUser(ctx context.Context, name string) error {
	if m.deleteUserFn != nil {
		return m.deleteUserFn(ctx, name)
	}
	return nil
}

func strPtr(s string) *string { return &s }

func TestIsUserUpToDate(t *testing.T) {
	mkStr := func(s string) *string { return &s }

	baseUser := &clients.User{
		Email:       "test@example.com",
		FullName:    "Test User",
		Website:     "https://example.com",
		Location:    "Earth",
		Description: "Hello",
	}

	t.Run("matching fields is up to date", func(t *testing.T) {
		cr := &v2.User{
			Spec: v2.UserSpec{
				ForProvider: v2.UserParameters{
					Email:       "test@example.com",
					FullName:    mkStr("Test User"),
					Website:     mkStr("https://example.com"),
					Location:    mkStr("Earth"),
					Description: mkStr("Hello"),
				},
			},
		}
		assert.True(t, isUserUpToDate(cr, baseUser))
	})

	t.Run("email mismatch is not up to date", func(t *testing.T) {
		cr := &v2.User{
			Spec: v2.UserSpec{
				ForProvider: v2.UserParameters{Email: "new@example.com"},
			},
		}
		assert.False(t, isUserUpToDate(cr, baseUser))
	})

	t.Run("full name mismatch is not up to date", func(t *testing.T) {
		cr := &v2.User{
			Spec: v2.UserSpec{
				ForProvider: v2.UserParameters{FullName: strPtr("Different Name")},
			},
		}
		assert.False(t, isUserUpToDate(cr, baseUser))
	})

	t.Run("description mismatch is not up to date", func(t *testing.T) {
		cr := &v2.User{
			Spec: v2.UserSpec{
				ForProvider: v2.UserParameters{Description: strPtr("New bio")},
			},
		}
		assert.False(t, isUserUpToDate(cr, baseUser))
	})

	t.Run("nil optional fields are ignored", func(t *testing.T) {
		cr := &v2.User{
			Spec: v2.UserSpec{
				ForProvider: v2.UserParameters{Email: "test@example.com"},
			},
		}
		assert.True(t, isUserUpToDate(cr, baseUser))
	})
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockUserClient{}}
		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
		}

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{
			client: &mockUserClient{
				getUserFn: func(ctx context.Context, name string) (*clients.User, error) {
					return nil, fmt.Errorf("API request failed with status 404: not found")
				},
			},
		}
		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
		}
		meta.SetExternalName(cr, "test-user")

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("non-404 returns error", func(t *testing.T) {
		ec := &externalClient{
			client: &mockUserClient{
				getUserFn: func(ctx context.Context, name string) (*clients.User, error) {
					return nil, fmt.Errorf("API request failed with status 500: internal error")
				},
			},
		}
		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
		}
		meta.SetExternalName(cr, "test-user")

		_, err := ec.Observe(context.Background(), cr)
		require.Error(t, err)
	})

	t.Run("resource exists updates status", func(t *testing.T) {
		ec := &externalClient{
			client: &mockUserClient{
				getUserFn: func(ctx context.Context, name string) (*clients.User, error) {
					return &clients.User{
						ID:        7,
						Username:  "test-user",
						Email:     "test@example.com",
						IsAdmin:   false,
						LastLogin: "2026-08-01T00:00:00Z",
					}, nil
				},
			},
		}
		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
			Spec: v2.UserSpec{
				ForProvider: v2.UserParameters{
					Username: "test-user",
					Email:    "test@example.com",
				},
			},
		}
		meta.SetExternalName(cr, "test-user")

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
		require.NotNil(t, cr.Status.AtProvider.ID)
		assert.Equal(t, int64(7), *cr.Status.AtProvider.ID)
	})

	t.Run("drift detected returns not up to date", func(t *testing.T) {
		ec := &externalClient{
			client: &mockUserClient{
				getUserFn: func(ctx context.Context, name string) (*clients.User, error) {
					return &clients.User{
						Username:   "test-user",
						Email:      "old@example.com",
						FullName:   "Old Name",
						IsAdmin:    false,
						Restricted: false,
						Active:     true,
					}, nil
				},
			},
		}
		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
			Spec: v2.UserSpec{
				ForProvider: v2.UserParameters{
					Username: "test-user",
					Email:    "new@example.com",
					FullName: strPtr("New Name"),
				},
			},
		}
		meta.SetExternalName(cr, "test-user")

		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	t.Run("creates user with full spec", func(t *testing.T) {
		ec := &externalClient{
			client: &mockUserClient{
				createUserFn: func(ctx context.Context, req *clients.CreateUserRequest) (*clients.User, error) {
					assert.Equal(t, "test-user", req.Username)
					assert.Equal(t, "test@example.com", req.Email)
					assert.Equal(t, "Password123!", req.Password)
					assert.Equal(t, "Test User", req.FullName)
					return &clients.User{
						ID:       5,
						Username: "test-user",
					}, nil
				},
			},
		}

		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
			Spec: v2.UserSpec{
				ForProvider: v2.UserParameters{
					Username: "test-user",
					Email:    "test@example.com",
					Password: "Password123!",
					FullName: strPtr("Test User"),
				},
			},
		}

		_, err := ec.Create(context.Background(), cr)
		require.NoError(t, err)
		assert.Equal(t, "test-user", meta.GetExternalName(cr))
		require.NotNil(t, cr.Status.AtProvider.ID)
		assert.Equal(t, int64(5), *cr.Status.AtProvider.ID)
	})

	t.Run("create failure returns error", func(t *testing.T) {
		ec := &externalClient{
			client: &mockUserClient{
				createUserFn: func(ctx context.Context, req *clients.CreateUserRequest) (*clients.User, error) {
					return nil, fmt.Errorf("API request failed with status 500: internal error")
				},
			},
		}

		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
			Spec: v2.UserSpec{
				ForProvider: v2.UserParameters{
					Username: "test-user",
					Email:    "test@example.com",
					Password: "Password123!",
				},
			},
		}

		_, err := ec.Create(context.Background(), cr)
		require.Error(t, err)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("updates user", func(t *testing.T) {
		updated := false
		ec := &externalClient{
			client: &mockUserClient{
				updateUserFn: func(ctx context.Context, name string, req *clients.UpdateUserRequest) (*clients.User, error) {
					assert.Equal(t, "test-user", name)
					updated = true
					return &clients.User{Username: "test-user"}, nil
				},
			},
		}

		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
			Spec: v2.UserSpec{
				ForProvider: v2.UserParameters{
					Username: "test-user",
					Email:    "new@example.com",
				},
			},
		}
		meta.SetExternalName(cr, "test-user")

		_, err := ec.Update(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, updated)
	})

	t.Run("missing external name returns error", func(t *testing.T) {
		ec := &externalClient{client: &mockUserClient{}}
		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
		}

		_, err := ec.Update(context.Background(), cr)
		require.Error(t, err)
	})
}

func TestDelete(t *testing.T) {
	t.Run("deletes user when exists", func(t *testing.T) {
		deleted := false
		ec := &externalClient{
			client: &mockUserClient{
				getUserFn: func(ctx context.Context, name string) (*clients.User, error) {
					return &clients.User{Username: name}, nil
				},
				deleteUserFn: func(ctx context.Context, name string) error {
					assert.Equal(t, "test-user", name)
					deleted = true
					return nil
				},
			},
		}

		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
		}
		meta.SetExternalName(cr, "test-user")

		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})

	t.Run("404 returns nil (already deleted)", func(t *testing.T) {
		deleted := false
		ec := &externalClient{
			client: &mockUserClient{
				getUserFn: func(ctx context.Context, name string) (*clients.User, error) {
					return nil, fmt.Errorf("API request failed with status 404: not found")
				},
				deleteUserFn: func(ctx context.Context, name string) error {
					deleted = true
					return nil
				},
			},
		}

		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
		}
		meta.SetExternalName(cr, "test-user")

		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, deleted, "Delete should not be called when external resource is already gone")
	})

	t.Run("missing external name returns nil", func(t *testing.T) {
		ec := &externalClient{client: &mockUserClient{}}
		cr := &v2.User{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-user"},
		}

		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})
}
