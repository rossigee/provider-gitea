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

package runner

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/runner/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the runner
// verbs this controller uses; every other method panics (via the embedded nil
// interface) so an accidental call is loud.
type fakeClient struct {
	clients.Client
	getRunner  *clients.Runner
	getErr     error
	created    *clients.CreateRunnerRequest
	createResp *clients.Runner
	createErr  error
	updated    *clients.UpdateRunnerRequest
	deleteErr  error
	deleted    bool
}

func (f *fakeClient) GetRunner(_ context.Context, _, _ string, _ int64) (*clients.Runner, error) {
	return f.getRunner, f.getErr
}

func (f *fakeClient) CreateRunner(_ context.Context, _, _ string, req *clients.CreateRunnerRequest) (*clients.Runner, error) {
	f.created = req
	return f.createResp, f.createErr
}

func (f *fakeClient) UpdateRunner(_ context.Context, _, _ string, _ int64, req *clients.UpdateRunnerRequest) (*clients.Runner, error) {
	f.updated = req
	return f.getRunner, nil
}

func (f *fakeClient) DeleteRunner(_ context.Context, _, _ string, _ int64) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.Runner {
	cr := &v2.Runner{}
	cr.SetName("my-runner")
	cr.Spec.ForProvider.Name = "my-runner"
	cr.Spec.ForProvider.Scope = "organization"
	cr.Spec.ForProvider.ScopeValue = ptr.To("acme")
	cr.Spec.ForProvider.Labels = []string{"linux"}
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

// TestObserveNotFound: a typed 404 must be classified as absent via
// clients.IsNotFound, not surfaced as an error (lesson #3).
func TestObserveNotFound(t *testing.T) {
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("runner", "1"), "failed to get runner")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("1"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

// TestObserveRealErrorSurfaces: a non-404 failure must surface as an error
// (lesson #3).
func TestObserveRealErrorSurfaces(t *testing.T) {
	f := &fakeClient{getErr: &clients.APIError{StatusCode: 500, Body: "boom"}}
	e := &external{client: f}

	if _, err := e.Observe(context.Background(), newCR("1")); err == nil {
		t.Fatalf("expected 500 to surface as an error")
	}
}

// TestObserveAvailableAndUpToDate: an existing, matching runner must be marked
// Available and up-to-date (lesson #2/#6).
func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getRunner: &clients.Runner{ID: 1, Name: "my-runner", Labels: []string{"linux"}}}
	e := &external{client: f}

	cr := newCR("1")

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
// ResourceUpToDate:false while KEEPING Ready=Available (lesson #6).
func TestObserveDriftStillAvailable(t *testing.T) {
	f := &fakeClient{getRunner: &clients.Runner{ID: 1, Name: "my-runner", Labels: []string{"windows"}}}
	e := &external{client: f}

	cr := newCR("1")

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

// TestCreateSetsExternalName: Create must pin the external-name to the
// authoritative runner id returned by the backend (lesson #3/#7).
func TestCreateSetsExternalName(t *testing.T) {
	f := &fakeClient{createResp: &clients.Runner{ID: 42, Name: "my-runner"}}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "42" {
		t.Fatalf("expected external-name 42, got %q", got)
	}
}

// TestDeleteIdempotentOn404: a delete against an already-absent runner must
// succeed so the finalizer releases (lesson #16).
func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("runner", "1")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("1")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteRunner to have been called")
	}
}
