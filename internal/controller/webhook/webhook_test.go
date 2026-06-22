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

package webhook

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/webhook/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the webhook
// verbs this controller uses; every other method panics so an accidental call
// is loud. It records the last request, which variant (repo/org) was hit, and
// serves canned responses/errors.
type fakeClient struct {
	clients.Client
	getWebhook *clients.Webhook
	getErr     error
	createResp *clients.Webhook
	createErr  error
	deleteErr  error

	// call tracking
	repoGetCalled    bool
	orgGetCalled     bool
	repoCreateCalled bool
	orgCreateCalled  bool
	repoDeleteCalled bool
	orgDeleteCalled  bool

	created *clients.CreateWebhookRequest
	updated *clients.UpdateWebhookRequest
}

func (f *fakeClient) GetRepositoryWebhook(_ context.Context, _, _ string, _ int64) (*clients.Webhook, error) {
	f.repoGetCalled = true
	return f.getWebhook, f.getErr
}

func (f *fakeClient) GetOrganizationWebhook(_ context.Context, _ string, _ int64) (*clients.Webhook, error) {
	f.orgGetCalled = true
	return f.getWebhook, f.getErr
}

func (f *fakeClient) CreateRepositoryWebhook(_ context.Context, _, _ string, req *clients.CreateWebhookRequest) (*clients.Webhook, error) {
	f.repoCreateCalled = true
	f.created = req
	return f.createResp, f.createErr
}

func (f *fakeClient) CreateOrganizationWebhook(_ context.Context, _ string, req *clients.CreateWebhookRequest) (*clients.Webhook, error) {
	f.orgCreateCalled = true
	f.created = req
	return f.createResp, f.createErr
}

func (f *fakeClient) UpdateRepositoryWebhook(_ context.Context, _, _ string, _ int64, req *clients.UpdateWebhookRequest) (*clients.Webhook, error) {
	f.updated = req
	return f.getWebhook, nil
}

func (f *fakeClient) UpdateOrganizationWebhook(_ context.Context, _ string, _ int64, req *clients.UpdateWebhookRequest) (*clients.Webhook, error) {
	f.updated = req
	return f.getWebhook, nil
}

func (f *fakeClient) DeleteRepositoryWebhook(_ context.Context, _, _ string, _ int64) error {
	f.repoDeleteCalled = true
	return f.deleteErr
}

func (f *fakeClient) DeleteOrganizationWebhook(_ context.Context, _ string, _ int64) error {
	f.orgDeleteCalled = true
	return f.deleteErr
}

// newCR builds a repository-variant webhook (owner + repository set).
func newCR(externalName string) *v2.Webhook {
	cr := &v2.Webhook{}
	cr.SetName("my-webhook")
	cr.Spec.ForProvider.Owner = ptr.To("acme")
	cr.Spec.ForProvider.Repository = ptr.To("my-repo")
	cr.Spec.ForProvider.URL = "https://example.com/hook"
	cr.Spec.ForProvider.Type = ptr.To("gitea")
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

// newOrgCR builds an organization-variant webhook (organization set).
func newOrgCR(externalName string) *v2.Webhook {
	cr := &v2.Webhook{}
	cr.SetName("my-webhook")
	cr.Spec.ForProvider.Organization = ptr.To("acme-org")
	cr.Spec.ForProvider.URL = "https://example.com/hook"
	cr.Spec.ForProvider.Type = ptr.To("gitea")
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
	if f.repoGetCalled || f.orgGetCalled {
		t.Fatalf("no GET must be issued for an empty external-name")
	}
}

// TestObserveNotFound: a typed 404 from the backend must be classified as
// absent via clients.IsNotFound, not surfaced as an error (lesson #3).
func TestObserveNotFound(t *testing.T) {
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("webhook", "7"), "failed to get webhook")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("7"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

// TestObserveAvailableAndUpToDate: an existing, matching webhook must be marked
// Available and up-to-date (lesson #2/#6 — runtime v2 won't set Ready for us).
func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getWebhook: &clients.Webhook{
		ID:     7,
		Type:   "gitea",
		Active: true,
		Events: []string{"push"},
	}}
	e := &external{client: f}

	cr := newCR("7")
	cr.Spec.ForProvider.Active = ptr.To(true)
	cr.Spec.ForProvider.Events = []string{"push"}

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
	if !f.repoGetCalled {
		t.Fatalf("repo-variant GET must be used when organization is unset")
	}
}

// TestObserveOrgVariant: when spec.forProvider.organization is set, Observe must
// resolve identity via the ORG client method, not the repo one.
func TestObserveOrgVariant(t *testing.T) {
	f := &fakeClient{getWebhook: &clients.Webhook{ID: 7, Type: "gitea", Active: true, Events: []string{"push"}}}
	e := &external{client: f}

	cr := newOrgCR("7")
	cr.Spec.ForProvider.Active = ptr.To(true)
	cr.Spec.ForProvider.Events = []string{"push"}

	if _, err := e.Observe(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.orgGetCalled {
		t.Fatalf("org-variant GET must be used when organization is set")
	}
	if f.repoGetCalled {
		t.Fatalf("repo-variant GET must NOT be used when organization is set")
	}
}

// TestCreateSetsExternalNameFromBackend: Create must pin the external-name to
// the backend-assigned numeric id, not a spec guess (lesson #3/#7).
func TestCreateSetsExternalNameFromBackend(t *testing.T) {
	f := &fakeClient{createResp: &clients.Webhook{ID: 7, Type: "gitea"}}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "7" {
		t.Fatalf("expected external-name 7, got %q", got)
	}
	if !f.repoCreateCalled {
		t.Fatalf("repo-variant Create must be used when organization is unset")
	}
}

// TestDeleteIdempotentOn404: a delete against an already-absent webhook must
// succeed so the finalizer releases (lesson #16).
func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("webhook", "7")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("7")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.repoDeleteCalled {
		t.Fatalf("expected DeleteRepositoryWebhook to have been called")
	}
}
