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

package teamrepository

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/teamrepository/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the verbs this
// controller uses; every other method panics via the embedded nil interface so
// an accidental call is loud.
type fakeClient struct {
	clients.Client
	teams        []clients.Team
	listTeamsErr error

	getRepo   *clients.Repository
	getErr    error
	added     bool
	addErr    error
	removed   bool
	deleteErr error
}

func (f *fakeClient) ListOrganizationTeams(_ context.Context, _ string) ([]clients.Team, error) {
	return f.teams, f.listTeamsErr
}

func (f *fakeClient) GetTeamRepository(_ context.Context, _ int64, _, _ string) (*clients.Repository, error) {
	return f.getRepo, f.getErr
}

func (f *fakeClient) AddTeamRepository(_ context.Context, _ int64, _, _ string) error {
	f.added = true
	return f.addErr
}

func (f *fakeClient) RemoveTeamRepository(_ context.Context, _ int64, _, _ string) error {
	f.removed = true
	return f.deleteErr
}

func newCR() *v2.TeamRepository {
	cr := &v2.TeamRepository{}
	cr.SetName("my-attachment")
	cr.Spec.ForProvider.Organization = "acme"
	cr.Spec.ForProvider.Team = "devs"
	cr.Spec.ForProvider.Repository = "widget"
	return cr
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

func TestObserveNotCreated(t *testing.T) {
	f := &fakeClient{listTeamsErr: errors.New("ListOrganizationTeams should not be called")}
	e := &external{client: f}

	cr := newCR()
	cr.Spec.ForProvider.Repository = ""

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false when spec.repository is empty")
	}
}

func TestObserveTeamNotFound(t *testing.T) {
	// Team resolution 404s -> not created, let Create surface the real error.
	f := &fakeClient{teams: []clients.Team{{ID: 1, Name: "other"}}}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR())
	if err != nil {
		t.Fatalf("team-not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false when team cannot be resolved")
	}
}

func TestObserveRepositoryNotAttached(t *testing.T) {
	f := &fakeClient{
		teams:  []clients.Team{{ID: 8, Name: "devs"}},
		getErr: clients.NewNotFoundError("team repository", "8/acme/widget"),
	}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR())
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

func TestObserveAvailable(t *testing.T) {
	f := &fakeClient{
		teams:   []clients.Team{{ID: 8, Name: "devs"}},
		getRepo: &clients.Repository{Name: "widget"},
	}
	e := &external{client: f}

	cr := newCR()
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
	if cr.Status.AtProvider.TeamID == nil || *cr.Status.AtProvider.TeamID != 8 {
		t.Fatalf("expected resolved TeamID=8 in status, got %+v", cr.Status.AtProvider)
	}
}

func TestObserveUsesTeamIDEscapeHatch(t *testing.T) {
	// With TeamID set, ListOrganizationTeams must never be called.
	teamID := int64(99)
	f := &fakeClient{
		listTeamsErr: errors.New("ListOrganizationTeams should not be called"),
		getRepo:      &clients.Repository{Name: "widget"},
	}
	e := &external{client: f}

	cr := newCR()
	cr.Spec.ForProvider.TeamID = &teamID

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatalf("expected ResourceExists=true")
	}
}

func TestCreateSetsExternalName(t *testing.T) {
	f := &fakeClient{teams: []clients.Team{{ID: 8, Name: "devs"}}}
	e := &external{client: f}

	cr := newCR()
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.added {
		t.Fatalf("expected AddTeamRepository to have been called")
	}
	if got, want := meta.GetExternalName(cr), "acme/devs/widget"; got != want {
		t.Fatalf("expected external-name %q, got %q", want, got)
	}
}

func TestCreateIdempotentOnAlreadyAttached(t *testing.T) {
	f := &fakeClient{teams: []clients.Team{{ID: 8, Name: "devs"}}}
	e := &external{client: f}

	if _, err := e.Create(context.Background(), newCR()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateIsNoop(t *testing.T) {
	e := &external{client: &fakeClient{}}
	if _, err := e.Update(context.Background(), newCR()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{
		teams:     []clients.Team{{ID: 8, Name: "devs"}},
		deleteErr: clients.NewNotFoundError("team repository", "8/acme/widget"),
	}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR()); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.removed {
		t.Fatalf("expected RemoveTeamRepository to have been called")
	}
}

func TestDeleteTeamAlreadyGoneIsNoop(t *testing.T) {
	// Team itself no longer resolves -> attachment is moot.
	f := &fakeClient{teams: nil}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.removed {
		t.Fatalf("RemoveTeamRepository should not be called when the team cannot be resolved")
	}
}
