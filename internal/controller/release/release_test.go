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

package release

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/release/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the release
// verbs this controller uses; every other method is inherited from the embedded
// interface (and panics if called). It records the last request and serves
// canned responses/errors.
type fakeClient struct {
	clients.Client
	getRel     *clients.Release
	getErr     error
	created    *clients.CreateReleaseOptions
	createResp *clients.Release
	createErr  error
	updated    *clients.UpdateReleaseOptions
	deleteErr  error
	deleted    bool
}

func (f *fakeClient) GetRelease(_ context.Context, _, _ string, _ int64) (*clients.Release, error) {
	return f.getRel, f.getErr
}

func (f *fakeClient) CreateRelease(_ context.Context, _, _ string, req *clients.CreateReleaseOptions) (*clients.Release, error) {
	f.created = req
	return f.createResp, f.createErr
}

func (f *fakeClient) UpdateRelease(_ context.Context, _, _ string, _ int64, req *clients.UpdateReleaseOptions) (*clients.Release, error) {
	f.updated = req
	return f.getRel, nil
}

func (f *fakeClient) DeleteRelease(_ context.Context, _, _ string, _ int64) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.Release {
	cr := &v2.Release{}
	cr.SetName("my-release")
	cr.Spec.ForProvider.Owner = "acme"
	cr.Spec.ForProvider.Repository = "my-repo"
	cr.Spec.ForProvider.TagName = "v1.0.0"
	cr.Spec.ForProvider.Name = ptr.To("Release 1.0.0")
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

// TestObserveNotCreated: an empty external-name means "not created" — Observe
// must report ResourceExists:false without issuing a GET (lesson #7/#14).
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
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("release", "7"), "failed to get release")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("7"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

// TestObserveAvailableAndUpToDate: an existing, matching release must be marked
// Available and up-to-date (lesson #2/#6 — runtime v2 won't set Ready for us).
func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getRel: &clients.Release{ID: 7, TagName: "v1.0.0", Name: "Release 1.0.0"}}
	e := &external{client: f}

	cr := newCR("7")

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

// TestCreateSetsExternalNameFromBackend: Create must pin the external-name to
// the backend-assigned id, not a spec guess (lesson #3/#7).
func TestCreateSetsExternalNameFromBackend(t *testing.T) {
	f := &fakeClient{createResp: &clients.Release{ID: 7}}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "7" {
		t.Fatalf("expected external-name 7, got %q", got)
	}
}

// TestDeleteIdempotentOn404: a delete against an already-absent release must
// succeed so the finalizer releases (lesson #16).
func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("release", "7")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("7")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteRelease to have been called")
	}
}
