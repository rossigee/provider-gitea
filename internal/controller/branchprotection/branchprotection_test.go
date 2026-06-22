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

package branchprotection

import (
	"context"
	"testing"

	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/branchprotection/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the
// branch-protection verbs this controller uses.
type fakeClient struct {
	clients.Client
	getBP     *clients.BranchProtection
	getErr    error
	createErr error
	deleteErr error
	deleted   bool
}

func (f *fakeClient) GetBranchProtection(_ context.Context, _, _ string) (*clients.BranchProtection, error) {
	return f.getBP, f.getErr
}

func (f *fakeClient) CreateBranchProtection(_ context.Context, _, _ string, _ *clients.CreateBranchProtectionRequest) (*clients.BranchProtection, error) {
	return f.getBP, f.createErr
}

func (f *fakeClient) UpdateBranchProtection(_ context.Context, _, _ string, _ *clients.UpdateBranchProtectionRequest) (*clients.BranchProtection, error) {
	return f.getBP, nil
}

func (f *fakeClient) DeleteBranchProtection(_ context.Context, _, _ string) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.BranchProtection {
	cr := &v2.BranchProtection{}
	cr.SetName("my-bp")
	cr.Spec.ForProvider.Repository = "acme/my-repo"
	cr.Spec.ForProvider.Branch = "main"
	cr.Spec.ForProvider.RuleName = "main"
	cr.Spec.ForProvider.EnablePush = ptr.To(true)
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

func TestObserveNotCreated(t *testing.T) {
	f := &fakeClient{getErr: clients.NewNotFoundError("branch protection", "main")}
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
	f := &fakeClient{getErr: clients.NewNotFoundError("branch protection", "main")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("main"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getBP: &clients.BranchProtection{RuleName: "main", EnablePush: true}}
	e := &external{client: f}

	cr := newCR("main")
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
	f := &fakeClient{getBP: &clients.BranchProtection{RuleName: "main"}}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "main" {
		t.Fatalf("expected external-name main, got %q", got)
	}
}

func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("branch protection", "main")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("main")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteBranchProtection to have been called")
	}
}
