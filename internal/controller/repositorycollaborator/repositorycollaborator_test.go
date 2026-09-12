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

package repositorycollaborator

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v2 "github.com/rossigee/provider-gitea/apis/repositorycollaborator/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockCollabClient struct {
	testutil.NoopClient
	getFn    func(ctx context.Context, owner, repo, username string) (*clients.RepositoryCollaborator, error)
	addFn    func(ctx context.Context, owner, repo, username string, req *clients.AddCollaboratorRequest) error
	updateFn func(ctx context.Context, owner, repo, username string, req *clients.UpdateCollaboratorRequest) error
	removeFn func(ctx context.Context, owner, repo, username string) error
}

func (m *mockCollabClient) GetRepositoryCollaborator(ctx context.Context, owner, repo, username string) (*clients.RepositoryCollaborator, error) {
	if m.getFn != nil {
		return m.getFn(ctx, owner, repo, username)
	}
	return nil, nil
}
func (m *mockCollabClient) AddRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *clients.AddCollaboratorRequest) error {
	if m.addFn != nil {
		return m.addFn(ctx, owner, repo, username, req)
	}
	return nil
}
func (m *mockCollabClient) UpdateRepositoryCollaborator(ctx context.Context, owner, repo, username string, req *clients.UpdateCollaboratorRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, owner, repo, username, req)
	}
	return nil
}
func (m *mockCollabClient) RemoveRepositoryCollaborator(ctx context.Context, owner, repo, username string) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, owner, repo, username)
	}
	return nil
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockCollabClient{}}
		cr := &v2.RepositoryCollaborator{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "c"}}
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockCollabClient{
			getFn: func(ctx context.Context, owner, repo, username string) (*clients.RepositoryCollaborator, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.RepositoryCollaborator{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "c"}}
		meta.SetExternalName(cr, "acme/app/bob")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("matching permission is up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockCollabClient{
			getFn: func(ctx context.Context, owner, repo, username string) (*clients.RepositoryCollaborator, error) {
				return &clients.RepositoryCollaborator{Username: "bob",
					Permissions: clients.RepositoryCollaboratorPermissions{Push: true, Pull: true}}, nil
			},
		}}
		cr := &v2.RepositoryCollaborator{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "c"}}
		cr.Spec.ForProvider.Permission = "write"
		meta.SetExternalName(cr, "acme/app/bob")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("permission drift is not up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockCollabClient{
			getFn: func(ctx context.Context, owner, repo, username string) (*clients.RepositoryCollaborator, error) {
				return &clients.RepositoryCollaborator{Username: "bob",
					Permissions: clients.RepositoryCollaboratorPermissions{Pull: true}}, nil
			},
		}}
		cr := &v2.RepositoryCollaborator{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "c"}}
		cr.Spec.ForProvider.Permission = "write"
		meta.SetExternalName(cr, "acme/app/bob")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	ec := &externalClient{client: &mockCollabClient{
		addFn: func(ctx context.Context, owner, repo, username string, req *clients.AddCollaboratorRequest) error {
			assert.Equal(t, "acme", owner)
			assert.Equal(t, "bob", username)
			assert.Equal(t, "read", req.Permission)
			return nil
		},
	}}
	cr := &v2.RepositoryCollaborator{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "c"}}
	cr.Spec.ForProvider.Repository = "acme/app"
	cr.Spec.ForProvider.Username = "bob"
	cr.Spec.ForProvider.Permission = "read"
	_, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "acme/app/bob", meta.GetExternalName(cr))
}

func TestDelete(t *testing.T) {
	t.Run("missing collaborator is success", func(t *testing.T) {
		ec := &externalClient{client: &mockCollabClient{
			getFn: func(ctx context.Context, owner, repo, username string) (*clients.RepositoryCollaborator, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.RepositoryCollaborator{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "c"}}
		meta.SetExternalName(cr, "acme/app/bob")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("existing collaborator is removed", func(t *testing.T) {
		var removed bool
		ec := &externalClient{client: &mockCollabClient{
			getFn: func(ctx context.Context, owner, repo, username string) (*clients.RepositoryCollaborator, error) {
				return &clients.RepositoryCollaborator{Username: "bob"}, nil
			},
			removeFn: func(ctx context.Context, owner, repo, username string) error {
				removed = true
				return nil
			},
		}}
		cr := &v2.RepositoryCollaborator{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "c"}}
		meta.SetExternalName(cr, "acme/app/bob")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, removed)
	})
}
