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

// Package runner implements the Crossplane managed-resource reconciler for the
// Gitea Actions Runner resource. It follows the canonical pattern documented in
// internal/controller/repository/repository.go and the lessons in
// crossplane-provider-template dev/docs/09-lessons-learned.md.
package runner

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/runner/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRunner         = "managed resource is not a Runner custom resource"
	errGetRunner         = "failed to get runner"
	errCreateRunner      = "failed to create runner"
	errUpdateRunner      = "failed to update runner"
	errDeleteRunner      = "failed to delete runner"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected numeric runner id"
)

// Setup adds a controller that reconciles Runner managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.RunnerKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RunnerGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.Runner{}).
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
	cr, ok := mg.(*v2.Runner)
	if !ok {
		return nil, errors.New(errNotRunner)
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

// external observes/creates/updates/deletes the backend runner.
type external struct {
	client clients.Client
}

// runnerID parses the numeric runner id carried by the
// crossplane.io/external-name annotation. The annotation is authoritative for
// Observe, Update AND Delete (lesson #14). The scope/scopeValue always come
// from cr.Spec.ForProvider.
func runnerID(cr *v2.Runner) (int64, bool) {
	id, err := strconv.ParseInt(meta.GetExternalName(cr), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Runner)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRunner)
	}

	id, ok := runnerID(cr)
	if !ok {
		// No usable external-name yet -> not created. Don't try to GET it.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	scope := cr.Spec.ForProvider.Scope
	scopeValue := ptr.Deref(cr.Spec.ForProvider.ScopeValue, "")

	runner, err := e.client.GetRunner(ctx, scope, scopeValue, id)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). Real failures (auth/network/5xx) must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRunner)
	}

	cr.Status.AtProvider = v2.RunnerObservation{
		ID:              &runner.ID,
		Name:            &runner.Name,
		UUID:            &runner.UUID,
		Status:          &runner.Status,
		LastOnline:      &runner.LastOnline,
		CreatedAt:       &runner.CreatedAt,
		UpdatedAt:       &runner.UpdatedAt,
		Labels:          runner.Labels,
		Description:     &runner.Description,
		Scope:           &runner.Scope,
		ScopeValue:      &runner.ScopeValue,
		Version:         &runner.Version,
		Architecture:    &runner.Architecture,
		OperatingSystem: &runner.OperatingSystem,
		TokenExpiresAt:  &runner.TokenExpiresAt,
	}

	upToDate := runnerUpToDate(cr, runner)

	// crossplane-runtime v2's managed reconciler no longer auto-sets
	// Available(); readiness is the provider's job (lesson #2/#6). Set Available
	// on the exists path; drift is signalled via ResourceUpToDate, never by
	// withholding Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// runnerUpToDate reports whether the observed backend runner matches the
// managed fields of the desired spec. Only fields this provider pushes on
// Update are compared; a nil desired pointer means "do not manage".
func runnerUpToDate(cr *v2.Runner, observed *clients.Runner) bool {
	p := cr.Spec.ForProvider
	if p.Name != observed.Name {
		return false
	}
	if p.Description != nil && *p.Description != observed.Description {
		return false
	}
	if !equalStringSlices(p.Labels, observed.Labels) {
		return false
	}
	return true
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Runner)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRunner)
	}
	cr.SetConditions(xpv1.Creating())

	createReq := &clients.CreateRunnerRequest{
		Name:          cr.Spec.ForProvider.Name,
		Labels:        cr.Spec.ForProvider.Labels,
		RunnerGroupID: cr.Spec.ForProvider.RunnerGroupID,
	}
	if cr.Spec.ForProvider.Description != nil {
		createReq.Description = *cr.Spec.ForProvider.Description
	}

	scope := cr.Spec.ForProvider.Scope
	scopeValue := ptr.Deref(cr.Spec.ForProvider.ScopeValue, "")

	runner, err := e.client.CreateRunner(ctx, scope, scopeValue, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRunner)
	}

	// Capture the authoritative runner id from the backend response and pin it
	// as the external name (lesson #3/#7/#14).
	meta.SetExternalName(cr, strconv.FormatInt(runner.ID, 10))

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Runner)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRunner)
	}

	id, ok := runnerID(cr)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	updateReq := &clients.UpdateRunnerRequest{
		Name:          &cr.Spec.ForProvider.Name,
		Labels:        cr.Spec.ForProvider.Labels,
		Description:   cr.Spec.ForProvider.Description,
		RunnerGroupID: cr.Spec.ForProvider.RunnerGroupID,
	}

	scope := cr.Spec.ForProvider.Scope
	scopeValue := ptr.Deref(cr.Spec.ForProvider.ScopeValue, "")

	if _, err := e.client.UpdateRunner(ctx, scope, scopeValue, id, updateReq); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRunner)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Runner)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRunner)
	}
	cr.SetConditions(xpv1.Deleting())

	id, ok := runnerID(cr)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	scope := cr.Spec.ForProvider.Scope
	scopeValue := ptr.Deref(cr.Spec.ForProvider.ScopeValue, "")

	err := e.client.DeleteRunner(ctx, scope, scopeValue, id)
	// Treat an already-absent runner as a successful delete (idempotent, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRunner)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
