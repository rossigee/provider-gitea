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

package deploykey

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/deploykey/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the deploy-key
// verbs this controller uses; every other method is inherited from the embedded
// interface (nil) so an accidental call panics loudly. It records the last
// request and serves canned responses/errors.
type fakeClient struct {
	clients.Client
	getKey     *clients.DeployKey
	getErr     error
	created    *clients.CreateDeployKeyRequest
	createResp *clients.DeployKey
	createErr  error
	deleteErr  error
	deleted    bool
}

func (f *fakeClient) GetDeployKey(_ context.Context, _, _ string, _ int64) (*clients.DeployKey, error) {
	return f.getKey, f.getErr
}

func (f *fakeClient) CreateDeployKey(_ context.Context, _, _ string, req *clients.CreateDeployKeyRequest) (*clients.DeployKey, error) {
	f.created = req
	return f.createResp, f.createErr
}

func (f *fakeClient) DeleteDeployKey(_ context.Context, _, _ string, _ int64) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.DeployKey {
	cr := &v2.DeployKey{}
	cr.SetName("my-deploy-key")
	cr.Spec.ForProvider.Owner = "acme"
	cr.Spec.ForProvider.Repository = "my-repo"
	cr.Spec.ForProvider.Title = "ci-key"
	cr.Spec.ForProvider.Key = "ssh-ed25519 AAAA..."
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
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("deploykey", "5"), "failed to get deploy key")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("5"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

// TestObserveAvailableAndUpToDate: an existing deploy key must be marked
// Available and up-to-date (deploy keys are immutable, so always up-to-date).
func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getKey: &clients.DeployKey{ID: 5, Title: "ci-key", Key: "ssh-ed25519 AAAA..."}}
	e := &external{client: f}

	cr := newCR("5")

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
// the backend-assigned numeric id, not a spec guess (lesson #3/#7).
func TestCreateSetsExternalNameFromBackend(t *testing.T) {
	f := &fakeClient{createResp: &clients.DeployKey{ID: 7, Title: "ci-key"}}
	e := &external{client: f}

	cr := newCR("")
	cr.Spec.ForProvider.ReadOnly = ptr.To(true)
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "7" {
		t.Fatalf("expected external-name 7, got %q", got)
	}
	if f.created == nil || f.created.Title != "ci-key" || !f.created.ReadOnly {
		t.Fatalf("expected create request built from spec, got %+v", f.created)
	}
}

// TestDeleteIdempotentOn404: a delete against an already-absent deploy key must
// succeed so the finalizer releases (lesson #16).
func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("deploykey", "5")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("5")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteDeployKey to have been called")
	}
}
