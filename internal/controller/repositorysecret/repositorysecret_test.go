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

package repositorysecret

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/repositorysecret/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

// fakeClient is a hand-rolled clients.Client stub exposing only the
// repository-secret verbs this controller uses; every other method panics via
// the embedded nil interface so an accidental call is loud.
type fakeClient struct {
	clients.Client
	getErr    error
	createErr error
	deleteErr error
	deleted   bool
}

func (f *fakeClient) GetRepositorySecret(_ context.Context, _, _ string) (*clients.RepositorySecret, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &clients.RepositorySecret{Name: "MY_SECRET"}, nil
}

func (f *fakeClient) CreateRepositorySecret(_ context.Context, _, _ string, _ *clients.CreateRepositorySecretRequest) error {
	return f.createErr
}

func (f *fakeClient) UpdateRepositorySecret(_ context.Context, _, _ string, _ *clients.UpdateRepositorySecretRequest) error {
	return nil
}

func (f *fakeClient) DeleteRepositorySecret(_ context.Context, _, _ string) error {
	f.deleted = true
	return f.deleteErr
}

func newCR(externalName string) *v2.RepositorySecret {
	cr := &v2.RepositorySecret{}
	cr.SetName("my-secret")
	cr.Spec.ForProvider.Repository = "acme/my-repo"
	cr.Spec.ForProvider.SecretName = "MY_SECRET"
	cr.Spec.ForProvider.ValueSecretRef.Namespace = "default"
	cr.Spec.ForProvider.ValueSecretRef.Name = "secret-value"
	cr.Spec.ForProvider.ValueSecretRef.Key = "value"
	if externalName != "" {
		meta.SetExternalName(cr, externalName)
	}
	return cr
}

// kubeWithValue is a fake kube client seeded with the Secret that
// valueSecretRef points at (Create/Update read the secret value from it).
func kubeWithValue() client.Client {
	return fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "secret-value"},
		Data:       map[string][]byte{"value": []byte("s3cret")},
	}).Build()
}

func isAvailable(cr resource.Managed) bool {
	return cr.GetCondition(xpv1.TypeReady).Reason == xpv1.ReasonAvailable
}

func TestObserveNotCreated(t *testing.T) {
	f := &fakeClient{getErr: clients.NewNotFoundError("repository secret", "MY_SECRET")}
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
	f := &fakeClient{getErr: clients.NewNotFoundError("repository secret", "MY_SECRET")}
	e := &external{client: f}

	obs, err := e.Observe(context.Background(), newCR("MY_SECRET"))
	if err != nil {
		t.Fatalf("not-found must not surface as error, got %v", err)
	}
	if obs.ResourceExists {
		t.Fatalf("expected ResourceExists=false on 404")
	}
}

func TestObserveAvailableAndUpToDate(t *testing.T) {
	f := &fakeClient{}
	e := &external{client: f}

	cr := newCR("MY_SECRET")
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
	e := &external{client: f, kube: kubeWithValue()}

	cr := newCR("")
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := meta.GetExternalName(cr); got != "MY_SECRET" {
		t.Fatalf("expected external-name MY_SECRET, got %q", got)
	}
}

func TestDeleteIdempotentOn404(t *testing.T) {
	f := &fakeClient{deleteErr: clients.NewNotFoundError("repository secret", "MY_SECRET")}
	e := &external{client: f}

	if _, err := e.Delete(context.Background(), newCR("MY_SECRET")); err != nil {
		t.Fatalf("404 on delete must be treated as success, got %v", err)
	}
	if !f.deleted {
		t.Fatalf("expected DeleteRepositorySecret to have been called")
	}
}
