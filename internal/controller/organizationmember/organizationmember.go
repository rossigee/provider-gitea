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

// Package organizationmember implements the Crossplane managed-resource
// reconciler for the Gitea OrganizationMember resource. It follows the
// canonical pattern documented in repository.go and
// crossplane-provider-template dev/docs/09-lessons-learned.md.
package organizationmember

import (
	"context"

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

	"github.com/rossigee/provider-gitea/apis/organizationmember/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotOrganizationMember = "managed resource is not an OrganizationMember custom resource"
	errGetMember             = "failed to get organization member"
	errAddMember             = "failed to add organization member"
	errUpdateMember          = "failed to update organization member"
	errRemoveMember          = "failed to remove organization member"
	errGetProviderConfig     = "failed to get provider config"
	errExternalName          = "invalid external-name, expected the member username"
)

// Setup adds a controller that reconciles OrganizationMember managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.OrganizationMemberKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.OrganizationMemberGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.OrganizationMember{}).
		// A non-nil rate limiter is mandatory (lesson #1).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.OrganizationMember)
	if !ok {
		return nil, errors.New(errNotOrganizationMember)
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

// external observes/creates/updates/deletes the backend membership.
type external struct {
	client clients.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.OrganizationMember)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationMember)
	}

	// Identity is the external-name (the username), authoritative for
	// Observe/Update/Delete (lesson #14). Empty -> not created, no GET.
	username := meta.GetExternalName(cr)
	if username == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	org := cr.Spec.ForProvider.Organization
	member, err := e.client.GetOrganizationMember(ctx, org, username)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetMember)
	}

	cr.Status.AtProvider = v2.OrganizationMemberObservation{
		Username:     &member.Username,
		Role:         &member.Role,
		Visibility:   &member.Visibility,
		Organization: &org,
	}

	// Set Available on the exists path; drift is carried by ResourceUpToDate
	// (lesson #2/#6).
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: memberUpToDate(cr, member),
	}, nil
}

// memberUpToDate compares the mutable membership fields (role, visibility) the
// provider pushes on Update.
func memberUpToDate(cr *v2.OrganizationMember, observed *clients.OrganizationMember) bool {
	if cr.Spec.ForProvider.Role != observed.Role {
		return false
	}
	if cr.Spec.ForProvider.Visibility != nil && *cr.Spec.ForProvider.Visibility != observed.Visibility {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.OrganizationMember)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationMember)
	}
	cr.SetConditions(xpv1.Creating())

	req := &clients.AddOrganizationMemberRequest{Role: cr.Spec.ForProvider.Role}
	if _, err := e.client.AddOrganizationMember(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Username, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errAddMember)
	}

	// Membership is keyed by username; pin it as the external-name.
	meta.SetExternalName(cr, cr.Spec.ForProvider.Username)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(_ context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// No-op: Gitea has no "update org member" endpoint — membership is add
	// (PUT /orgs/{org}/members/{user}) / remove (DELETE) only; there is no
	// PATCH on that path (it 404s). Role/visibility are fixed at add time, so
	// there is nothing to reconcile on update. (Discovered via e2e: the prior
	// PATCH-based Update failed every drift with "no route: PATCH …/members/…".)
	if _, ok := mg.(*v2.OrganizationMember); !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationMember)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.OrganizationMember)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganizationMember)
	}
	cr.SetConditions(xpv1.Deleting())

	username := meta.GetExternalName(cr)
	if username == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.RemoveOrganizationMember(ctx, cr.Spec.ForProvider.Organization, username)
	// An already-absent member is a successful delete (lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errRemoveMember)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
