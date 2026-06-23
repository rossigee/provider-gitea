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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

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
	lastUpdate *clients.UpdateUserRequest
	updateCnt  int
}

func (f *fakeClient) GetUser(_ context.Context, _ string) (*clients.User, error) {
	return f.getUser, f.getErr
}

func (f *fakeClient) CreateUser(_ context.Context, _ *clients.CreateUserRequest) (*clients.User, error) {
	return f.createResp, f.createErr
}

func (f *fakeClient) UpdateUser(_ context.Context, _ string, req *clients.UpdateUserRequest) (*clients.User, error) {
	f.lastUpdate = req
	f.updateCnt++
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
	cr.Spec.ForProvider.PasswordSecretRef = &xpv1.SecretKeySelector{
		SecretReference: xpv1.SecretReference{Namespace: "default", Name: "user-password"},
		Key:             "password",
	}
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

// kubeWithPassword is a fake kube client seeded with the Secret that
// passwordSecretRef points at (Create reads the password from it).
func kubeWithPassword() client.Client {
	return fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "user-password"},
		Data:       map[string][]byte{"password": []byte("s3cret")},
	}).Build()
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
	e := &external{client: f, kube: kubeWithPassword()}

	cr := newCR("my-user")
	// The provider has already applied the current Secret content -> no password drift.
	h := hashPassword("s3cret")
	cr.Status.AtProvider.PasswordHash = &h

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

// A never-applied password (empty stored hash) must read as drift so the managed
// reconciler calls Update to push it.
func TestObservePasswordDriftWhenHashUnset(t *testing.T) {
	f := &fakeClient{getUser: &clients.User{ID: 7, Username: "my-user", Email: "u@example.com"}}
	e := &external{client: f, kube: kubeWithPassword()}

	cr := newCR("my-user") // no PasswordHash in status
	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Fatalf("expected not-up-to-date when stored password hash is empty")
	}
}

// A changed Secret (stored hash != current content) must read as drift.
func TestObservePasswordDriftWhenSecretChanged(t *testing.T) {
	f := &fakeClient{getUser: &clients.User{ID: 7, Username: "my-user", Email: "u@example.com"}}
	e := &external{client: f, kube: kubeWithPassword()}

	cr := newCR("my-user")
	stale := hashPassword("old-password")
	cr.Status.AtProvider.PasswordHash = &stale

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Fatalf("expected not-up-to-date when Secret content differs from stored hash")
	}
}

// Update with a changed/unset hash pushes the password and advances the stored hash.
func TestUpdatePushesPasswordAndAdvancesHash(t *testing.T) {
	f := &fakeClient{getUser: &clients.User{ID: 7, Username: "my-user"}}
	e := &external{client: f, kube: kubeWithPassword()}

	cr := newCR("my-user") // empty stored hash -> rotation
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.lastUpdate == nil || f.lastUpdate.Password != "s3cret" {
		t.Fatalf("expected Update to push the password, got %+v", f.lastUpdate)
	}
	// Admin edit-user API requires login_name alongside password.
	if f.lastUpdate.LoginName == nil || *f.lastUpdate.LoginName != "my-user" {
		t.Fatalf("expected login_name defaulted to username, got %+v", f.lastUpdate.LoginName)
	}
	want := hashPassword("s3cret")
	if cr.Status.AtProvider.PasswordHash == nil || *cr.Status.AtProvider.PasswordHash != want {
		t.Fatalf("expected stored hash to advance to %q, got %+v", want, cr.Status.AtProvider.PasswordHash)
	}
}

// Update with a matching hash must not re-push the password (no spurious PATCH content).
func TestUpdateDoesNotRePushWhenHashMatches(t *testing.T) {
	f := &fakeClient{getUser: &clients.User{ID: 7, Username: "my-user"}}
	e := &external{client: f, kube: kubeWithPassword()}

	cr := newCR("my-user")
	h := hashPassword("s3cret")
	cr.Status.AtProvider.PasswordHash = &h

	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.lastUpdate == nil {
		t.Fatalf("expected UpdateUser to be called for other-field drift handling")
	}
	if f.lastUpdate.Password != "" {
		t.Fatalf("expected no password in PATCH when hash matches, got %q", f.lastUpdate.Password)
	}
}

func TestCreateSetsExternalNameFromBackend(t *testing.T) {
	f := &fakeClient{createResp: &clients.User{ID: 7, Username: "my-user"}}
	e := &external{client: f, kube: kubeWithPassword()}

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
