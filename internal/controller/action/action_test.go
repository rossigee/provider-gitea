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

package action

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v2 "github.com/rossigee/provider-gitea/apis/action/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockActionClient struct {
	testutil.NoopClient
	getFn     func(ctx context.Context, repository, workflow string) (*clients.Action, error)
	createFn  func(ctx context.Context, repository string, req *clients.CreateActionRequest) (*clients.Action, error)
	updateFn  func(ctx context.Context, repository, workflow string, req *clients.UpdateActionRequest) (*clients.Action, error)
	deleteFn  func(ctx context.Context, repository, workflow string) error
	enableFn  func(ctx context.Context, repository, workflow string) error
	disableFn func(ctx context.Context, repository, workflow string) error
}

func (m *mockActionClient) GetAction(ctx context.Context, repository, workflow string) (*clients.Action, error) {
	if m.getFn != nil {
		return m.getFn(ctx, repository, workflow)
	}
	return nil, nil
}
func (m *mockActionClient) CreateAction(ctx context.Context, repository string, req *clients.CreateActionRequest) (*clients.Action, error) {
	if m.createFn != nil {
		return m.createFn(ctx, repository, req)
	}
	return nil, nil
}
func (m *mockActionClient) UpdateAction(ctx context.Context, repository, workflow string, req *clients.UpdateActionRequest) (*clients.Action, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, repository, workflow, req)
	}
	return nil, nil
}
func (m *mockActionClient) DeleteAction(ctx context.Context, repository, workflow string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, repository, workflow)
	}
	return nil
}
func (m *mockActionClient) EnableAction(ctx context.Context, repository, workflow string) error {
	if m.enableFn != nil {
		return m.enableFn(ctx, repository, workflow)
	}
	return nil
}
func (m *mockActionClient) DisableAction(ctx context.Context, repository, workflow string) error {
	if m.disableFn != nil {
		return m.disableFn(ctx, repository, workflow)
	}
	return nil
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockActionClient{}}
		cr := &v2.Action{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "wf"}}
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockActionClient{
			getFn: func(ctx context.Context, repository, workflow string) (*clients.Action, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.Action{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "wf"}}
		meta.SetExternalName(cr, "acme/app/ci.yml")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("active workflow with enabled matches", func(t *testing.T) {
		enabled := true
		ec := &externalClient{client: &mockActionClient{
			getFn: func(ctx context.Context, repository, workflow string) (*clients.Action, error) {
				assert.Equal(t, "acme/app", repository)
				assert.Equal(t, "ci.yml", workflow)
				return &clients.Action{WorkflowName: "ci.yml", State: "active"}, nil
			},
		}}
		cr := &v2.Action{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "wf"}}
		cr.Spec.ForProvider.Enabled = &enabled
		meta.SetExternalName(cr, "acme/app/ci.yml")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("disabled workflow with enabled drifts", func(t *testing.T) {
		enabled := true
		ec := &externalClient{client: &mockActionClient{
			getFn: func(ctx context.Context, repository, workflow string) (*clients.Action, error) {
				return &clients.Action{WorkflowName: "ci.yml", State: "disabled_manually"}, nil
			},
		}}
		cr := &v2.Action{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "wf"}}
		cr.Spec.ForProvider.Enabled = &enabled
		meta.SetExternalName(cr, "acme/app/ci.yml")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	ec := &externalClient{client: &mockActionClient{
		createFn: func(ctx context.Context, repository string, req *clients.CreateActionRequest) (*clients.Action, error) {
			assert.Equal(t, "acme/app", repository)
			assert.Equal(t, "ci.yml", req.WorkflowName)
			return &clients.Action{WorkflowName: "ci.yml"}, nil
		},
	}}
	cr := &v2.Action{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "wf"}}
	cr.Spec.ForProvider.Repository = "acme/app"
	cr.Spec.ForProvider.WorkflowName = "ci.yml"
	cr.Spec.ForProvider.Content = "jobs: {}"
	_, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "acme/app/ci.yml", meta.GetExternalName(cr))
}

func TestDelete(t *testing.T) {
	t.Run("missing workflow is success", func(t *testing.T) {
		ec := &externalClient{client: &mockActionClient{
			getFn: func(ctx context.Context, repository, workflow string) (*clients.Action, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.Action{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "wf"}}
		meta.SetExternalName(cr, "acme/app/ci.yml")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("existing workflow is deleted", func(t *testing.T) {
		var deleted bool
		ec := &externalClient{client: &mockActionClient{
			getFn: func(ctx context.Context, repository, workflow string) (*clients.Action, error) {
				return &clients.Action{WorkflowName: "ci.yml"}, nil
			},
			deleteFn: func(ctx context.Context, repository, workflow string) error {
				deleted = true
				return nil
			},
		}}
		cr := &v2.Action{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "wf"}}
		meta.SetExternalName(cr, "acme/app/ci.yml")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}
