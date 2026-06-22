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

// Package deploykey implements the Crossplane managed-resource reconciler for
// the Gitea DeployKey resource. It mirrors the canonical repository controller,
// adapted for an ID-keyed, immutable resource: the backend assigns an int64 id
// (pinned as the external-name), the parent owner/repo come from the immutable
// spec, and there is no Update verb (deploy keys cannot be mutated in Gitea).
package deploykey

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

	v2 "github.com/rossigee/provider-gitea/apis/deploykey/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotDeployKey      = "managed resource is not a DeployKey custom resource"
	errGetDeployKey      = "failed to get deploy key"
	errCreateDeployKey   = "failed to create deploy key"
	errDeleteDeployKey   = "failed to delete deploy key"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected numeric deploy-key id"
)

// Setup adds a controller that reconciles DeployKey managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.DeployKeyKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.DeployKeyGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.DeployKey{}).
		// A non-nil rate limiter is mandatory: ratelimiter.Reconciler.When()
		// dereferences it every reconcile, so a nil limiter panics on the first
		// event (lesson #1). o.ForControllerRuntime() also carries
		// MaxConcurrentReconciles through; without WithOptions both are dropped.
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.DeployKey)
	if !ok {
		return nil, errors.New(errNotDeployKey)
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

// external observes/creates/deletes the backend deploy key.
type external struct {
	client clients.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.DeployKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDeployKey)
	}

	// The external-name is the numeric deploy-key id. An empty/unparseable value
	// means "not created" — don't try to GET it.
	id, err := strconv.ParseInt(meta.GetExternalName(cr), 10, 64)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Parent owner/repo come from the immutable spec, not the external-name.
	owner := cr.Spec.ForProvider.Owner
	repo := cr.Spec.ForProvider.Repository

	key, err := e.client.GetDeployKey(ctx, owner, repo, id)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure (auth/network/5xx) must surface so we
		// don't spuriously recreate an existing deploy key.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetDeployKey)
	}

	cr.Status.AtProvider = v2.DeployKeyObservation{
		ID:          &key.ID,
		URL:         &key.URL,
		Fingerprint: &key.Fingerprint,
	}

	// crossplane-runtime v2's managed reconciler no longer auto-sets
	// Available(): readiness is the provider's job (lesson #2/#6). Set it on the
	// exists path.
	cr.SetConditions(xpv1.Available())

	// Deploy keys are immutable in Gitea — there are no mutable fields that can
	// drift, so the resource is always up-to-date once it exists.
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.DeployKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDeployKey)
	}
	cr.SetConditions(xpv1.Creating())

	owner := cr.Spec.ForProvider.Owner
	repo := cr.Spec.ForProvider.Repository

	createReq := &clients.CreateDeployKeyRequest{
		Title: cr.Spec.ForProvider.Title,
		Key:   cr.Spec.ForProvider.Key,
	}
	if cr.Spec.ForProvider.ReadOnly != nil {
		createReq.ReadOnly = *cr.Spec.ForProvider.ReadOnly
	}

	key, err := e.client.CreateDeployKey(ctx, owner, repo, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateDeployKey)
	}

	// Capture the backend-assigned id and pin it as the external-name so every
	// later Observe/Delete resolves identity from the annotation (lessons
	// #3/#7/#14).
	meta.SetExternalName(cr, strconv.FormatInt(key.ID, 10))

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(_ context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// No-op: there is no UpdateDeployKey method — deploy keys are immutable in
	// Gitea, so there is nothing to push and Observe always reports up-to-date.
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.DeployKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotDeployKey)
	}
	cr.SetConditions(xpv1.Deleting())

	id, err := strconv.ParseInt(meta.GetExternalName(cr), 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	owner := cr.Spec.ForProvider.Owner
	repo := cr.Spec.ForProvider.Repository

	err = e.client.DeleteDeployKey(ctx, owner, repo, id)
	// Treat an already-absent deploy key as a successful delete so the finalizer
	// can release (idempotent delete, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteDeployKey)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
