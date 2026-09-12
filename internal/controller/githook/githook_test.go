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

package githook

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v2 "github.com/rossigee/provider-gitea/apis/githook/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockGitHookClient struct {
	testutil.NoopClient
	getFn    func(ctx context.Context, repository, hookType string) (*clients.GitHook, error)
	createFn func(ctx context.Context, repository string, req *clients.CreateGitHookRequest) (*clients.GitHook, error)
	updateFn func(ctx context.Context, repository, hookType string, req *clients.UpdateGitHookRequest) (*clients.GitHook, error)
	deleteFn func(ctx context.Context, repository, hookType string) error
}

func (m *mockGitHookClient) GetGitHook(ctx context.Context, repository, hookType string) (*clients.GitHook, error) {
	if m.getFn != nil {
		return m.getFn(ctx, repository, hookType)
	}
	return nil, nil
}
func (m *mockGitHookClient) CreateGitHook(ctx context.Context, repository string, req *clients.CreateGitHookRequest) (*clients.GitHook, error) {
	if m.createFn != nil {
		return m.createFn(ctx, repository, req)
	}
	return nil, nil
}
func (m *mockGitHookClient) UpdateGitHook(ctx context.Context, repository, hookType string, req *clients.UpdateGitHookRequest) (*clients.GitHook, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, repository, hookType, req)
	}
	return nil, nil
}
func (m *mockGitHookClient) DeleteGitHook(ctx context.Context, repository, hookType string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, repository, hookType)
	}
	return nil
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockGitHookClient{}}
		cr := &v2.GitHook{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hook"}}
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockGitHookClient{
			getFn: func(ctx context.Context, repository, hookType string) (*clients.GitHook, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.GitHook{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hook"}}
		meta.SetExternalName(cr, "acme/app/pre-receive")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("matching content is up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockGitHookClient{
			getFn: func(ctx context.Context, repository, hookType string) (*clients.GitHook, error) {
				assert.Equal(t, "acme/app", repository)
				assert.Equal(t, "pre-receive", hookType)
				return &clients.GitHook{Name: "pre-receive", Content: "#!/bin/sh\nexit 0", IsActive: true}, nil
			},
		}}
		cr := &v2.GitHook{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hook"}}
		cr.Spec.ForProvider.Content = "#!/bin/sh\nexit 0"
		meta.SetExternalName(cr, "acme/app/pre-receive")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("content drift is not up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockGitHookClient{
			getFn: func(ctx context.Context, repository, hookType string) (*clients.GitHook, error) {
				return &clients.GitHook{Name: "pre-receive", Content: "#!/bin/sh\nexit 1", IsActive: true}, nil
			},
		}}
		cr := &v2.GitHook{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hook"}}
		cr.Spec.ForProvider.Content = "#!/bin/sh\nexit 0"
		meta.SetExternalName(cr, "acme/app/pre-receive")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	ec := &externalClient{client: &mockGitHookClient{
		createFn: func(ctx context.Context, repository string, req *clients.CreateGitHookRequest) (*clients.GitHook, error) {
			assert.Equal(t, "acme/app", repository)
			assert.Equal(t, "pre-receive", req.HookType)
			return &clients.GitHook{Name: "pre-receive"}, nil
		},
	}}
	cr := &v2.GitHook{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hook"}}
	cr.Spec.ForProvider.Repository = "acme/app"
	cr.Spec.ForProvider.HookType = "pre-receive"
	cr.Spec.ForProvider.Content = "#!/bin/sh\nexit 0"
	_, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "acme/app/pre-receive", meta.GetExternalName(cr))
}

func TestDelete(t *testing.T) {
	t.Run("missing hook is success", func(t *testing.T) {
		ec := &externalClient{client: &mockGitHookClient{
			getFn: func(ctx context.Context, repository, hookType string) (*clients.GitHook, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.GitHook{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hook"}}
		meta.SetExternalName(cr, "acme/app/pre-receive")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("existing hook is deleted", func(t *testing.T) {
		var deleted bool
		ec := &externalClient{client: &mockGitHookClient{
			getFn: func(ctx context.Context, repository, hookType string) (*clients.GitHook, error) {
				return &clients.GitHook{Name: "pre-receive"}, nil
			},
			deleteFn: func(ctx context.Context, repository, hookType string) error {
				deleted = true
				return nil
			},
		}}
		cr := &v2.GitHook{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hook"}}
		meta.SetExternalName(cr, "acme/app/pre-receive")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}
