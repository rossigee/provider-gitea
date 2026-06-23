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

package variable

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/variable/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the variable
// verbs this controller uses; every other method panics (via the embedded nil
// interface) so an accidental call is loud. It records which scoped endpoint was
// hit and serves canned responses/errors.
type fakeClient struct {
	clients.Client

	getVar *clients.Variable
	getErr error

	// scope routing recorders
	orgCreate, repoCreate bool
	orgUpdate, repoUpdate bool
	orgDelete, repoDelete bool

	createErr error
	deleteErr error
}

func (f *fakeClient) GetOrganizationVariable(_ context.Context, _, _ string) (*clients.Variable, error) {
	return f.getVar, f.getErr
}

func (f *fakeClient) GetRepositoryVariable(_ context.Context, _, _, _ string) (*clients.Variable, error) {
	return f.getVar, f.getErr
}

func (f *fakeClient) CreateOrganizationVariable(_ context.Context, _, _ string, _ *clients.VariableRequest) error {
	f.orgCreate = true
	return f.createErr
}

func (f *fakeClient) CreateRepositoryVariable(_ context.Context, _, _, _ string, _ *clients.VariableRequest) error {
	f.repoCreate = true
	return f.createErr
}

func (f *fakeClient) UpdateOrganizationVariable(_ context.Context, _, _ string, _ *clients.VariableRequest) error {
	f.orgUpdate = true
	return nil
}

func (f *fakeClient) UpdateRepositoryVariable(_ context.Context, _, _, _ string, _ *clients.VariableRequest) error {
	f.repoUpdate = true
	return nil
}

func (f *fakeClient) DeleteOrganizationVariable(_ context.Context, _, _ string) error {
	f.orgDelete = true
	return f.deleteErr
}

func (f *fakeClient) DeleteRepositoryVariable(_ context.Context, _, _, _ string) error {
	f.repoDelete = true
	return f.deleteErr
}

func orgCR(externalName string) *v2.Variable {
	cr := &v2.Variable{}
	cr.SetName("my-var")
	cr.Spec.ForProvider.Name = "MY_VAR"
	cr.Spec.ForProvider.Value = "hello"
	cr.Spec.ForProvider.Organization = ptr.To("acme")
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func repoCR(externalName string) *v2.Variable {
	cr := &v2.Variable{}
	cr.SetName("my-var")
	cr.Spec.ForProvider.Name = "MY_VAR"
	cr.Spec.ForProvider.Value = "hello"
	cr.Spec.ForProvider.Repository = ptr.To("acme/my-repo")
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

	obs, err := e.Observe(context.Background(), orgCR(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false for empty external-name")
	}
}

// TestObserveNotFound: a typed 404 must be classified as absent, not surfaced.
func TestObserveNotFound(t *testing.T) {
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("variable", "MY_VAR"), "failed to get variable")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), orgCR("MY_VAR"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

// TestObserveValueMatchUpToDate: a live value equal to spec is up-to-date and
// Available. Variables are readable, so this is REAL drift detection.
func TestObserveValueMatchUpToDate(t *testing.T) {
	f := &fakeClient{getVar: &clients.Variable{ID: 5, Name: "MY_VAR", Data: "hello"}}
	e := &external{client: f}

	cr := orgCR("MY_VAR")
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

// TestObserveValueDrift: a live value differing from spec must report
// ResourceUpToDate:false while keeping Ready=Available (lesson #6).
func TestObserveValueDrift(t *testing.T) {
	f := &fakeClient{getVar: &clients.Variable{ID: 5, Name: "MY_VAR", Data: "stale"}}
	e := &external{client: f}

	cr := orgCR("MY_VAR") // spec value "hello"
	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Fatalf("expected value drift (ResourceUpToDate=false)")
	}
	if !isAvailable(cr) {
		t.Fatalf("drift must not downgrade Ready")
	}
}

// TestScopeValidation: exactly one of organization/repository must be set.
func TestScopeValidation(t *testing.T) {
	e := &external{client: &fakeClient{}}

	// neither set
	cr := &v2.Variable{}
	cr.Spec.ForProvider.Name = "MY_VAR"
	cr.Spec.ForProvider.Value = "x"
	if _, err := e.Create(context.Background(), cr); err == nil {
		t.Fatalf("expected scope error when neither org nor repo set")
	}

	// both set
	cr2 := orgCR("")
	cr2.Spec.ForProvider.Repository = ptr.To("acme/repo")
	if _, err := e.Create(context.Background(), cr2); err == nil {
		t.Fatalf("expected scope error when both org and repo set")
	}
}

// TestCreateOrgRoutesOrgEndpoint: org-scoped create hits the org endpoint and
// pins external-name from spec.
func TestCreateOrgRoutesOrgEndpoint(t *testing.T) {
	f := &fakeClient{}
	e := &external{client: f}

	cr := orgCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.orgCreate || f.repoCreate {
		t.Fatalf("expected org endpoint, got org=%v repo=%v", f.orgCreate, f.repoCreate)
	}
	if got := meta.GetExternalName(cr); got != "MY_VAR" {
		t.Fatalf("expected external-name MY_VAR, got %q", got)
	}
}

// TestCreateRepoRoutesRepoEndpoint: repo-scoped create hits the repo endpoint.
func TestCreateRepoRoutesRepoEndpoint(t *testing.T) {
	f := &fakeClient{}
	e := &external{client: f}

	if _, err := e.Create(context.Background(), repoCR("")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.repoCreate || f.orgCreate {
		t.Fatalf("expected repo endpoint, got org=%v repo=%v", f.orgCreate, f.repoCreate)
	}
}

// TestUpdateRoutesByScope: update reaches the scope-correct endpoint.
func TestUpdateRoutesByScope(t *testing.T) {
	f := &fakeClient{}
	e := &external{client: f}
	if _, err := e.Update(context.Background(), repoCR("MY_VAR")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.repoUpdate || f.orgUpdate {
		t.Fatalf("expected repo update, got org=%v repo=%v", f.orgUpdate, f.repoUpdate)
	}
}

// TestDeleteIdempotentOn404: delete against an absent variable succeeds.
func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("variable", "MY_VAR")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), orgCR("MY_VAR")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.orgDelete {
		t.Fatalf("expected DeleteOrganizationVariable to have been called")
	}
}
