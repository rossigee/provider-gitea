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

// Package repositorysecret implements the Crossplane managed-resource reconciler
// for the Gitea RepositorySecret resource. It mirrors the canonical reference
// controller (internal/controller/repository/repository.go): Available() in
// Observe, typed not-found classification, external-name-as-identity, a non-nil
// rate limiter. See crossplane-provider-template dev/docs/09-lessons-learned.md.
package repositorysecret

import (
	"context"

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

	v2 "github.com/rossigee/provider-gitea/apis/repositorysecret/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRepositorySecret    = "managed resource is not a RepositorySecret custom resource"
	errGetRepositorySecret    = "failed to get repositorysecret"
	errCreateRepositorySecret = "failed to create repositorysecret"
	errUpdateRepositorySecret = "failed to update repositorysecret"
	errDeleteRepositorySecret = "failed to delete repositorysecret"
	errGetProviderConfig      = "failed to get provider config"
	errExternalName           = "invalid external-name, expected secret name"
	errGetValue               = "failed to read secret value from valueSecretRef"
)

// Setup adds a controller that reconciles RepositorySecret managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.RepositorySecretKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RepositorySecretGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.RepositorySecret{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.RepositorySecret)
	if !ok {
		return nil, errors.New(errNotRepositorySecret)
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

	return &external{client: conn, kube: c.kube}, nil
}

type external struct {
	client clients.Client
	kube   client.Client
}

// resolveValue reads the secret value from the referenced Kubernetes Secret.
// Gitea requires a non-empty "data" on create/update (422 "[Data]: Required").
// The value is never taken from the spec (secret-ref convention).
func (e *external) resolveValue(ctx context.Context, cr *v2.RepositorySecret) (string, error) {
	if cr.Spec.ForProvider.ValueSecretRef == nil {
		return "", errors.New(errGetValue + ": valueSecretRef is required")
	}
	v, err := clients.ResolveSecretValue(ctx, e.kube, cr.Spec.ForProvider.ValueSecretRef)
	if err != nil {
		return "", errors.Wrap(err, errGetValue)
	}
	return v, nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.RepositorySecret)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepositorySecret)
	}

	// Identity is the external-name (the secret name); the parent repository is
	// read from spec every reconcile (lesson #14). Empty external-name -> not
	// created yet; don't issue a GET.
	secretName := meta.GetExternalName(cr)
	if secretName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	_, err := e.client.GetRepositorySecret(ctx, cr.Spec.ForProvider.Repository, secretName)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure (auth/network/5xx) must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepositorySecret)
	}

	cr.Status.AtProvider = v2.RepositorySecretObservation{
		SecretName: &secretName,
		Repository: &cr.Spec.ForProvider.Repository,
	}

	// crossplane-runtime v2 no longer auto-sets Available() (lesson #2/#6); set
	// it on the exists path.
	cr.SetConditions(xpv1.Available())

	// The secret VALUE is write-only — Gitea never returns it on GET — so it
	// cannot be diffed for drift. Treat the secret as always up-to-date; the
	// only mutable field (the value) is pushed unconditionally on Update.
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.RepositorySecret)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepositorySecret)
	}
	cr.SetConditions(xpv1.Creating())

	value, err := e.resolveValue(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	req := &clients.CreateRepositorySecretRequest{Data: value}
	if err := e.client.CreateRepositorySecret(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.SecretName, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepositorySecret)
	}

	// For this resource the name IS the identity; pin it from spec after a
	// successful create so Observe/Update/Delete resolve from the annotation.
	meta.SetExternalName(cr, cr.Spec.ForProvider.SecretName)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.RepositorySecret)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepositorySecret)
	}

	secretName := meta.GetExternalName(cr)
	if secretName == "" {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	value, err := e.resolveValue(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	req := &clients.UpdateRepositorySecretRequest{Data: value}
	if err := e.client.UpdateRepositorySecret(ctx, cr.Spec.ForProvider.Repository, secretName, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepositorySecret)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.RepositorySecret)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepositorySecret)
	}
	cr.SetConditions(xpv1.Deleting())

	secretName := meta.GetExternalName(cr)
	if secretName == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteRepositorySecret(ctx, cr.Spec.ForProvider.Repository, secretName)
	// An already-absent secret is a successful delete (idempotent, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepositorySecret)
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}
