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
	"testing"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/repositorycollaborator/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the
// collaborator verbs this controller uses; every other method panics via the
// embedded nil interface so an accidental call is loud.
type fakeClient struct {
	clients.Client
	getCollab *clients.RepositoryCollaborator
	getErr    error
	added     *clients.AddCollaboratorRequest
	addErr    error
	updated   *clients.UpdateCollaboratorRequest
	deleteErr error
	deleted   bool
}

func (f *fakeClient) GetRepositoryCollaborator(_ context.Context, _, _, _ string) (*clients.RepositoryCollaborator, error) {
	return f.getCollab, f.getErr
}

func (f *fakeClient) AddRepositoryCollaborator(_ context.Context, _, _, _ string, req *clients.AddCollaboratorRequest) error {
	f.added = req
	return f.addErr
}

func (f *fakeClient) UpdateRepositoryCollaborator(_ context.Context, _, _, _ string, req *clients.UpdateCollaboratorRequest) error {
	f.updated = req
	return nil
}

func (f *fakeClient) RemoveRepositoryCollaborator(_ context.Context, _, _, _ string) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.RepositoryCollaborator {
	cr := &v2.RepositoryCollaborator{}
	cr.SetName("my-collab")
	cr.Spec.ForProvider.Username = "alice"
	cr.Spec.ForProvider.Repository = "acme/widget"
	cr.Spec.ForProvider.Permission = "read"
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

func TestObserveNotCreated(t *testing.T) {
	// Identity is spec.Username (not the external-name). With no username there
	// is nothing to GET -> not created.
	f := &fakeClient{getErr: errors.New("GET should not be called")}
	e := &external{client: f}

	cr := newCR("")
	cr.Spec.ForProvider.Username = ""

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false when spec.username is empty")
	}
}

func TestObserveNotFound(t *testing.T) {
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("collaborator", "acme/widget/alice"), "failed to get repository collaborator")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("alice"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getCollab: &clients.RepositoryCollaborator{
		Username:    "alice",
		Permissions: clients.RepositoryCollaboratorPermissions{Pull: true},
	}}
	e := &external{client: f}

	cr := newCR("alice")

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

func TestCreateSetsExternalName(t *testing.T) {
	f := &fakeClient{}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "alice" {
		t.Fatalf("expected external-name alice, got %q", got)
	}
	if f.added == nil || f.added.Permission != "read" {
		t.Fatalf("expected Add called with permission read, got %+v", f.added)
	}
}

func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("collaborator", "acme/widget/alice")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("alice")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected RemoveRepositoryCollaborator to have been called")
	}
}
