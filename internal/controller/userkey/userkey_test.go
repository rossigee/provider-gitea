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

package userkey

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/userkey/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the user-key
// verbs this controller uses.
type fakeClient struct {
	clients.Client
	getKey     *clients.UserKey
	getErr     error
	created    *clients.CreateUserKeyRequest
	createResp *clients.UserKey
	createErr  error
	deleteErr  error
	deleted    bool
}

func (f *fakeClient) GetUserKey(_ context.Context, _ string, _ int64) (*clients.UserKey, error) {
	return f.getKey, f.getErr
}

func (f *fakeClient) CreateUserKey(_ context.Context, _ string, req *clients.CreateUserKeyRequest) (*clients.UserKey, error) {
	f.created = req
	return f.createResp, f.createErr
}

func (f *fakeClient) DeleteUserKey(_ context.Context, _ string, _ int64) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.UserKey {
	cr := &v2.UserKey{}
	cr.SetName("my-key")
	cr.Spec.ForProvider.Username = "alice"
	cr.Spec.ForProvider.Title = "laptop"
	cr.Spec.ForProvider.Key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 alice@laptop"
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

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

func TestObserveNotFound(t *testing.T) {
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("user key", "alice/7"), "failed to get user key")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("7"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getKey: &clients.UserKey{ID: 7, Title: "laptop"}}
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

func TestCreateSetsExternalName(t *testing.T) {
	f := &fakeClient{createResp: &clients.UserKey{ID: 7, Title: "laptop"}}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "7" {
		t.Fatalf("expected external-name 7, got %q", got)
	}
	if f.created == nil || f.created.Title != "laptop" {
		t.Fatalf("expected Create called with title laptop, got %+v", f.created)
	}
}

func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("user key", "alice/7")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("7")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteUserKey to have been called")
	}
}
