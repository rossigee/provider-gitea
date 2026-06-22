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

package user

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/user/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient embeds clients.Client so unimplemented methods panic if called.
type fakeClient struct {
	clients.Client
	getUser    *clients.User
	getErr     error
	createResp *clients.User
	createErr  error
	deleteErr  error
	deleted    bool
}

func (f *fakeClient) GetUser(_ context.Context, _ string) (*clients.User, error) {
	return f.getUser, f.getErr
}

func (f *fakeClient) CreateUser(_ context.Context, _ *clients.CreateUserRequest) (*clients.User, error) {
	return f.createResp, f.createErr
}

func (f *fakeClient) UpdateUser(_ context.Context, _ string, _ *clients.UpdateUserRequest) (*clients.User, error) {
	return f.getUser, nil
}

func (f *fakeClient) DeleteUser(_ context.Context, _ string) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.User {
	cr := &v2.User{}
	cr.SetName("my-user")
	cr.Spec.ForProvider.Username = "my-user"
	cr.Spec.ForProvider.Email = "u@example.com"
	cr.Spec.ForProvider.Password = "s3cret"
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
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("user", "my-user"), "failed to get user")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("my-user"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getUser: &clients.User{ID: 7, Username: "my-user", Email: "u@example.com"}}
	e := &external{client: f}

	cr := newCR("my-user")
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

func TestCreateSetsExternalNameFromBackend(t *testing.T) {
	f := &fakeClient{createResp: &clients.User{ID: 7, Username: "my-user"}}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "my-user" {
		t.Fatalf("expected external-name my-user, got %q", got)
	}
}

func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("user", "my-user")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("my-user")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteUser to have been called")
	}
}
