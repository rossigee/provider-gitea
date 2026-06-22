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

package adminuser

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/adminuser/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient embeds clients.Client so unimplemented methods panic if called.
type fakeClient struct {
	clients.Client
	getUser    *clients.AdminUser
	getErr     error
	createResp *clients.AdminUser
	createErr  error
	deleteErr  error
	deleted    bool
}

func (f *fakeClient) GetAdminUser(_ context.Context, _ string) (*clients.AdminUser, error) {
	return f.getUser, f.getErr
}

func (f *fakeClient) CreateAdminUser(_ context.Context, _ *clients.CreateAdminUserRequest) (*clients.AdminUser, error) {
	return f.createResp, f.createErr
}

func (f *fakeClient) UpdateAdminUser(_ context.Context, _ string, _ *clients.UpdateAdminUserRequest) (*clients.AdminUser, error) {
	return f.getUser, nil
}

func (f *fakeClient) DeleteAdminUser(_ context.Context, _ string) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.AdminUser {
	cr := &v2.AdminUser{}
	cr.SetName("my-admin")
	cr.Spec.ForProvider.Username = "my-admin"
	cr.Spec.ForProvider.Email = "a@example.com"
	cr.Spec.ForProvider.PasswordSecretRef = xpv1.SecretKeySelector{
		SecretReference: xpv1.SecretReference{Name: "pw", Namespace: "default"},
		Key:             "password",
	}
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
	f := &fakeClient{getErr: errors.Wrap(clients.NewNotFoundError("admin user", "my-admin"), "failed to get admin user")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("my-admin"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{getUser: &clients.AdminUser{ID: 7, Username: "my-admin", Email: "a@example.com"}}
	e := &external{client: f}

	cr := newCR("my-admin")
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
	f := &fakeClient{createResp: &clients.AdminUser{ID: 7, Username: "my-admin"}}
	kube := fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "pw", Namespace: "default"},
		Data:       map[string][]byte{"password": []byte("s3cret")},
	}).Build()
	e := &external{client: f, kube: kube}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "my-admin" {
		t.Fatalf("expected external-name my-admin, got %q", got)
	}
}

func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("admin user", "my-admin")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("my-admin")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteAdminUser to have been called")
	}
}
