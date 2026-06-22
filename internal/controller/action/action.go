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

// Package action implements the Crossplane managed-resource reconciler for the
// Gitea Action (workflow) resource. It mirrors the canonical reference
// controller (internal/controller/repository/repository.go). See
// crossplane-provider-template dev/docs/09-lessons-learned.md.
package action

import (
	"context"
	"fmt"

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

	v2 "github.com/rossigee/provider-gitea/apis/action/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotAction         = "managed resource is not an Action custom resource"
	errGetAction         = "failed to get action"
	errCreateAction      = "failed to create action"
	errUpdateAction      = "failed to update action"
	errDeleteAction      = "failed to delete action"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected workflow name"
)

// Setup adds a controller that reconciles Action managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.ActionKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.ActionGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.Action{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.Action)
	if !ok {
		return nil, errors.New(errNotAction)
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
	cr, ok := mg.(*v2.Action)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAction)
	}

	// Identity is the external-name (the workflow name); the parent repository
	// is read from spec every reconcile (lesson #14). Empty external-name -> not
	// created yet; don't issue a GET.
	workflowName := meta.GetExternalName(cr)
	if workflowName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	action, err := e.client.GetAction(ctx, cr.Spec.ForProvider.Repository, workflowName)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure (auth/network/5xx) must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetAction)
	}

	cr.Status.AtProvider = v2.ActionObservation{
		WorkflowName: &action.WorkflowName,
		State:        &action.State,
		CreatedAt:    &action.CreatedAt,
		UpdatedAt:    &action.UpdatedAt,
		BadgeURL:     &action.Badge,
		Repository:   &cr.Spec.ForProvider.Repository,
	}

	// crossplane-runtime v2 no longer auto-sets Available() (lesson #2/#6).
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: actionUpToDate(cr, action),
	}, nil
}

// actionUpToDate compares the mutable workflow content this provider pushes on
// Update against the observed workflow file.
func actionUpToDate(cr *v2.Action, observed *clients.Action) bool {
	return cr.Spec.ForProvider.Content == observed.WorkflowFile.Content
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Action)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAction)
	}
	cr.SetConditions(xpv1.Creating())

	p := cr.Spec.ForProvider
	req := &clients.CreateActionRequest{
		WorkflowName: p.WorkflowName,
		WorkflowFile: p.Content,
		Path:         fmt.Sprintf(".github/workflows/%s", p.WorkflowName),
		Message:      ptr.Deref(p.CommitMessage, ""),
		Branch:       ptr.Deref(p.Branch, ""),
	}
	if _, err := e.client.CreateAction(ctx, p.Repository, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAction)
	}

	// For this resource the name (workflow name) IS the identity; pin it from
	// spec after a successful create so Observe/Update/Delete resolve from the
	// annotation.
	meta.SetExternalName(cr, p.WorkflowName)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Action)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAction)
	}

	workflowName := meta.GetExternalName(cr)
	if workflowName == "" {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	p := cr.Spec.ForProvider
	req := &clients.UpdateActionRequest{
		WorkflowFile: ptr.To(p.Content),
		Message:      p.CommitMessage,
		Branch:       p.Branch,
	}
	if _, err := e.client.UpdateAction(ctx, p.Repository, workflowName, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAction)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Action)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAction)
	}
	cr.SetConditions(xpv1.Deleting())

	workflowName := meta.GetExternalName(cr)
	if workflowName == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteAction(ctx, cr.Spec.ForProvider.Repository, workflowName)
	// An already-absent workflow is a successful delete (idempotent, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAction)
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}
