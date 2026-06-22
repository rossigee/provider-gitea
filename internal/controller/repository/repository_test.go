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

package repository

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/repository/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the repository
// verbs this controller uses; every other method panics so an accidental call
// is loud. It records the last request and serves canned responses/errors.
type fakeClient struct {
	clients.Client
	getRepo    *clients.Repository
	getErr     error
	created    *clients.CreateRepositoryRequest
	createResp *clients.Repository
	createErr  error
	updated    *clients.UpdateRepositoryRequest
	deleteErr  error
	deleted    bool
}

func (f *fakeClient) GetRepository(_ context.Context, _, _ string) (*clients.Repository, error) {
	return f.getRepo, f.getErr
}

func (f *fakeClient) CreateRepository(_ context.Context, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
	f.created = req
	return f.createResp, f.createErr
}

func (f *fakeClient) CreateOrganizationRepository(_ context.Context, _ string, req *clients.CreateRepositoryRequest) (*clients.Repository, error) {
	f.created = req
	return f.createResp, f.createErr
}

func (f *fakeClient) UpdateRepository(_ context.Context, _, _ string, req *clients.UpdateRepositoryRequest) (*clients.Repository, error) {
	f.updated = req
	return f.getRepo, nil
}

func (f *fakeClient) DeleteRepository(_ context.Context, _, _ string) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.Repository {
	cr := &v2.Repository{}
	cr.SetName("my-repo")
	cr.Spec.ForProvider.Name = "my-repo"
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

// TestObserveNotCreated: an empty/garbage external-name means "not created" —
// Observe must report ResourceExists:false without issuing a GET (lesson #7/#14).
func TestObserveNotCreated(t *testing.T) {
	f := &fakeClient{getErr: errors.New("GET should not be called")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false for empty external-name")
	}
}

// TestObserveNotFound: a typed 404 from the backend must be classified as
// absent via clients.IsNotFound, not surfaced as an error (lesson #3).
func TestObserveNotFound(t *testing.T) {
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("repository", "acme/my-repo"), "failed to get repository")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("acme/my-repo"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

// TestObserveRealErrorSurfaces: a non-404 failure (e.g. 500/auth) must surface
// as an error so the resource is never spuriously recreated (lesson #3).
func TestObserveRealErrorSurfaces(t *testing.T) {
	f := &fakeClient{getErr: &clients.APIError{StatusCode: 500, Body: "boom"}}
	e := &external{client: f}

	if _, err := e.Observe(context.Background(), newCR("acme/my-repo")); err == nil {
		t.Fatalf("expected 500 to surface as an error")
	}
}

// TestObserveAvailableAndUpToDate: an existing, matching repo must be marked
// Available and up-to-date (lesson #2/#6 — runtime v2 won't set Ready for us).
func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getRepo: &clients.Repository{ID: 7, Name: "my-repo", Description: "hello"}}
	e := &external{client: f}

	cr := newCR("acme/my-repo")
	cr.Spec.ForProvider.Description = ptr.To("hello")

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

// TestObserveDriftStillAvailable: a spec/backend mismatch must report
// ResourceUpToDate:false (so Update fires) while KEEPING Ready=Available — drift
// is signalled via Synced, never by withholding Ready (lesson #6).
func TestObserveDriftStillAvailable(t *testing.T) {
	f := &fakeClient{getRepo: &clients.Repository{ID: 7, Name: "my-repo", Description: "old"}}
	e := &external{client: f}

	cr := newCR("acme/my-repo")
	cr.Spec.ForProvider.Description = ptr.To("new")

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Fatalf("expected drift (ResourceUpToDate=false)")
	}
	if !isAvailable(cr) {
		t.Fatalf("drift must not downgrade Ready")
	}
}

// TestCreateSetsExternalNameFromBackend: Create must pin the external-name to the
// authoritative owner/name returned by the backend, not a spec guess (lesson
// #3/#7).
func TestCreateSetsExternalNameFromBackend(t *testing.T) {
	f := &fakeClient{createResp: &clients.Repository{
		ID: 7, Name: "my-repo", Owner: &clients.User{Username: "acme"},
	}}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "acme/my-repo" {
		t.Fatalf("expected external-name acme/my-repo, got %q", got)
	}
}

// TestDeleteIdempotentOn404: a delete against an already-absent repo must
// succeed so the finalizer releases (lesson #16).
func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("repository", "acme/my-repo")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("acme/my-repo")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteRepository to have been called")
	}
}
