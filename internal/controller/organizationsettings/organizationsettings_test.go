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
	"testing"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/organizationsettings/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the
// organization-settings verbs this controller uses; every other method panics
// (via the embedded nil interface) so an accidental call is loud.
type fakeClient struct {
	clients.Client
	getSettings *clients.OrganizationSettings
	getErr      error
	updated     *clients.UpdateOrganizationSettingsRequest
	updateResp  *clients.OrganizationSettings
	updateErr   error
}

func (f *fakeClient) GetOrganizationSettings(_ context.Context, _ string) (*clients.OrganizationSettings, error) {
	return f.getSettings, f.getErr
}

func (f *fakeClient) UpdateOrganizationSettings(_ context.Context, _ string, req *clients.UpdateOrganizationSettingsRequest) (*clients.OrganizationSettings, error) {
	f.updated = req
	return f.updateResp, f.updateErr
}

func newCR(externalName string) *v2.OrganizationSettings {
	cr := &v2.OrganizationSettings{}
	cr.SetName("acme-settings")
	cr.Spec.ForProvider.Organization = "acme"
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

// TestObserveNoOrg: an empty org (no identity) means "not created" — Observe
// must report ResourceExists:false without issuing a GET. (Adapted from the
// reference TestObserveNotCreated; OrganizationSettings keys identity off the
// org from spec, not an external-name, since it has no create call.)
func TestObserveNoOrg(t *testing.T) {
	f := &fakeClient{getErr: errors.New("GET should not be called")}
	e := &external{client: f}

	cr := newCR("")
	cr.Spec.ForProvider.Organization = ""

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false for empty org")
	}
}

// TestObserveNotFound: a typed 404 must be classified as absent via
// clients.IsNotFound, not surfaced as an error (lesson #3).
func TestObserveNotFound(t *testing.T) {
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("organization", "acme"), "failed to get organization settings")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("acme"))
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

	if _, err := e.Observe(context.Background(), newCR("acme")); err == nil {
		t.Fatalf("expected 500 to surface as an error")
	}
}

// TestObserveAvailableAndUpToDate: existing, matching settings must be marked
// Available and up-to-date (lesson #2/#6).
func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getSettings: &clients.OrganizationSettings{
		DefaultRepoPermission: "read",
		MembersCanCreateRepos: true,
	}}
	e := &external{client: f}

	cr := newCR("acme")
	cr.Spec.ForProvider.DefaultRepoPermission = ptr.To("read")
	cr.Spec.ForProvider.MembersCanCreateRepos = ptr.To(true)

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
	f := &fakeClient{getSettings: &clients.OrganizationSettings{
		DefaultRepoPermission: "read",
	}}
	e := &external{client: f}

	cr := newCR("acme")
	cr.Spec.ForProvider.DefaultRepoPermission = ptr.To("admin")

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

// TestCreateAppliesSettingsAndSetsExternalName: Create has no backend "create"
// call — it applies the desired settings via Update and pins external-name=org.
func TestCreateAppliesSettingsAndSetsExternalName(t *testing.T) {
	f := &fakeClient{updateResp: &clients.OrganizationSettings{}}
	e := &external{client: f}

	cr := newCR("")
	cr.Spec.ForProvider.DefaultRepoPermission = ptr.To("write")

	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.updated == nil {
		t.Fatalf("expected UpdateOrganizationSettings to have been called by Create")
	}
	if f.updated.DefaultRepoPermission == nil || *f.updated.DefaultRepoPermission != "write" {
		t.Fatalf("Create must apply the desired settings")
	}
	if got := meta.GetExternalName(cr); got != "acme" {
		t.Fatalf("expected external-name acme, got %q", got)
	}
}

// TestDeleteIsNoOp: OrganizationSettings cannot be deleted — Delete must be a
// no-op that succeeds (so the finalizer releases) and never calls the backend.
// (Adapted from the reference TestDeleteIdempotentOn404.)
func TestDeleteIsNoOp(t *testing.T) {
	f := &fakeClient{}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("acme")); err != nil {
		t.Fatalf("no-op delete must succeed, got %v", err)
	}
	if f.updated != nil {
		t.Fatalf("Delete must not call the backend")
	}
}
