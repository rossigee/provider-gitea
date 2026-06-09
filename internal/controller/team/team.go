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
	"strconv"
	"strings"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/team/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotTeam         = "managed resource is not a Team custom resource"
	errGetTeam         = "failed to get team"
	errCreateTeam      = "failed to create team"
	errUpdateTeam      = "failed to update team"
	errDeleteTeam      = "failed to delete team"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

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

	return &externalClient{client: conn}, nil
}

type externalClient struct {
	client clients.Client
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotTeam)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	teamID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to parse team ID")
	}

	team, err := e.client.GetTeam(ctx, teamID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetTeam)
	}

	cr.Status.AtProvider = v2.TeamObservation{
		ID:             &team.ID,
		OrganizationID: &team.Organization.ID,
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotTeam)
	}

	createReq := &clients.CreateTeamRequest{
		Name: cr.Spec.ForProvider.Name,
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
	if len(cr.Spec.ForProvider.Units) > 0 {
		createReq.Units = cr.Spec.ForProvider.Units
	}

	team, err := e.client.CreateTeam(ctx, cr.Spec.ForProvider.Organization, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateTeam)
	}

	meta.SetExternalName(cr, strconv.FormatInt(team.ID, 10))
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotTeam)
	}

	externalID := meta.GetExternalName(cr)
	teamID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to parse team ID")
	}

	updateReq := &clients.UpdateTeamRequest{}

	if cr.Spec.ForProvider.Description != nil {
		updateReq.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Permission != nil {
		updateReq.Permission = cr.Spec.ForProvider.Permission
	}
	if cr.Spec.ForProvider.CanCreateOrgRepo != nil {
		updateReq.CanCreateOrgRepo = cr.Spec.ForProvider.CanCreateOrgRepo
	}
	if cr.Spec.ForProvider.IncludesAllRepositories != nil {
		updateReq.IncludesAllRepositories = cr.Spec.ForProvider.IncludesAllRepositories
	}
	if len(cr.Spec.ForProvider.Units) > 0 {
		updateReq.Units = cr.Spec.ForProvider.Units
	}

	_, err = e.client.UpdateTeam(ctx, teamID, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateTeam)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Team)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotTeam)
	}

	externalID := meta.GetExternalName(cr)
	teamID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to parse team ID")
	}

	err = e.client.DeleteTeam(ctx, teamID)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteTeam)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.TeamKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.TeamGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Team{}).
		Complete(r)
}
