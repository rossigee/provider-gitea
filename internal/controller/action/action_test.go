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

package action

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/action/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the action
// verbs this controller uses.
type fakeClient struct {
	clients.Client
	getAction *clients.Action
	getErr    error
	createErr error
	deleteErr error
	deleted   bool
}

func (f *fakeClient) GetAction(_ context.Context, _, _ string) (*clients.Action, error) {
	return f.getAction, f.getErr
}

func (f *fakeClient) CreateAction(_ context.Context, _ string, _ *clients.CreateActionRequest) (*clients.Action, error) {
	return f.getAction, f.createErr
}

func (f *fakeClient) UpdateAction(_ context.Context, _, _ string, _ *clients.UpdateActionRequest) (*clients.Action, error) {
	return f.getAction, nil
}

func (f *fakeClient) DeleteAction(_ context.Context, _, _ string) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.Action {
	cr := &v2.Action{}
	cr.SetName("my-action")
	cr.Spec.ForProvider.Repository = "acme/my-repo"
	cr.Spec.ForProvider.WorkflowName = "ci.yml"
	cr.Spec.ForProvider.Content = "name: ci"
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

func TestObserveNotCreated(t *testing.T) {
	f := &fakeClient{getErr: clients.NewNotFoundError("action workflow", "ci.yml")}
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
	f := &fakeClient{getErr: clients.NewNotFoundError("action workflow", "ci.yml")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("ci.yml"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getAction: &clients.Action{
		WorkflowName: "ci.yml",
		WorkflowFile: clients.ActionWorkflowFile{Content: "name: ci"},
	}}
	e := &external{client: f}

	cr := newCR("ci.yml")
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
	f := &fakeClient{getAction: &clients.Action{WorkflowName: "ci.yml"}}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "ci.yml" {
		t.Fatalf("expected external-name ci.yml, got %q", got)
	}
}

func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("action workflow", "ci.yml")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("ci.yml")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteAction to have been called")
	}
}
