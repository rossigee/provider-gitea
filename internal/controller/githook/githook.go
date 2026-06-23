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

// Package githook implements the Crossplane managed-resource reconciler for the
// Gitea GitHook resource. It mirrors the canonical reference controller
// (internal/controller/repository/repository.go). See
// crossplane-provider-template dev/docs/09-lessons-learned.md.
package githook

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/githook/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotGitHook        = "managed resource is not a GitHook custom resource"
	errGetGitHook        = "failed to get githook"
	errCreateGitHook     = "failed to create githook"
	errUpdateGitHook     = "failed to update githook"
	errDeleteGitHook     = "failed to delete githook"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected hook type"
)

// Setup adds a controller that reconciles GitHook managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.GitHookKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}
	// Honour spec.managementPolicies (ObserveOnly, no-delete, pause, ...) when the
	// operator runs the provider with --enable-management-policies.
	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.GitHookGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.GitHook{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.GitHook)
	if !ok {
		return nil, errors.New(errNotGitHook)
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

type external struct {
	client clients.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.GitHook)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotGitHook)
	}

	// Identity is the external-name (the hook type); the parent repository is
	// read from spec every reconcile (lesson #14). Empty external-name -> not
	// created yet; don't issue a GET.
	hookType := meta.GetExternalName(cr)
	if hookType == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	hook, err := e.client.GetGitHook(ctx, cr.Spec.ForProvider.Repository, hookType)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure (auth/network/5xx) must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetGitHook)
	}

	cr.Status.AtProvider = v2.GitHookObservation{
		Name: &hook.Name,
	}

	// crossplane-runtime v2 no longer auto-sets Available() (lesson #2/#6).
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: gitHookUpToDate(cr, hook),
	}, nil
}

// gitHookUpToDate compares the mutable fields this provider pushes on Update
// (content + active flag) against the observed hook.
func gitHookUpToDate(cr *v2.GitHook, observed *clients.GitHook) bool {
	if cr.Spec.ForProvider.Content != observed.Content {
		return false
	}
	if ptr.Deref(cr.Spec.ForProvider.IsActive, true) != observed.IsActive {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.GitHook)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotGitHook)
	}
	cr.SetConditions(xpv1.Creating())

	req := &clients.CreateGitHookRequest{
		HookType: cr.Spec.ForProvider.HookType,
		Content:  cr.Spec.ForProvider.Content,
		IsActive: ptr.Deref(cr.Spec.ForProvider.IsActive, true),
	}
	if _, err := e.client.CreateGitHook(ctx, cr.Spec.ForProvider.Repository, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateGitHook)
	}

	// For this resource the name (hook type) IS the identity; pin it from spec
	// after a successful create so Observe/Update/Delete resolve from the
	// annotation.
	meta.SetExternalName(cr, cr.Spec.ForProvider.HookType)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.GitHook)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotGitHook)
	}

	hookType := meta.GetExternalName(cr)
	if hookType == "" {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	req := &clients.UpdateGitHookRequest{
		Content:  cr.Spec.ForProvider.Content,
		IsActive: ptr.Deref(cr.Spec.ForProvider.IsActive, true),
	}
	if _, err := e.client.UpdateGitHook(ctx, cr.Spec.ForProvider.Repository, hookType, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateGitHook)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.GitHook)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotGitHook)
	}
	cr.SetConditions(xpv1.Deleting())

	hookType := meta.GetExternalName(cr)
	if hookType == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteGitHook(ctx, cr.Spec.ForProvider.Repository, hookType)
	// An already-absent hook is a successful delete (idempotent, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteGitHook)
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}
