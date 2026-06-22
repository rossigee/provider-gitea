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

// Package repositorykey implements the Crossplane managed-resource reconciler
// for the Gitea RepositoryKey (deploy key) resource. It follows the canonical
// repository controller: Available() in Observe, typed not-found
// classification, real drift detection, external-name-as-identity, and a
// non-nil rate limiter.
package repositorykey

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

	v2 "github.com/rossigee/provider-gitea/apis/repositorykey/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRepositoryKey    = "managed resource is not a RepositoryKey custom resource"
	errGetRepositoryKey    = "failed to get repositorykey"
	errCreateRepositoryKey = "failed to create repositorykey"
	errUpdateRepositoryKey = "failed to update repositorykey"
	errDeleteRepositoryKey = "failed to delete repositorykey"
	errGetProviderConfig   = "failed to get provider config"
	errExternalName        = "invalid external-name, expected numeric key id"
)

// Setup adds a controller that reconciles RepositoryKey managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.RepositoryKeyKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RepositoryKeyGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.RepositoryKey{}).
		// A non-nil rate limiter is mandatory: ratelimiter.Reconciler.When()
		// dereferences it every reconcile, so a nil limiter panics on the first
		// event. o.ForControllerRuntime() also carries MaxConcurrentReconciles
		// through; without WithOptions both are dropped.
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.RepositoryKey)
	if !ok {
		return nil, errors.New(errNotRepositoryKey)
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

// external observes/creates/updates/deletes the backend deploy key.
type external struct {
	client clients.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.RepositoryKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepositoryKey)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		// No usable external-name yet -> not created. Don't try to GET it.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	keyID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		// A non-numeric external-name is the default (metadata.name) before
		// Create has run — treat it as "not created yet" so Create fires and
		// pins the real numeric id, rather than erroring forever (lesson #7).
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	key, err := e.client.GetRepositoryKey(ctx, cr.Spec.ForProvider.Repository, keyID)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match. A
		// real failure (auth/network/5xx) must surface so we don't spuriously
		// recreate an existing key.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepositoryKey)
	}

	cr.Status.AtProvider = v2.RepositoryKeyObservation{
		ID:          &key.ID,
		Title:       &key.Title,
		Fingerprint: &key.Fingerprint,
		ReadOnly:    &key.ReadOnly,
		URL:         &key.URL,
		Repository:  &cr.Spec.ForProvider.Repository,
	}

	upToDate := repositoryKeyUpToDate(cr, key)

	// crossplane-runtime v2's managed reconciler no longer auto-sets
	// Available(): readiness is the provider's job. Set Available on the exists
	// path; drift is signalled via ResourceUpToDate, never by withholding Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// repositoryKeyUpToDate reports whether the observed backend key matches the
// managed fields of the desired spec. Only fields UpdateRepositoryKeyRequest
// can push are compared; a nil desired pointer means "do not manage", so it
// never causes drift. Key itself is immutable and not compared.
func repositoryKeyUpToDate(cr *v2.RepositoryKey, observed *clients.RepositoryKey) bool {
	p := cr.Spec.ForProvider
	if p.Title != observed.Title {
		return false
	}
	if p.ReadOnly != nil && *p.ReadOnly != observed.ReadOnly {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.RepositoryKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepositoryKey)
	}
	cr.SetConditions(xpv1.Creating())

	createReq := &clients.CreateRepositoryKeyRequest{
		Key:      cr.Spec.ForProvider.Key,
		Title:    cr.Spec.ForProvider.Title,
		ReadOnly: cr.Spec.ForProvider.ReadOnly,
	}

	key, err := e.client.CreateRepositoryKey(ctx, cr.Spec.ForProvider.Repository, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepositoryKey)
	}

	// Pin the backend-assigned numeric key id as the external name so every
	// later Observe/Update/Delete resolves identity from the annotation.
	meta.SetExternalName(cr, strconv.FormatInt(key.ID, 10))

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.RepositoryKey)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepositoryKey)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}
	keyID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errExternalName)
	}

	updateReq := &clients.UpdateRepositoryKeyRequest{
		Title:    &cr.Spec.ForProvider.Title,
		ReadOnly: cr.Spec.ForProvider.ReadOnly,
	}

	if _, err := e.client.UpdateRepositoryKey(ctx, cr.Spec.ForProvider.Repository, keyID, updateReq); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepositoryKey)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.RepositoryKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepositoryKey)
	}
	cr.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}
	keyID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errExternalName)
	}

	err = e.client.DeleteRepositoryKey(ctx, cr.Spec.ForProvider.Repository, keyID)
	// Treat an already-absent key as a successful delete so the finalizer can
	// release (idempotent delete).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepositoryKey)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
