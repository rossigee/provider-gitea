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

package repositorykey

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/repositorykey/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the repository
// key verbs this controller uses; every other method falls through to the
// embedded nil interface (panics) so an accidental call is loud. It records the
// last request and serves canned responses/errors.
type fakeClient struct {
	clients.Client
	getKey     *clients.RepositoryKey
	getErr     error
	created    *clients.CreateRepositoryKeyRequest
	createResp *clients.RepositoryKey
	createErr  error
	updated    *clients.UpdateRepositoryKeyRequest
	deleteErr  error
	deleted    bool
}

func (f *fakeClient) GetRepositoryKey(_ context.Context, _ string, _ int64) (*clients.RepositoryKey, error) {
	return f.getKey, f.getErr
}

func (f *fakeClient) CreateRepositoryKey(_ context.Context, _ string, req *clients.CreateRepositoryKeyRequest) (*clients.RepositoryKey, error) {
	f.created = req
	return f.createResp, f.createErr
}

func (f *fakeClient) UpdateRepositoryKey(_ context.Context, _ string, _ int64, req *clients.UpdateRepositoryKeyRequest) (*clients.RepositoryKey, error) {
	f.updated = req
	return f.getKey, nil
}

func (f *fakeClient) DeleteRepositoryKey(_ context.Context, _ string, _ int64) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.RepositoryKey {
	cr := &v2.RepositoryKey{}
	cr.SetName("my-key")
	cr.Spec.ForProvider.Repository = "acme/my-repo"
	cr.Spec.ForProvider.Key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIabc deploy"
	cr.Spec.ForProvider.Title = "deploy-key"
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

// TestObserveNotCreated: an empty external-name means "not created" — Observe
// must report ResourceExists:false without issuing a GET.
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
// absent via clients.IsNotFound, not surfaced as an error.
func TestObserveNotFound(t *testing.T) {
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("repositorykey", "7"), "failed to get repositorykey")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("7"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

// TestObserveAvailableAndUpToDate: an existing, matching key must be marked
// Available and up-to-date (runtime v2 won't set Ready for us).
func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getKey: &clients.RepositoryKey{
		ID: 7, Key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIabc deploy",
		Title: "deploy-key", ReadOnly: true,
	}}
	e := &external{client: f}

	cr := newCR("7")
	cr.Spec.ForProvider.ReadOnly = ptr.To(true)

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
// the backend-assigned numeric key id, not a spec guess.
func TestCreateSetsExternalNameFromBackend(t *testing.T) {
	f := &fakeClient{createResp: &clients.RepositoryKey{ID: 7, Title: "deploy-key"}}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "7" {
		t.Fatalf("expected external-name 7, got %q", got)
	}
}

// TestDeleteIdempotentOn404: a delete against an already-absent key must
// succeed so the finalizer releases.
func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("repositorykey", "7")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("7")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteRepositoryKey to have been called")
	}
}
