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

package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamAssociationOperations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/orgs/acme/teams":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id": 7, "name": "Owners"}, {"id": 8, "name": "devs"}]`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/orgs/empty/teams":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/teams/7/members/alice":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 42, "username": "alice"}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/teams/7/members/bob":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == "PUT" && r.URL.Path == "/api/v1/teams/7/members/alice":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == "DELETE" && r.URL.Path == "/api/v1/teams/7/members/alice":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == "GET" && r.URL.Path == "/api/v1/teams/7/repos/acme/widget":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 55, "name": "widget", "full_name": "acme/widget"}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/teams/7/repos/acme/missing":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == "PUT" && r.URL.Path == "/api/v1/teams/7/repos/acme/widget":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == "DELETE" && r.URL.Path == "/api/v1/teams/7/repos/acme/widget":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := &giteaClient{
		httpClient: &http.Client{},
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
	}
	ctx := context.Background()

	t.Run("ListOrganizationTeams", func(t *testing.T) {
		teams, err := c.ListOrganizationTeams(ctx, "acme")
		require.NoError(t, err)
		assert.Len(t, teams, 2)
		assert.Equal(t, "Owners", teams[0].Name)
	})

	t.Run("ResolveTeamID found", func(t *testing.T) {
		id, err := ResolveTeamID(ctx, c, "acme", "Owners")
		require.NoError(t, err)
		assert.Equal(t, int64(7), id)
	})

	t.Run("ResolveTeamID not found", func(t *testing.T) {
		_, err := ResolveTeamID(ctx, c, "acme", "nonexistent")
		require.Error(t, err)
		assert.True(t, IsNotFound(err))
	})

	t.Run("ResolveTeamID empty org", func(t *testing.T) {
		_, err := ResolveTeamID(ctx, c, "empty", "anything")
		require.Error(t, err)
		assert.True(t, IsNotFound(err))
	})

	t.Run("GetTeamMember exists", func(t *testing.T) {
		m, err := c.GetTeamMember(ctx, 7, "alice")
		require.NoError(t, err)
		assert.Equal(t, "alice", m.Username)
	})

	t.Run("GetTeamMember not found", func(t *testing.T) {
		_, err := c.GetTeamMember(ctx, 7, "bob")
		require.Error(t, err)
		assert.True(t, IsNotFound(err))
	})

	t.Run("AddTeamMember", func(t *testing.T) {
		err := c.AddTeamMember(ctx, 7, "alice")
		require.NoError(t, err)
	})

	t.Run("RemoveTeamMember", func(t *testing.T) {
		err := c.RemoveTeamMember(ctx, 7, "alice")
		require.NoError(t, err)
	})

	t.Run("GetTeamRepository exists", func(t *testing.T) {
		repo, err := c.GetTeamRepository(ctx, 7, "acme", "widget")
		require.NoError(t, err)
		assert.Equal(t, "widget", repo.Name)
	})

	t.Run("GetTeamRepository not found", func(t *testing.T) {
		_, err := c.GetTeamRepository(ctx, 7, "acme", "missing")
		require.Error(t, err)
		assert.True(t, IsNotFound(err))
	})

	t.Run("AddTeamRepository", func(t *testing.T) {
		err := c.AddTeamRepository(ctx, 7, "acme", "widget")
		require.NoError(t, err)
	})

	t.Run("RemoveTeamRepository", func(t *testing.T) {
		err := c.RemoveTeamRepository(ctx, 7, "acme", "widget")
		require.NoError(t, err)
	})
}
