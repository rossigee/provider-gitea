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

package organization

import (
	"context"

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

	"github.com/crossplane-contrib/provider-gitea/apis/organization/v1alpha1"
	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
	giteaclients "github.com/crossplane-contrib/provider-gitea/internal/clients"
)

const (
	errNotOrganization    = "managed resource is not an Organization custom resource"
	errTrackPCUsage       = "cannot track ProviderConfig usage"
	errGetPC              = "cannot get ProviderConfig"
	errGetCreds           = "cannot get credentials"
	errNewClient          = "cannot create new Service"
	errCreateOrganization = "cannot create organization"
	errUpdateOrganization = "cannot update organization"
	errDeleteOrganization = "cannot delete organization"
	errGetOrganization    = "cannot get organization"
)

// Setup adds a controller that reconciles Organization managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.OrganizationKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.OrganizationGroupVersionKind),
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
		For(&v1alpha1.Organization{}).
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
	cr, ok := mg.(*v1alpha1.Organization)
	if !ok {
		return nil, errors.New(errNotOrganization)
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
	cr, ok := mg.(*v1alpha1.Organization)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganization)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	organization, err := c.client.GetOrganization(ctx, externalName)
	if err != nil {
		// If organization doesn't exist, return that it needs to be created
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.OrganizationObservation{
		ID:        &organization.ID,
		Email:     &organization.Email,
		AvatarURL: &organization.AvatarURL,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, organization),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Organization)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganization)
	}

	cr.SetConditions(xpv1.Creating())

	req := &giteaclients.CreateOrganizationRequest{
		Username: cr.Spec.ForProvider.Username,
	}

	if cr.Spec.ForProvider.Name != nil {
		req.Name = *cr.Spec.ForProvider.Name
	}
	if cr.Spec.ForProvider.FullName != nil {
		req.FullName = *cr.Spec.ForProvider.FullName
	}
	if cr.Spec.ForProvider.Description != nil {
		req.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Website != nil {
		req.Website = *cr.Spec.ForProvider.Website
	}
	if cr.Spec.ForProvider.Location != nil {
		req.Location = *cr.Spec.ForProvider.Location
	}
	if cr.Spec.ForProvider.Visibility != nil {
		req.Visibility = *cr.Spec.ForProvider.Visibility
	}
	if cr.Spec.ForProvider.RepoAdminChangeTeamAccess != nil {
		req.RepoAdminChangeTeamAccess = *cr.Spec.ForProvider.RepoAdminChangeTeamAccess
	}

	organization, err := c.client.CreateOrganization(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateOrganization)
	}

	meta.SetExternalName(cr, organization.Username)

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Organization)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganization)
	}

	externalName := meta.GetExternalName(cr)

	req := &giteaclients.UpdateOrganizationRequest{}

	if cr.Spec.ForProvider.Name != nil {
		req.Name = cr.Spec.ForProvider.Name
	}
	if cr.Spec.ForProvider.FullName != nil {
		req.FullName = cr.Spec.ForProvider.FullName
	}
	if cr.Spec.ForProvider.Description != nil {
		req.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Website != nil {
		req.Website = cr.Spec.ForProvider.Website
	}
	if cr.Spec.ForProvider.Location != nil {
		req.Location = cr.Spec.ForProvider.Location
	}
	if cr.Spec.ForProvider.Visibility != nil {
		req.Visibility = cr.Spec.ForProvider.Visibility
	}
	if cr.Spec.ForProvider.RepoAdminChangeTeamAccess != nil {
		req.RepoAdminChangeTeamAccess = cr.Spec.ForProvider.RepoAdminChangeTeamAccess
	}

	_, err := c.client.UpdateOrganization(ctx, externalName, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganization)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Organization)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganization)
	}

	cr.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(cr)

	err := c.client.DeleteOrganization(ctx, externalName)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteOrganization)
	}

	return managed.ExternalDelete{}, nil
}

// isUpToDate checks if the organization is up to date with the desired state
func (c *external) isUpToDate(cr *v1alpha1.Organization, organization *giteaclients.Organization) bool {
	if cr.Spec.ForProvider.Name != nil && *cr.Spec.ForProvider.Name != organization.Name {
		return false
	}
	if cr.Spec.ForProvider.FullName != nil && *cr.Spec.ForProvider.FullName != organization.FullName {
		return false
	}
	if cr.Spec.ForProvider.Description != nil && *cr.Spec.ForProvider.Description != organization.Description {
		return false
	}
	if cr.Spec.ForProvider.Website != nil && *cr.Spec.ForProvider.Website != organization.Website {
		return false
	}
	if cr.Spec.ForProvider.Location != nil && *cr.Spec.ForProvider.Location != organization.Location {
		return false
	}
	if cr.Spec.ForProvider.Visibility != nil && *cr.Spec.ForProvider.Visibility != organization.Visibility {
		return false
	}
	if cr.Spec.ForProvider.RepoAdminChangeTeamAccess != nil && *cr.Spec.ForProvider.RepoAdminChangeTeamAccess != organization.RepoAdminChangeTeamAccess {
		return false
	}

	return true
}
