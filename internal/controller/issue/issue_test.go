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

package issue

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v2 "github.com/rossigee/provider-gitea/apis/issue/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockIssueClient struct {
	testutil.NoopClient
	getFn    func(ctx context.Context, owner, repo string, number int64) (*clients.Issue, error)
	createFn func(ctx context.Context, owner, repo string, req *clients.CreateIssueOptions) (*clients.Issue, error)
	updateFn func(ctx context.Context, owner, repo string, number int64, req *clients.UpdateIssueOptions) (*clients.Issue, error)
	deleteFn func(ctx context.Context, owner, repo string, number int64) error
}

func (m *mockIssueClient) GetIssue(ctx context.Context, owner, repo string, number int64) (*clients.Issue, error) {
	if m.getFn != nil {
		return m.getFn(ctx, owner, repo, number)
	}
	return nil, nil
}
func (m *mockIssueClient) CreateIssue(ctx context.Context, owner, repo string, req *clients.CreateIssueOptions) (*clients.Issue, error) {
	if m.createFn != nil {
		return m.createFn(ctx, owner, repo, req)
	}
	return nil, nil
}
func (m *mockIssueClient) UpdateIssue(ctx context.Context, owner, repo string, number int64, req *clients.UpdateIssueOptions) (*clients.Issue, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, owner, repo, number, req)
	}
	return nil, nil
}
func (m *mockIssueClient) DeleteIssue(ctx context.Context, owner, repo string, number int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, owner, repo, number)
	}
	return nil
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockIssueClient{}}
		cr := &v2.Issue{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "bug"}}
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockIssueClient{
			getFn: func(ctx context.Context, owner, repo string, number int64) (*clients.Issue, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.Issue{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "bug"}}
		meta.SetExternalName(cr, "acme/app/12")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("matching issue is up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockIssueClient{
			getFn: func(ctx context.Context, owner, repo string, number int64) (*clients.Issue, error) {
				assert.Equal(t, int64(12), number)
				return &clients.Issue{ID: 99, Number: 12, Title: "Bug", State: "open"}, nil
			},
		}}
		cr := &v2.Issue{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "bug"}}
		cr.Spec.ForProvider.Title = "Bug"
		meta.SetExternalName(cr, "acme/app/12")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("title drift is not up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockIssueClient{
			getFn: func(ctx context.Context, owner, repo string, number int64) (*clients.Issue, error) {
				return &clients.Issue{ID: 99, Number: 12, Title: "Old title", State: "open"}, nil
			},
		}}
		cr := &v2.Issue{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "bug"}}
		cr.Spec.ForProvider.Title = "New title"
		meta.SetExternalName(cr, "acme/app/12")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	ec := &externalClient{client: &mockIssueClient{
		createFn: func(ctx context.Context, owner, repo string, req *clients.CreateIssueOptions) (*clients.Issue, error) {
			assert.Equal(t, "acme", owner)
			assert.Equal(t, "app", repo)
			assert.Equal(t, "Bug", req.Title)
			return &clients.Issue{ID: 99, Number: 12, Title: "Bug"}, nil
		},
	}}
	cr := &v2.Issue{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "bug"}}
	cr.Spec.ForProvider.Owner = "acme"
	cr.Spec.ForProvider.Repository = "app"
	cr.Spec.ForProvider.Title = "Bug"
	_, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "acme/app/12", meta.GetExternalName(cr))
}

func TestDelete(t *testing.T) {
	t.Run("missing issue is success", func(t *testing.T) {
		ec := &externalClient{client: &mockIssueClient{
			getFn: func(ctx context.Context, owner, repo string, number int64) (*clients.Issue, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.Issue{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "bug"}}
		meta.SetExternalName(cr, "acme/app/12")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("existing issue is deleted", func(t *testing.T) {
		var deleted bool
		ec := &externalClient{client: &mockIssueClient{
			getFn: func(ctx context.Context, owner, repo string, number int64) (*clients.Issue, error) {
				return &clients.Issue{ID: 99, Number: 12}, nil
			},
			deleteFn: func(ctx context.Context, owner, repo string, number int64) error {
				deleted = true
				return nil
			},
		}}
		cr := &v2.Issue{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "bug"}}
		meta.SetExternalName(cr, "acme/app/12")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}
