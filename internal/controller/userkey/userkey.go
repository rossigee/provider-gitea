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

// Package userkey implements the Crossplane managed-resource reconciler for the
// Gitea UserKey (SSH key) resource. It follows the canonical pattern documented
// in repository.go and crossplane-provider-template
// dev/docs/09-lessons-learned.md.
package userkey

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/userkey/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotUserKey        = "managed resource is not a UserKey custom resource"
	errGetKey            = "failed to get user key"
	errCreateKey         = "failed to create user key"
	errDeleteKey         = "failed to delete user key"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected the numeric key id"
)

// Setup adds a controller that reconciles UserKey managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.UserKeyKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.UserKeyGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.UserKey{}).
		// A non-nil rate limiter is mandatory (lesson #1).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.UserKey)
	if !ok {
		return nil, errors.New(errNotUserKey)
	}

	pcRef := cr.Spec.ProviderConfigReference
	if pcRef == nil {
		return nil, errors.New("providerConfigRef is required")
	}

	var pc v1beta1.ProviderConfig
	if err := c.kube.Get(ctx, client.ObjectKey{
		Namespace: cr.GetNamespace(),
		Name:      pcRef.Name,
	}, &pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	conn, err := clients.NewClient(ctx, &pc, c.kube)
	if err != nil {
		return nil, err
	}

	return &external{client: conn}, nil
}

// external observes/creates/deletes the backend SSH key.
type external struct {
	client clients.Client
}

// keyID parses the numeric key id carried by the crossplane.io/external-name
// annotation. It is authoritative for Observe/Delete (lesson #14). A zero/empty
// value means "not created yet" — never GET id 0 (lesson #7).
func keyID(cr *v2.UserKey) (int64, bool) {
	name := meta.GetExternalName(cr)
	if name == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(name, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.UserKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUserKey)
	}

	id, ok := keyID(cr)
	if !ok {
		// No usable external-name yet -> not created. Don't try to GET it.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	key, err := e.client.GetUserKey(ctx, cr.Spec.ForProvider.Username, id)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetKey)
	}

	cr.Status.AtProvider = v2.UserKeyObservation{
		ID:          &key.ID,
		Title:       &key.Title,
		Fingerprint: &key.Fingerprint,
		CreatedAt:   &key.CreatedAt,
		URL:         &key.URL,
		Username:    &cr.Spec.ForProvider.Username,
	}

	// Set Available on the exists path (lesson #2/#6).
	cr.SetConditions(xpv1.Available())

	// An SSH key is immutable on Gitea: the key material is write-once and can't
	// be read back, and title changes are not supported by the API, so the
	// resource is treated as up-to-date once it exists to avoid false drift.
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.UserKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUserKey)
	}
	cr.SetConditions(xpv1.Creating())

	req := &clients.CreateUserKeyRequest{
		Key:   cr.Spec.ForProvider.Key,
		Title: cr.Spec.ForProvider.Title,
	}
	key, err := e.client.CreateUserKey(ctx, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateKey)
	}

	// Pin the authoritative numeric id from the create response as the
	// external-name so Observe/Delete resolve identity from the annotation
	// (lesson #3/#14).
	meta.SetExternalName(cr, strconv.FormatInt(key.ID, 10))

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(_ context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// An SSH key has no meaningful in-place update: the key material is fixed at
	// create time. Observe always reports up-to-date, so Update is never driven
	// by drift; keep it a no-op.
	if _, ok := mg.(*v2.UserKey); !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUserKey)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.UserKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUserKey)
	}
	cr.SetConditions(xpv1.Deleting())

	id, ok := keyID(cr)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteUserKey(ctx, cr.Spec.ForProvider.Username, id)
	// An already-absent key is a successful delete (lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteKey)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
