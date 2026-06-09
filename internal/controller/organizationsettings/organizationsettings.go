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

package organizationsettings

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/organizationsettings/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotOrganizationSettings         = "managed resource is not a OrganizationSettings custom resource"
	errGetOrganizationSettings         = "failed to get organizationsettings"
	errCreateOrganizationSettings      = "failed to create organizationsettings"
	errUpdateOrganizationSettings      = "failed to update organizationsettings"
	errDeleteOrganizationSettings      = "failed to delete organizationsettings"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return nil, errors.New(errNotOrganizationSettings)
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
	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationSettings)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	_, err := e.client.GetOrganizationSettings(ctx, cr.Spec.ForProvider.Organization)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetOrganizationSettings)
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationSettings)
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.Organization)
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.OrganizationSettings)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationSettings)
	}

	updateReq := &clients.UpdateOrganizationSettingsRequest{
		DefaultRepoPermission:    cr.Spec.ForProvider.DefaultRepoPermission,
		MembersCanCreateRepos:    cr.Spec.ForProvider.MembersCanCreateRepos,
		MembersCanCreatePrivate:  cr.Spec.ForProvider.MembersCanCreatePrivate,
		MembersCanCreateInternal: cr.Spec.ForProvider.MembersCanCreateInternal,
		MembersCanDeleteRepos:    cr.Spec.ForProvider.MembersCanDeleteRepos,
		MembersCanFork:           cr.Spec.ForProvider.MembersCanFork,
		MembersCanCreatePages:    cr.Spec.ForProvider.MembersCanCreatePages,
		DefaultRepoVisibility:    cr.Spec.ForProvider.DefaultRepoVisibility,
		RequireSignedCommits:     cr.Spec.ForProvider.RequireSignedCommits,
		EnableDependencyGraph:    cr.Spec.ForProvider.EnableDependencyGraph,
		AllowGitHooks:            cr.Spec.ForProvider.AllowGitHooks,
		AllowCustomGitHooks:      cr.Spec.ForProvider.AllowCustomGitHooks,
	}

	_, err := e.client.UpdateOrganizationSettings(ctx, cr.Spec.ForProvider.Organization, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganizationSettings)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.OrganizationSettingsKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.OrganizationSettingsGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.OrganizationSettings{}).
		Complete(r)
}
