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

package organizationmember

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v2 "github.com/rossigee/provider-gitea/apis/organizationmember/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockMemberClient struct {
	testutil.NoopClient
	getFn    func(ctx context.Context, org, username string) (*clients.OrganizationMember, error)
	addFn    func(ctx context.Context, org, username string, req *clients.AddOrganizationMemberRequest) (*clients.OrganizationMember, error)
	updateFn func(ctx context.Context, org, username string, req *clients.UpdateOrganizationMemberRequest) (*clients.OrganizationMember, error)
	removeFn func(ctx context.Context, org, username string) error
}

func (m *mockMemberClient) GetOrganizationMember(ctx context.Context, org, username string) (*clients.OrganizationMember, error) {
	if m.getFn != nil {
		return m.getFn(ctx, org, username)
	}
	return nil, nil
}
func (m *mockMemberClient) AddOrganizationMember(ctx context.Context, org, username string, req *clients.AddOrganizationMemberRequest) (*clients.OrganizationMember, error) {
	if m.addFn != nil {
		return m.addFn(ctx, org, username, req)
	}
	return nil, nil
}
func (m *mockMemberClient) UpdateOrganizationMember(ctx context.Context, org, username string, req *clients.UpdateOrganizationMemberRequest) (*clients.OrganizationMember, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, org, username, req)
	}
	return nil, nil
}
func (m *mockMemberClient) RemoveOrganizationMember(ctx context.Context, org, username string) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, org, username)
	}
	return nil
}

func TestObserve(t *testing.T) {
	t.Run("no external name returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockMemberClient{}}
		cr := &v2.OrganizationMember{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "m"}}
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockMemberClient{
			getFn: func(ctx context.Context, org, username string) (*clients.OrganizationMember, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.OrganizationMember{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "m"}}
		meta.SetExternalName(cr, "acme/alice")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("matching member is up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockMemberClient{
			getFn: func(ctx context.Context, org, username string) (*clients.OrganizationMember, error) {
				assert.Equal(t, "acme", org)
				assert.Equal(t, "alice", username)
				return &clients.OrganizationMember{Username: "alice", Role: "member", Visibility: "private"}, nil
			},
		}}
		cr := &v2.OrganizationMember{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "m"}}
		cr.Spec.ForProvider.Role = "member"
		meta.SetExternalName(cr, "acme/alice")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("role drift is not up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockMemberClient{
			getFn: func(ctx context.Context, org, username string) (*clients.OrganizationMember, error) {
				return &clients.OrganizationMember{Username: "alice", Role: "member", Visibility: "private"}, nil
			},
		}}
		cr := &v2.OrganizationMember{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "m"}}
		cr.Spec.ForProvider.Role = "admin"
		meta.SetExternalName(cr, "acme/alice")
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	ec := &externalClient{client: &mockMemberClient{
		addFn: func(ctx context.Context, org, username string, req *clients.AddOrganizationMemberRequest) (*clients.OrganizationMember, error) {
			assert.Equal(t, "acme", org)
			assert.Equal(t, "alice", username)
			assert.Equal(t, "member", req.Role)
			return &clients.OrganizationMember{Username: "alice"}, nil
		},
	}}
	cr := &v2.OrganizationMember{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "m"}}
	cr.Spec.ForProvider.Organization = "acme"
	cr.Spec.ForProvider.Username = "alice"
	cr.Spec.ForProvider.Role = "member"
	_, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "acme/alice", meta.GetExternalName(cr))
}

func TestDelete(t *testing.T) {
	t.Run("missing member is success", func(t *testing.T) {
		ec := &externalClient{client: &mockMemberClient{
			getFn: func(ctx context.Context, org, username string) (*clients.OrganizationMember, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.OrganizationMember{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "m"}}
		meta.SetExternalName(cr, "acme/alice")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
	})

	t.Run("existing member is removed", func(t *testing.T) {
		var removed bool
		ec := &externalClient{client: &mockMemberClient{
			getFn: func(ctx context.Context, org, username string) (*clients.OrganizationMember, error) {
				return &clients.OrganizationMember{Username: "alice"}, nil
			},
			removeFn: func(ctx context.Context, org, username string) error {
				removed = true
				return nil
			},
		}}
		cr := &v2.OrganizationMember{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "m"}}
		meta.SetExternalName(cr, "acme/alice")
		_, err := ec.Delete(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, removed)
	})
}
