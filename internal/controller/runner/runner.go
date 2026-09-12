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

package runner

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
	v2 "github.com/rossigee/provider-gitea/apis/runner/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotRunner         = "managed resource is not a Runner custom resource"
	errGetRunner         = "failed to get runner"
	errCreateRunner      = "failed to create runner"
	errUpdateRunner      = "failed to update runner"
	errDeleteRunner      = "failed to delete runner"
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
	cr, ok := mg.(*v2.Runner)
	if !ok {
		return nil, errors.New(errNotRunner)
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

// scopeValue returns the scope value string (empty for system scope).
func scopeValue(cr *v2.Runner) string {
	if cr.Spec.ForProvider.ScopeValue == nil {
		return ""
	}
	return *cr.Spec.ForProvider.ScopeValue
}

// parseExternalName splits a "scope/scopevalue.../id" external name.
// The scope value may itself contain slashes (owner/name repositories),
// so the first segment is the scope and the last is the numeric id.
func parseExternalName(externalID string) (scope, scopeValue string, id int64, err error) {
	parts := strings.Split(externalID, "/")
	if len(parts) < 2 {
		return "", "", 0, fmt.Errorf("invalid external-id format %q, expected scope/[scopevalue/]id", externalID)
	}
	if _, err := fmt.Sscanf(parts[len(parts)-1], "%d", &id); err != nil {
		return "", "", 0, fmt.Errorf("invalid external-id %q: %w", externalID, err)
	}
	scope = parts[0]
	if len(parts) > 2 {
		scopeValue = strings.Join(parts[1:len(parts)-1], "/")
	}
	return scope, scopeValue, id, nil
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "runner.observe",
		tracing.SpanAttrs("runner", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.Runner)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRunner)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	scope, scopeValue, id, err := parseExternalName(externalID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	runner, err := e.client.GetRunner(ctx, scope, scopeValue, id)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRunner)
	}

	// Update observed state
	cr.Status.AtProvider = v2.RunnerObservation{
		ID:     &runner.ID,
		Name:   &runner.Name,
		UUID:   &runner.UUID,
		Status: &runner.Status,
	}

	uo := isRunnerUpToDate(cr, runner)

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, s := range a {
		set[s] = struct{}{}
	}
	for _, s := range b {
		if _, ok := set[s]; !ok {
			return false
		}
	}
	return true
}

func isRunnerUpToDate(cr *v2.Runner, runner *clients.Runner) bool {
	fp := cr.Spec.ForProvider
	if fp.Name != runner.Name {
		return false
	}
	if !sameStrings(fp.Labels, runner.Labels) {
		return false
	}
	if fp.Description != nil && *fp.Description != runner.Description {
		return false
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "runner.create",
		tracing.SpanAttrs("runner", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.Runner)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRunner)
	}

	fp := cr.Spec.ForProvider
	req := &clients.CreateRunnerRequest{
		Name:          fp.Name,
		Labels:        fp.Labels,
		RunnerGroupID: fp.RunnerGroupID,
	}
	if fp.Description != nil {
		req.Description = *fp.Description
	}

	sv := scopeValue(cr)
	runner, err := e.client.CreateRunner(ctx, fp.Scope, sv, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRunner)
	}

	// Set external name to scope/[scopevalue/]id format for future lookups
	if sv == "" {
		meta.SetExternalName(cr, fmt.Sprintf("%s/%d", fp.Scope, runner.ID))
	} else {
		meta.SetExternalName(cr, fmt.Sprintf("%s/%s/%d", fp.Scope, sv, runner.ID))
	}

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "runner.update",
		tracing.SpanAttrs("runner", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.Runner)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRunner)
	}

	scope, scopeValue, id, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	fp := cr.Spec.ForProvider
	name := fp.Name
	_, err = e.client.UpdateRunner(ctx, scope, scopeValue, id, &clients.UpdateRunnerRequest{
		Name:          &name,
		Labels:        fp.Labels,
		Description:   fp.Description,
		RunnerGroupID: fp.RunnerGroupID,
	})
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRunner)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "runner.delete",
		tracing.SpanAttrs("runner", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.Runner)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRunner)
	}

	scope, scopeValue, id, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetRunner(ctx, scope, scopeValue, id); err == nil {
		if err := e.client.DeleteRunner(ctx, scope, scopeValue, id); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRunner)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRunner)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.RunnerKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RunnerGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Runner{}).
		Complete(r)
}
