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

package organizationsettings

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v2 "github.com/rossigee/provider-gitea/apis/organizationsettings/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockSettingsClient struct {
	testutil.NoopClient
	getFn    func(ctx context.Context, org string) (*clients.OrganizationSettings, error)
	updateFn func(ctx context.Context, org string, req *clients.UpdateOrganizationSettingsRequest) (*clients.OrganizationSettings, error)
}

func (m *mockSettingsClient) GetOrganizationSettings(ctx context.Context, org string) (*clients.OrganizationSettings, error) {
	if m.getFn != nil {
		return m.getFn(ctx, org)
	}
	return nil, nil
}
func (m *mockSettingsClient) UpdateOrganizationSettings(ctx context.Context, org string, req *clients.UpdateOrganizationSettingsRequest) (*clients.OrganizationSettings, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, org, req)
	}
	return nil, nil
}

func TestObserve(t *testing.T) {
	t.Run("404 returns not exists", func(t *testing.T) {
		ec := &externalClient{client: &mockSettingsClient{
			getFn: func(ctx context.Context, org string) (*clients.OrganizationSettings, error) {
				return nil, fmt.Errorf("API request failed with status 404: not found")
			},
		}}
		cr := &v2.OrganizationSettings{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "s"}}
		cr.Spec.ForProvider.Organization = "acme"
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.False(t, obs.ResourceExists)
	})

	t.Run("matching settings are up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockSettingsClient{
			getFn: func(ctx context.Context, org string) (*clients.OrganizationSettings, error) {
				assert.Equal(t, "acme", org)
				return &clients.OrganizationSettings{DefaultRepoPermission: "read", MembersCanCreateRepos: true}, nil
			},
		}}
		perm := "read"
		create := true
		cr := &v2.OrganizationSettings{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "s"}}
		cr.Spec.ForProvider.Organization = "acme"
		cr.Spec.ForProvider.DefaultRepoPermission = &perm
		cr.Spec.ForProvider.MembersCanCreateRepos = &create
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.True(t, obs.ResourceUpToDate)
	})

	t.Run("permission drift is not up to date", func(t *testing.T) {
		ec := &externalClient{client: &mockSettingsClient{
			getFn: func(ctx context.Context, org string) (*clients.OrganizationSettings, error) {
				return &clients.OrganizationSettings{DefaultRepoPermission: "read"}, nil
			},
		}}
		perm := "write"
		cr := &v2.OrganizationSettings{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "s"}}
		cr.Spec.ForProvider.Organization = "acme"
		cr.Spec.ForProvider.DefaultRepoPermission = &perm
		obs, err := ec.Observe(context.Background(), cr)
		require.NoError(t, err)
		assert.True(t, obs.ResourceExists)
		assert.False(t, obs.ResourceUpToDate)
	})
}

func TestCreate(t *testing.T) {
	ec := &externalClient{client: &mockSettingsClient{
		updateFn: func(ctx context.Context, org string, req *clients.UpdateOrganizationSettingsRequest) (*clients.OrganizationSettings, error) {
			assert.Equal(t, "acme", org)
			require.NotNil(t, req.DefaultRepoPermission)
			assert.Equal(t, "read", *req.DefaultRepoPermission)
			return &clients.OrganizationSettings{}, nil
		},
	}}
	perm := "read"
	cr := &v2.OrganizationSettings{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "s"}}
	cr.Spec.ForProvider.Organization = "acme"
	cr.Spec.ForProvider.DefaultRepoPermission = &perm
	_, err := ec.Create(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "acme", meta.GetExternalName(cr))
}

func TestDelete(t *testing.T) {
	ec := &externalClient{client: &mockSettingsClient{}}
	cr := &v2.OrganizationSettings{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "s"}}
	_, err := ec.Delete(context.Background(), cr)
	require.NoError(t, err)
}
