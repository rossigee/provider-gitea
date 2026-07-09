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

package teammembership

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/teammembership/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the verbs this
// controller uses; every other method panics via the embedded nil interface so
// an accidental call is loud.
type fakeClient struct {
	clients.Client
	teams        []clients.Team
	listTeamsErr error

	getMember *clients.User
	getErr    error
	added     bool
	addErr    error
	removed   bool
	deleteErr error
}

func (f *fakeClient) ListOrganizationTeams(_ context.Context, _ string) ([]clients.Team, error) {
	return f.teams, f.listTeamsErr
}

func (f *fakeClient) GetTeamMember(_ context.Context, _ int64, _ string) (*clients.User, error) {
	return f.getMember, f.getErr
}

func (f *fakeClient) AddTeamMember(_ context.Context, _ int64, _ string) error {
	f.added = true
	return f.addErr
}

func (f *fakeClient) RemoveTeamMember(_ context.Context, _ int64, _ string) error {
	f.removed = true
	return f.deleteErr
}

func newCR() *v2.TeamMembership {
	cr := &v2.TeamMembership{}
	cr.SetName("my-membership")
	cr.Spec.ForProvider.Organization = "acme"
	cr.Spec.ForProvider.Team = "Owners"
	cr.Spec.ForProvider.Username = "alice"
	return cr
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

func TestObserveNotCreated(t *testing.T) {
	f := &fakeClient{listTeamsErr: errors.New("ListOrganizationTeams should not be called")}
	e := &external{client: f}

	cr := newCR()
	cr.Spec.ForProvider.Username = ""

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false when spec.username is empty")
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

func TestObserveMemberNotFound(t *testing.T) {
	f := &fakeClient{
		teams:  []clients.Team{{ID: 7, Name: "Owners"}},
		getErr: clients.NewNotFoundError("team member", "7/alice"),
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
		teams:     []clients.Team{{ID: 7, Name: "Owners"}},
		getMember: &clients.User{Username: "alice"},
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
	if cr.Status.AtProvider.TeamID == nil || *cr.Status.AtProvider.TeamID != 7 {
		t.Fatalf("expected resolved TeamID=7 in status, got %+v", cr.Status.AtProvider)
	}
}

func TestObserveUsesTeamIDEscapeHatch(t *testing.T) {
	// With TeamID set, ListOrganizationTeams must never be called.
	teamID := int64(99)
	f := &fakeClient{
		listTeamsErr: errors.New("ListOrganizationTeams should not be called"),
		getMember:    &clients.User{Username: "alice"},
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
	f := &fakeClient{teams: []clients.Team{{ID: 7, Name: "Owners"}}}
	e := &external{client: f}

	cr := newCR()
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.added {
		t.Fatalf("expected AddTeamMember to have been called")
	}
	if got, want := meta.GetExternalName(cr), "acme/Owners/alice"; got != want {
		t.Fatalf("expected external-name %q, got %q", want, got)
	}
}

func TestCreateIdempotentOnAlreadyMember(t *testing.T) {
	// Gitea returns 204 even if the user is already a member; AddTeamMember
	// surfaces no error in that case.
	f := &fakeClient{teams: []clients.Team{{ID: 7, Name: "Owners"}}}
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
		teams:     []clients.Team{{ID: 7, Name: "Owners"}},
		deleteErr: clients.NewNotFoundError("team member", "7/alice"),
	}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR()); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.removed {
		t.Fatalf("expected RemoveTeamMember to have been called")
	}
}

func TestDeleteTeamAlreadyGoneIsNoop(t *testing.T) {
	// Team itself no longer resolves -> membership is moot.
	f := &fakeClient{teams: nil}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.removed {
		t.Fatalf("RemoveTeamMember should not be called when the team cannot be resolved")
	}
}
