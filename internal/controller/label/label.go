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

// Package label implements the Crossplane managed-resource reconciler for the
// Gitea Label resource. It follows the canonical repository controller shape:
// Available() in Observe, typed not-found classification, real drift detection,
// external-name-as-identity, a non-nil rate limiter. Unlike repository (an
// owner/name resource), a Label is ID-keyed: the backend assigns a numeric
// int64 id which is pinned as the external-name; the parent (owner/repo) comes
// from the immutable cr.Spec.ForProvider.Repository field.
package label

import (
	"context"
	"strconv"
	"strings"

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

	v2 "github.com/rossigee/provider-gitea/apis/label/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotLabel          = "managed resource is not a Label custom resource"
	errGetLabel          = "failed to get label"
	errCreateLabel       = "failed to create label"
	errUpdateLabel       = "failed to update label"
	errDeleteLabel       = "failed to delete label"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected a numeric label id"
	errRepository        = "invalid repository, expected owner/name"
)

// Setup adds a controller that reconciles Label managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.LabelKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.LabelGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.Label{}).
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
	cr, ok := mg.(*v2.Label)
	if !ok {
		return nil, errors.New(errNotLabel)
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

// external observes/creates/updates/deletes the backend label.
type external struct {
	client clients.Client
}

// splitRepository parses the owner/name parent carried by the immutable
// cr.Spec.ForProvider.Repository field. Unlike repository, the parent does NOT
// come from the external-name (which holds the numeric label id).
func splitRepository(cr *v2.Label) (owner, repo string, ok bool) {
	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// labelID parses the numeric backend id pinned in the external-name. An empty
// or unparseable external-name means "not created yet" (ok=false).
func labelID(cr *v2.Label) (id int64, ok bool) {
	name := meta.GetExternalName(cr)
	if name == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(name, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Label)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotLabel)
	}

	id, ok := labelID(cr)
	if !ok {
		// No usable external-name yet -> not created. Don't try to GET it.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	owner, repo, ok := splitRepository(cr)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errRepository)
	}

	label, err := e.client.GetLabel(ctx, owner, repo, id)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure (auth/network/5xx) must surface so we
		// don't spuriously recreate an existing label.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetLabel)
	}

	cr.Status.AtProvider = v2.LabelObservation{
		ID:  &label.ID,
		URL: &label.URL,
	}

	upToDate := labelUpToDate(cr, label)

	// crossplane-runtime v2's managed reconciler no longer auto-sets
	// Available(): it only marks ReconcileSuccess. Readiness is the provider's
	// job (lesson #2/#6). Set Available on the exists path; drift is signalled
	// via ResourceUpToDate (surfaced as Synced), never by withholding Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// labelUpToDate reports whether the observed backend label matches the managed
// fields of the desired spec. Required fields (Name/Color) are always compared;
// a nil desired pointer (Description/Exclusive) means "do not manage", so it
// never causes drift.
func labelUpToDate(cr *v2.Label, observed *clients.Label) bool {
	p := cr.Spec.ForProvider
	if p.Name != observed.Name {
		return false
	}
	if p.Color != observed.Color {
		return false
	}
	if p.Description != nil && *p.Description != observed.Description {
		return false
	}
	if p.Exclusive != nil && *p.Exclusive != observed.Exclusive {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Label)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotLabel)
	}
	cr.SetConditions(xpv1.Creating())

	owner, repo, ok := splitRepository(cr)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errRepository)
	}

	createReq := &clients.CreateLabelRequest{
		Name:  cr.Spec.ForProvider.Name,
		Color: cr.Spec.ForProvider.Color,
	}
	if cr.Spec.ForProvider.Description != nil {
		createReq.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Exclusive != nil {
		createReq.Exclusive = *cr.Spec.ForProvider.Exclusive
	}

	label, err := e.client.CreateLabel(ctx, owner, repo, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateLabel)
	}

	// Capture the authoritative numeric id from the backend response and pin it
	// as the external name so every later Observe/Update/Delete resolves
	// identity from the annotation (lessons #3/#7/#14).
	meta.SetExternalName(cr, strconv.FormatInt(label.ID, 10))

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Label)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotLabel)
	}

	id, ok := labelID(cr)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	owner, repo, ok := splitRepository(cr)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errRepository)
	}

	updateReq := &clients.UpdateLabelRequest{
		Name:        &cr.Spec.ForProvider.Name,
		Color:       &cr.Spec.ForProvider.Color,
		Description: cr.Spec.ForProvider.Description,
		Exclusive:   cr.Spec.ForProvider.Exclusive,
	}

	if _, err := e.client.UpdateLabel(ctx, owner, repo, id, updateReq); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateLabel)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Label)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotLabel)
	}
	cr.SetConditions(xpv1.Deleting())

	id, ok := labelID(cr)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	owner, repo, ok := splitRepository(cr)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errRepository)
	}

	err := e.client.DeleteLabel(ctx, owner, repo, id)
	// Treat an already-absent label as a successful delete so the finalizer can
	// release (idempotent delete, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteLabel)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
