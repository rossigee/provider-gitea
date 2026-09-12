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

package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/action/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotAction         = "managed resource is not an Action custom resource"
	errGetAction         = "failed to get action workflow"
	errCreateAction      = "failed to create action workflow"
	errUpdateAction      = "failed to update action workflow"
	errDeleteAction      = "failed to delete action workflow"
	errGetProviderConfig = "failed to get provider config"
)

// A connector is expected to produce an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect returns an ExternalClient by:
// 1. Getting the provider config
// 2. Creating a Gitea API client
// 3. Returning an external client wrapping the API client
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.Action)
	if !ok {
		return nil, errors.New(errNotAction)
	}

	// Get provider config reference from spec
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

	return &externalClient{client: conn}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it matches the managed resource's desired state.
type externalClient struct {
	client clients.Client
}

// externalName builds the "owner/name/workflow" external name.
func externalName(repository, workflow string) string {
	return fmt.Sprintf("%s/%s", repository, workflow)
}

// parseExternalName splits an "owner/name/workflow" external name.
func parseExternalName(externalID string) (repository, workflow string, err error) {
	parts := strings.Split(externalID, "/")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid external-id format %q, expected owner/name/workflow", externalID)
	}
	return parts[0] + "/" + parts[1], parts[2], nil
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "action.observe",
		tracing.SpanAttrs("action", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.Action)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAction)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	repository, workflow, err := parseExternalName(externalID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	action, err := e.client.GetAction(ctx, repository, workflow)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetAction)
	}

	// Update observed state
	cr.Status.AtProvider = v2.ActionObservation{
		WorkflowName: &action.WorkflowName,
		State:        &action.State,
		CreatedAt:    &action.CreatedAt,
		UpdatedAt:    &action.UpdatedAt,
	}

	uo := isActionUpToDate(cr, action)

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func isActionUpToDate(cr *v2.Action, action *clients.Action) bool {
	if cr.Spec.ForProvider.Enabled == nil {
		return true
	}
	if *cr.Spec.ForProvider.Enabled {
		return action.State == "active"
	}
	return action.State != "active"
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "action.create",
		tracing.SpanAttrs("action", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.Action)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAction)
	}

	fp := cr.Spec.ForProvider
	req := &clients.CreateActionRequest{
		WorkflowName: fp.WorkflowName,
		WorkflowFile: fp.Content,
		Path:         fmt.Sprintf(".github/workflows/%s", fp.WorkflowName),
	}
	if fp.CommitMessage != nil {
		req.Message = *fp.CommitMessage
	}
	if fp.Branch != nil {
		req.Branch = *fp.Branch
	}

	if _, err := e.client.CreateAction(ctx, fp.Repository, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAction)
	}

	if fp.Enabled != nil && !*fp.Enabled {
		if err := e.client.DisableAction(ctx, fp.Repository, fp.WorkflowName); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateAction)
		}
	}

	// Set external name to owner/name/workflow format for future lookups
	meta.SetExternalName(cr, externalName(fp.Repository, fp.WorkflowName))

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "action.update",
		tracing.SpanAttrs("action", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.Action)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAction)
	}

	repository, workflow, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	fp := cr.Spec.ForProvider
	req := &clients.UpdateActionRequest{
		WorkflowFile: &fp.Content,
	}
	if fp.CommitMessage != nil {
		req.Message = fp.CommitMessage
	}
	if fp.Branch != nil {
		req.Branch = fp.Branch
	}

	if _, err := e.client.UpdateAction(ctx, repository, workflow, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAction)
	}

	if fp.Enabled != nil {
		if *fp.Enabled {
			if err := e.client.EnableAction(ctx, repository, workflow); err != nil {
				return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAction)
			}
		} else {
			if err := e.client.DisableAction(ctx, repository, workflow); err != nil {
				return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAction)
			}
		}
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "action.delete",
		tracing.SpanAttrs("action", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.Action)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAction)
	}

	repository, workflow, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetAction(ctx, repository, workflow); err == nil {
		if err := e.client.DeleteAction(ctx, repository, workflow); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAction)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAction)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.ActionKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.ActionGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Action{}).
		Complete(r)
}
