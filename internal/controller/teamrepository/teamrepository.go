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

// Package teamrepository implements the Crossplane managed-resource
// reconciler for the Gitea TeamRepository resource — an association between a
// repository and a team. It follows the canonical pattern documented in
// repositorycollaborator.go and crossplane-provider-template
// dev/docs/09-lessons-learned.md.
package teamrepository

import (
	"context"
	"strings"

	"github.com/pkg/errors"
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

	v2 "github.com/rossigee/provider-gitea/apis/teamrepository/v1beta1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotTeamRepository = "managed resource is not a TeamRepository custom resource"
	errResolveTeam       = "failed to resolve team"
	errGetTeamRepo       = "failed to get team repository"
	errAddTeamRepo       = "failed to attach repository to team"
	errRemoveTeamRepo    = "failed to detach repository from team"
	errGetProviderConfig = "failed to get provider config"
	errTrackUsage        = "cannot track ProviderConfig usage"
)

// Setup adds a controller that reconciles TeamRepository managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.TeamRepositoryKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithPollJitterHook(o.PollInterval / 10),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}
	// Honour spec.managementPolicies (ObserveOnly, no-delete, pause, ...) when the
	// operator runs the provider with --enable-management-policies.
	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.TeamRepositoryGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.TeamRepository{}).
		// A non-nil rate limiter is mandatory (lesson #1).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube  client.Client
	usage resource.ModernTracker
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.TeamRepository)
	if !ok {
		return nil, errors.New(errNotTeamRepository)
	}

	if err := c.usage.Track(ctx, cr); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
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

// external observes/creates/updates/deletes the backend team-repository
// attachment.
type external struct {
	client clients.Client
}

// identity resolves (organization, team, repository) from spec, falling back
// to parsing the {org}/{team}/{repo} external-name for adoption of a
// pre-existing attachment whose spec fields are not yet fully populated.
func identity(cr *v2.TeamRepository) (org, team, repo string, ok bool) {
	org = cr.Spec.ForProvider.Organization
	team = cr.Spec.ForProvider.Team
	repo = cr.Spec.ForProvider.Repository
	if org != "" && team != "" && repo != "" {
		return org, team, repo, true
	}

	parts := strings.SplitN(meta.GetExternalName(cr), "/", 3)
	if len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != "" {
		return parts[0], parts[1], parts[2], true
	}

	return "", "", "", false
}

// externalName synthesizes the stable composite key {org}/{team}/{repo}.
func externalName(org, team, repo string) string {
	return org + "/" + team + "/" + repo
}

// resolveTeamID returns the numeric team id, honouring the optional teamId
// escape hatch before falling back to organization/team name resolution.
func (e *external) resolveTeamID(ctx context.Context, cr *v2.TeamRepository, org, team string) (int64, error) {
	if cr.Spec.ForProvider.TeamID != nil {
		return *cr.Spec.ForProvider.TeamID, nil
	}
	return clients.ResolveTeamID(ctx, e.client, org, team)
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.TeamRepository)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotTeamRepository)
	}

	org, team, repo, ok := identity(cr)
	if !ok {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	teamID, err := e.resolveTeamID(ctx, cr, org, team)
	if err != nil {
		// The team itself is absent -> not created; let Create run and surface
		// the error if the team truly doesn't exist.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errResolveTeam)
	}

	if _, err := e.client.GetTeamRepository(ctx, teamID, org, repo); err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3); real failures must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetTeamRepo)
	}

	cr.Status.AtProvider = v2.TeamRepositoryObservation{TeamID: &teamID}

	// Set Available on the exists path; the attachment carries no mutable
	// attributes, so it is always up to date once it exists (lesson #2/#6).
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.TeamRepository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotTeamRepository)
	}
	cr.SetConditions(xpv1.Creating())

	org, team, repo, ok := identity(cr)
	if !ok {
		return managed.ExternalCreation{}, errors.New("organization, team, and repository are required")
	}

	teamID, err := e.resolveTeamID(ctx, cr, org, team)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errResolveTeam)
	}

	if err := e.client.AddTeamRepository(ctx, teamID, org, repo); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errAddTeamRepo)
	}

	// The attachment is keyed by (org, team, repo); pin the composite key as
	// the external-name so every later Observe/Delete is self-describing.
	meta.SetExternalName(cr, externalName(org, team, repo))

	return managed.ExternalCreation{}, nil
}

// Update is a no-op: attachment is binary, so there is nothing to reconcile
// beyond existence.
func (e *external) Update(_ context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	if _, ok := mg.(*v2.TeamRepository); !ok {
		return managed.ExternalUpdate{}, errors.New(errNotTeamRepository)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.TeamRepository)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotTeamRepository)
	}
	cr.SetConditions(xpv1.Deleting())

	org, team, repo, ok := identity(cr)
	if !ok {
		// Nothing to identify -> nothing to delete.
		return managed.ExternalDelete{}, nil
	}

	teamID, err := e.resolveTeamID(ctx, cr, org, team)
	if err != nil {
		// An already-absent team means the attachment is moot (lesson #16).
		if clients.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, errors.Wrap(err, errResolveTeam)
	}

	err = e.client.RemoveTeamRepository(ctx, teamID, org, repo)
	// An already-detached repository is a successful delete.
	if err != nil && !clients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errRemoveTeamRepo)
	}
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
