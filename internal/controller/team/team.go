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

// Package team implements the Crossplane managed-resource reconciler for the
// Gitea Team resource. It follows the canonical pattern of the repository
// controller (see internal/controller/repository/repository.go and
// crossplane-provider-template dev/docs/09-lessons-learned.md). Team is
// id-keyed: the external-name is the numeric team ID; the owning organization
// comes from cr.Spec.ForProvider.Organization.
package team

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

	v2 "github.com/rossigee/provider-gitea/apis/team/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotTeam           = "managed resource is not a Team custom resource"
	errGetTeam           = "failed to get team"
	errCreateTeam        = "failed to create team"
	errUpdateTeam        = "failed to update team"
	errDeleteTeam        = "failed to delete team"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected a numeric team id"
)

// Setup adds a controller that reconciles Team managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.TeamKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.TeamGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.Team{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.Team)
	if !ok {
		return nil, errors.New(errNotTeam)
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

// external observes/creates/updates/deletes the backend team.
type external struct {
	client clients.Client
}

// teamID parses the numeric team id carried by the external-name annotation. It
// is authoritative for Observe/Update/Delete (lesson #14). A missing or
// non-numeric value means "not created yet".
func teamID(cr *v2.Team) (int64, bool) {
	id, err := strconv.ParseInt(meta.GetExternalName(cr), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotTeam)
	}

	id, ok := teamID(cr)
	if !ok {
		// No usable external-name yet -> not created. Don't try to GET id 0.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	team, err := e.client.GetTeam(ctx, id)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetTeam)
	}

	cr.Status.AtProvider = v2.TeamObservation{
		ID:             &team.ID,
		OrganizationID: &team.Organization.ID,
	}

	upToDate := teamUpToDate(cr, team)

	// crossplane-runtime v2 no longer auto-sets Available(); set it on the exists
	// path (lesson #2/#6). Drift is carried by ResourceUpToDate, not Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// teamUpToDate compares only the mutable fields this provider pushes on Update;
// a nil desired pointer means "do not manage", so it never drifts.
func teamUpToDate(cr *v2.Team, observed *clients.Team) bool {
	p := cr.Spec.ForProvider
	if p.Name != "" && p.Name != observed.Name {
		return false
	}
	if p.Description != nil && *p.Description != observed.Description {
		return false
	}
	if p.Permission != nil && *p.Permission != observed.Permission {
		return false
	}
	if p.CanCreateOrgRepo != nil && *p.CanCreateOrgRepo != observed.CanCreateOrgRepo {
		return false
	}
	if p.IncludesAllRepositories != nil && *p.IncludesAllRepositories != observed.IncludesAllRepositories {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotTeam)
	}
	cr.SetConditions(xpv1.Creating())

	createReq := &clients.CreateTeamRequest{
		Name:  cr.Spec.ForProvider.Name,
		Units: cr.Spec.ForProvider.Units,
	}
	if cr.Spec.ForProvider.Description != nil {
		createReq.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Permission != nil {
		createReq.Permission = *cr.Spec.ForProvider.Permission
	}
	if cr.Spec.ForProvider.CanCreateOrgRepo != nil {
		createReq.CanCreateOrgRepo = *cr.Spec.ForProvider.CanCreateOrgRepo
	}
	if cr.Spec.ForProvider.IncludesAllRepositories != nil {
		createReq.IncludesAllRepositories = *cr.Spec.ForProvider.IncludesAllRepositories
	}

	team, err := e.client.CreateTeam(ctx, cr.Spec.ForProvider.Organization, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateTeam)
	}

	// Pin external-name to the authoritative numeric team id from the backend
	// response (lesson #3/#7/#14), never a spec guess.
	meta.SetExternalName(cr, strconv.FormatInt(team.ID, 10))

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotTeam)
	}

	id, ok := teamID(cr)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	updateReq := &clients.UpdateTeamRequest{
		Description:             cr.Spec.ForProvider.Description,
		Permission:              cr.Spec.ForProvider.Permission,
		CanCreateOrgRepo:        cr.Spec.ForProvider.CanCreateOrgRepo,
		IncludesAllRepositories: cr.Spec.ForProvider.IncludesAllRepositories,
		Units:                   cr.Spec.ForProvider.Units,
	}
	if cr.Spec.ForProvider.Name != "" {
		updateReq.Name = &cr.Spec.ForProvider.Name
	}

	if _, err := e.client.UpdateTeam(ctx, id, updateReq); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateTeam)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotTeam)
	}
	cr.SetConditions(xpv1.Deleting())

	id, ok := teamID(cr)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteTeam(ctx, id)
	// Treat an already-absent team as a successful delete (lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteTeam)
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}
