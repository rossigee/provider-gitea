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
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/organizationmember/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotOrganizationMember    = "managed resource is not an OrganizationMember custom resource"
	errGetOrganizationMember    = "failed to get organization member"
	errCreateOrganizationMember = "failed to add organization member"
	errUpdateOrganizationMember = "failed to update organization member"
	errDeleteOrganizationMember = "failed to remove organization member"
	errGetProviderConfig        = "failed to get provider config"
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
	cr, ok := mg.(*v2.OrganizationMember)
	if !ok {
		return nil, errors.New(errNotOrganizationMember)
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

// parseExternalName splits an "org/username" external name.
func parseExternalName(externalID string) (org, username string, err error) {
	parts := strings.Split(externalID, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid external-id format %q, expected org/username", externalID)
	}
	return parts[0], parts[1], nil
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "organizationmember.observe",
		tracing.SpanAttrs("organizationmember", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.OrganizationMember)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationMember)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	org, username, err := parseExternalName(externalID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	member, err := e.client.GetOrganizationMember(ctx, org, username)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetOrganizationMember)
	}

	// Update observed state
	cr.Status.AtProvider = v2.OrganizationMemberObservation{
		Username:   &member.Username,
		Role:       &member.Role,
		Visibility: &member.Visibility,
	}

	uo := isOrganizationMemberUpToDate(cr, member)

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func isOrganizationMemberUpToDate(cr *v2.OrganizationMember, member *clients.OrganizationMember) bool {
	fp := cr.Spec.ForProvider
	if fp.Role != member.Role {
		return false
	}
	if fp.Visibility != nil && *fp.Visibility != member.Visibility {
		return false
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "organizationmember.create",
		tracing.SpanAttrs("organizationmember", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.OrganizationMember)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationMember)
	}

	fp := cr.Spec.ForProvider
	if _, err := e.client.AddOrganizationMember(ctx, fp.Organization, fp.Username, &clients.AddOrganizationMemberRequest{
		Role: fp.Role,
	}); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateOrganizationMember)
	}

	// Set external name to org/username format for future lookups
	meta.SetExternalName(cr, fmt.Sprintf("%s/%s", fp.Organization, fp.Username))

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "organizationmember.update",
		tracing.SpanAttrs("organizationmember", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.OrganizationMember)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationMember)
	}

	org, username, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	fp := cr.Spec.ForProvider
	role := fp.Role
	if _, err := e.client.UpdateOrganizationMember(ctx, org, username, &clients.UpdateOrganizationMemberRequest{
		Role:       &role,
		Visibility: fp.Visibility,
	}); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganizationMember)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "organizationmember.delete",
		tracing.SpanAttrs("organizationmember", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.OrganizationMember)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganizationMember)
	}

	org, username, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-removed external resource.
	if _, err := e.client.GetOrganizationMember(ctx, org, username); err == nil {
		if err := e.client.RemoveOrganizationMember(ctx, org, username); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteOrganizationMember)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteOrganizationMember)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.OrganizationMemberKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.OrganizationMemberGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.OrganizationMember{}).
		Complete(r)
}
