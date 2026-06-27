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

package team

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/team/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient embeds clients.Client so unimplemented methods panic if called.
type fakeClient struct {
	clients.Client
	getTeam    *clients.Team
	getErr     error
	createResp *clients.Team
	createErr  error
	deleteErr  error
	deleted    bool
}

func (f *fakeClient) GetTeam(_ context.Context, _ int64) (*clients.Team, error) {
	return f.getTeam, f.getErr
}

func (f *fakeClient) CreateTeam(_ context.Context, _ string, _ *clients.CreateTeamRequest) (*clients.Team, error) {
	return f.createResp, f.createErr
}

func (f *fakeClient) UpdateTeam(_ context.Context, _ int64, _ *clients.UpdateTeamRequest) (*clients.Team, error) {
	return f.getTeam, nil
}

func (f *fakeClient) DeleteTeam(_ context.Context, _ int64) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.Team {
	cr := &v2.Team{}
	cr.SetName("my-team")
	cr.Spec.ForProvider.Name = "my-team"
	cr.Spec.ForProvider.Organization = "acme"
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

func TestObserveNotCreated(t *testing.T) {
	f := &fakeClient{getErr: errors.New("GET should not be called")}
	e := &external{client: f}

	// Empty external-name -> not created, no GET.
	obs, err := e.Observe(context.Background(), newCR(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false for empty external-name")
	}

	// A non-numeric external-name is equally "not created" (don't GET id 0).
	obs, err = e.Observe(context.Background(), newCR("not-a-number"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false for non-numeric external-name")
	}
}

func TestObserveNotFound(t *testing.T) {
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("team", "42"), "failed to get team")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("42"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getTeam: &clients.Team{ID: 42, Name: "my-team", Description: "hello"}}
	e := &external{client: f}

	cr := newCR("42")
	desc := "hello"
	cr.Spec.ForProvider.Description = &desc

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists || !obs.ResourceUpToDate {
		t.Fatalf("expected exists+upToDate, got %+v", obs)
	}
	if !isAvailable(cr) {
		t.Fatalf("Observe must set Available() on the exists path")
	}
}

func TestCreateSetsExternalNameFromBackend(t *testing.T) {
	f := &fakeClient{createResp: &clients.Team{ID: 42, Name: "my-team"}}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "42" {
		t.Fatalf("expected external-name 42 (numeric team id), got %q", got)
	}
}

func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("team", "42")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("42")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteTeam to have been called")
	}
}
