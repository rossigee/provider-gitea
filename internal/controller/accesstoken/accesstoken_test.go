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

package accesstoken

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/accesstoken/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the
// access-token verbs this controller uses.
type fakeClient struct {
	clients.Client
	getToken   *clients.AccessToken
	getErr     error
	created    *clients.CreateAccessTokenRequest
	createResp *clients.AccessToken
	createErr  error
	deleteErr  error
	deleted    bool
}

func (f *fakeClient) GetAccessToken(_ context.Context, _ string, _ int64) (*clients.AccessToken, error) {
	return f.getToken, f.getErr
}

func (f *fakeClient) CreateAccessToken(_ context.Context, _ string, req *clients.CreateAccessTokenRequest) (*clients.AccessToken, error) {
	f.created = req
	return f.createResp, f.createErr
}

func (f *fakeClient) DeleteAccessToken(_ context.Context, _ string, _ int64) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.AccessToken {
	cr := &v2.AccessToken{}
	cr.SetName("my-token")
	cr.Spec.ForProvider.Username = "alice"
	cr.Spec.ForProvider.Name = "ci-token"
	cr.Spec.ForProvider.Scopes = []string{"read:repository"}
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
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("access token", "alice/7"), "failed to get access token")}
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
	f := &fakeClient{getToken: &clients.AccessToken{ID: 7, Name: "ci-token"}}
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
	// Forgejo/Gitea disclose the token value under the "sha1" field on create.
	f := &fakeClient{createResp: &clients.AccessToken{ID: 7, Name: "ci-token", Sha1: "secret-value"}}
	e := &external{client: f}

	cr := newCR("")
	obs, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "7" {
		t.Fatalf("expected external-name 7, got %q", got)
	}
	if string(obs.ConnectionDetails["token"]) != "secret-value" {
		t.Fatalf("expected token connection detail to be captured from sha1, got %q", obs.ConnectionDetails["token"])
	}
}

// TestCreateCapturesTokenFallback ensures that if a server ever populates the
// legacy "token" field instead of "sha1", the value is still captured.
func TestCreateCapturesTokenFallback(t *testing.T) {
	f := &fakeClient{createResp: &clients.AccessToken{ID: 8, Name: "ci-token", Token: "legacy-value"}}
	e := &external{client: f}

	obs, err := e.Create(context.Background(), newCR(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(obs.ConnectionDetails["token"]) != "legacy-value" {
		t.Fatalf("expected token connection detail from token fallback, got %q", obs.ConnectionDetails["token"])
	}
}

func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("access token", "alice/7")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("7")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteAccessToken to have been called")
	}
}
