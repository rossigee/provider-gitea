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

package organizationmember

import (
	"context"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	organizationmemberv2 "github.com/rossigee/provider-gitea/apis/organizationmember/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotOrganizationMember      = "managed resource is not an OrganizationMember"
	errGetProviderConfig          = "cannot get ProviderConfig"
	errProviderNotReady           = "provider is not ready"
	errGetOrganizationMember      = "cannot get organization member"
	errAddOrganizationMember      = "cannot add organization member"
	errUpdateOrganizationMember   = "cannot update organization member"
	errRemoveOrganizationMember   = "cannot remove organization member"
	controllerName                = "organizationmembers.organizationmember.gitea.m.crossplane.io"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(organizationmemberv2.OrganizationMemberGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", "OrganizationMember")),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(controllerName))),
		managed.WithPollInterval(o.PollInterval),
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&organizationmemberv2.OrganizationMember{}).
		Complete(r)
}

type connector struct{ kube client.Client }
type external struct{ client giteaclients.Client }

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*organizationmemberv2.OrganizationMember)
	if !ok {
		return nil, errors.New(errNotOrganizationMember)
	}
	pcRef := cr.Spec.ProviderConfigReference
	if pcRef == nil {
		return nil, errors.New(errGetProviderConfig + ": providerConfigRef is required")
	}
	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: pcRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}
	if pc.Status.GetCondition(xpv1.TypeReady).Status != corev1.ConditionTrue {
		return nil, errors.New(errProviderNotReady)
	}
	cl, err := giteaclients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Gitea client")
	}
	return &external{client: cl}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*organizationmemberv2.OrganizationMember)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationMember)
	}
	member, err := e.client.GetOrganizationMember(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Username)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetOrganizationMember)
	}
	cr.Status.AtProvider = organizationmemberv2.OrganizationMemberObservation{
		Username: &member.Username,
		Role:     &member.Role,
		Visibility: &member.Visibility,
	}
	cr.Status.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: organizationMemberUpToDate(&cr.Spec.ForProvider, member),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*organizationmemberv2.OrganizationMember)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationMember)
	}
	req := &giteaclients.AddOrganizationMemberRequest{
		Role: cr.Spec.ForProvider.Role,
	}
	_, err := e.client.AddOrganizationMember(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errAddOrganizationMember)
	}
	meta.SetExternalName(cr, cr.Spec.ForProvider.Username)
	cr.Status.SetConditions(xpv1.Creating())
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*organizationmemberv2.OrganizationMember)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationMember)
	}
	req := &giteaclients.UpdateOrganizationMemberRequest{
		Role:       &cr.Spec.ForProvider.Role,
		Visibility: cr.Spec.ForProvider.Visibility,
	}
	_, err := e.client.UpdateOrganizationMember(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganizationMember)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*organizationmemberv2.OrganizationMember)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganizationMember)
	}
	err := e.client.RemoveOrganizationMember(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Username)
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errRemoveOrganizationMember)
	}
	cr.Status.SetConditions(xpv1.Deleting())
	return managed.ExternalDelete{}, nil
}

func organizationMemberUpToDate(desired *organizationmemberv2.OrganizationMemberParameters, actual *giteaclients.OrganizationMember) bool {
	if desired.Role != actual.Role {
		return false
	}
	if desired.Visibility != nil && *desired.Visibility != actual.Visibility {
		return false
	}
	return true
}
