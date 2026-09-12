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

package team

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
	v2 "github.com/rossigee/provider-gitea/apis/team/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"strconv"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotTeam           = "managed resource is not a Team custom resource"
	errGetTeam           = "failed to get team"
	errCreateTeam        = "failed to create team"
	errUpdateTeam        = "failed to update team"
	errDeleteTeam        = "failed to delete team"
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
	cr, ok := mg.(*v2.Team)
	if !ok {
		return nil, errors.New(errNotTeam)
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

// parseExternalName parses a numeric team id external name.
func parseExternalName(externalID string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(externalID), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid external-id %q, expected numeric team id: %w", externalID, err)
	}
	return id, nil
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "team.observe",
		tracing.SpanAttrs("team", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotTeam)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	id, err := parseExternalName(externalID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	team, err := e.client.GetTeam(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetTeam)
	}

	// Update observed state
	cr.Status.AtProvider = v2.TeamObservation{
		ID:             &team.ID,
		OrganizationID: &team.Organization.ID,
	}

	uo := isTeamUpToDate(cr, team)

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func isTeamUpToDate(cr *v2.Team, team *clients.Team) bool {
	fp := cr.Spec.ForProvider
	if fp.Name != team.Name {
		return false
	}
	if fp.Description != nil && *fp.Description != team.Description {
		return false
	}
	if fp.Permission != nil && *fp.Permission != team.Permission {
		return false
	}
	if fp.CanCreateOrgRepo != nil && *fp.CanCreateOrgRepo != team.CanCreateOrgRepo {
		return false
	}
	if fp.IncludesAllRepositories != nil && *fp.IncludesAllRepositories != team.IncludesAllRepositories {
		return false
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "team.create",
		tracing.SpanAttrs("team", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotTeam)
	}

	fp := cr.Spec.ForProvider
	req := &clients.CreateTeamRequest{
		Name: fp.Name,
	}
	if fp.Description != nil {
		req.Description = *fp.Description
	}
	if fp.Permission != nil {
		req.Permission = *fp.Permission
	}
	if fp.CanCreateOrgRepo != nil {
		req.CanCreateOrgRepo = *fp.CanCreateOrgRepo
	}
	if fp.IncludesAllRepositories != nil {
		req.IncludesAllRepositories = *fp.IncludesAllRepositories
	}
	req.Units = fp.Units

	team, err := e.client.CreateTeam(ctx, fp.Organization, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateTeam)
	}

	// Set external name to numeric team id for future lookups
	meta.SetExternalName(cr, strconv.FormatInt(team.ID, 10))

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "team.update",
		tracing.SpanAttrs("team", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotTeam)
	}

	id, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	fp := cr.Spec.ForProvider
	name := fp.Name
	_, err = e.client.UpdateTeam(ctx, id, &clients.UpdateTeamRequest{
		Name:                    &name,
		Description:             fp.Description,
		Permission:              fp.Permission,
		CanCreateOrgRepo:        fp.CanCreateOrgRepo,
		IncludesAllRepositories: fp.IncludesAllRepositories,
		Units:                   fp.Units,
	})
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateTeam)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "team.delete",
		tracing.SpanAttrs("team", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotTeam)
	}

	id, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetTeam(ctx, id); err == nil {
		if err := e.client.DeleteTeam(ctx, id); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteTeam)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteTeam)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.TeamKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.TeamGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Team{}).
		Complete(r)
}
