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

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/organizationmember/v1alpha1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotOrganizationMember    = "managed resource is not a OrganizationMember custom resource"
	errTrackPCUsage             = "cannot track ProviderConfig usage"
	errGetPC                    = "cannot get ProviderConfig"
	errGetCreds                 = "cannot get credentials"
	errNewClient                = "cannot create new Service"
	errGetOrganizationMember    = "cannot get organization member"
	errAddOrganizationMember    = "cannot add organization member"
	errUpdateOrganizationMember = "cannot update organization member"
	errRemoveOrganizationMember = "cannot remove organization member"
)

// Setup adds a controller that reconciles OrganizationMember managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.OrganizationMemberKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.OrganizationMemberGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.OrganizationMember{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.OrganizationMember)
	if !ok {
		return nil, errors.New(errNotOrganizationMember)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	client, err := giteaclients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: client}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client giteaclients.Client
}

func (c *external) Disconnect(ctx context.Context) error {
	// No persistent connection to disconnect
	return nil
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.OrganizationMember)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationMember)
	}

	// External name format: organization/username (e.g., "myorg/myuser")
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		externalName = fmt.Sprintf("%s/%s", cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Username)
		meta.SetExternalName(cr, externalName)
	}

	member, err := c.client.GetOrganizationMember(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Username)
	if err != nil {
		// If member doesn't exist, it needs to be added
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.OrganizationMemberObservation{
		Username:     &member.Username,
		Role:         &member.Role,
		Visibility:   &member.Visibility,
		Organization: &cr.Spec.ForProvider.Organization,
		UserInfo: &v1alpha1.OrganizationMemberUserInfo{
			Email:     &member.Email,
			FullName:  &member.FullName,
			AvatarURL: &member.AvatarURL,
		},
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, member),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.OrganizationMember)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationMember)
	}

	cr.SetConditions(xpv1.Creating())

	req := c.buildAddRequest(cr)

	_, err := c.client.AddOrganizationMember(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errAddOrganizationMember)
	}

	// Set external name to organization/username format
	externalName := fmt.Sprintf("%s/%s", cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Username)
	meta.SetExternalName(cr, externalName)

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.OrganizationMember)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationMember)
	}

	req := c.buildUpdateRequest(cr)

	_, err := c.client.UpdateOrganizationMember(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganizationMember)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.OrganizationMember)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganizationMember)
	}

	err := c.client.RemoveOrganizationMember(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Username)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errRemoveOrganizationMember)
	}

	return managed.ExternalDelete{}, nil
}

// buildAddRequest creates an add request from the CR spec
func (c *external) buildAddRequest(cr *v1alpha1.OrganizationMember) *giteaclients.AddOrganizationMemberRequest {
	return &giteaclients.AddOrganizationMemberRequest{
		Role: cr.Spec.ForProvider.Role,
	}
}

// buildUpdateRequest creates an update request from the CR spec
func (c *external) buildUpdateRequest(cr *v1alpha1.OrganizationMember) *giteaclients.UpdateOrganizationMemberRequest {
	req := &giteaclients.UpdateOrganizationMemberRequest{
		Role: &cr.Spec.ForProvider.Role,
	}

	if cr.Spec.ForProvider.Visibility != nil {
		req.Visibility = cr.Spec.ForProvider.Visibility
	}

	return req
}

// isUpToDate checks if the organization member is up to date with the desired state
func (c *external) isUpToDate(cr *v1alpha1.OrganizationMember, member *giteaclients.OrganizationMember) bool {
	if cr.Spec.ForProvider.Role != member.Role {
		return false
	}
	if cr.Spec.ForProvider.Visibility != nil && *cr.Spec.ForProvider.Visibility != member.Visibility {
		return false
	}

	return true
}
