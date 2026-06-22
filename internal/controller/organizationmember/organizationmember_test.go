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

package organizationmember

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/organizationmember/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the
// organization-member verbs this controller uses.
type fakeClient struct {
	clients.Client
	getMember *clients.OrganizationMember
	getErr    error
	added     *clients.AddOrganizationMemberRequest
	addErr    error
	updated   *clients.UpdateOrganizationMemberRequest
	deleteErr error
	deleted   bool
}

func (f *fakeClient) GetOrganizationMember(_ context.Context, _, _ string) (*clients.OrganizationMember, error) {
	return f.getMember, f.getErr
}

func (f *fakeClient) AddOrganizationMember(_ context.Context, _, _ string, req *clients.AddOrganizationMemberRequest) (*clients.OrganizationMember, error) {
	f.added = req
	return &clients.OrganizationMember{}, f.addErr
}

func (f *fakeClient) UpdateOrganizationMember(_ context.Context, _, _ string, req *clients.UpdateOrganizationMemberRequest) (*clients.OrganizationMember, error) {
	f.updated = req
	return &clients.OrganizationMember{}, nil
}

func (f *fakeClient) RemoveOrganizationMember(_ context.Context, _, _ string) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.OrganizationMember {
	cr := &v2.OrganizationMember{}
	cr.SetName("my-member")
	cr.Spec.ForProvider.Organization = "acme"
	cr.Spec.ForProvider.Username = "alice"
	cr.Spec.ForProvider.Role = "member"
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
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("organization member", "acme/alice"), "failed to get organization member")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("alice"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getMember: &clients.OrganizationMember{Username: "alice", Role: "member"}}
	e := &external{client: f}

	cr := newCR("alice")

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
	f := &fakeClient{}
	e := &external{client: f}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "alice" {
		t.Fatalf("expected external-name alice, got %q", got)
	}
	if f.added == nil || f.added.Role != "member" {
		t.Fatalf("expected Add called with role member, got %+v", f.added)
	}
}

func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("organization member", "acme/alice")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("alice")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected RemoveOrganizationMember to have been called")
	}
}
