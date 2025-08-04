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

package webhook

import (
	"context"
	"strconv"

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

	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/apis/webhook/v1alpha1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotWebhook    = "managed resource is not a Webhook custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errNewClient     = "cannot create new Service"
	errCreateWebhook = "cannot create webhook"
	errUpdateWebhook = "cannot update webhook"
	errDeleteWebhook = "cannot delete webhook"
	errGetWebhook    = "cannot get webhook"
)

// Setup adds a controller that reconciles Webhook managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.WebhookKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.WebhookGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.Webhook{}).
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
	cr, ok := mg.(*v1alpha1.Webhook)
	if !ok {
		return nil, errors.New(errNotWebhook)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	giteaClient, err := giteaclients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: giteaClient}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client giteaclients.Client
}

func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// Helper methods to determine webhook type
func (c *external) isRepositoryWebhook(cr *v1alpha1.Webhook) bool {
	return cr.Spec.ForProvider.Owner != nil && cr.Spec.ForProvider.Repository != nil
}

func (c *external) isOrganizationWebhook(cr *v1alpha1.Webhook) bool {
	return cr.Spec.ForProvider.Organization != nil
}

// Helper method to convert webhook parameters to client request
func (c *external) buildWebhookRequest(cr *v1alpha1.Webhook) *giteaclients.CreateWebhookRequest {
	config := make(map[string]string)
	config["url"] = cr.Spec.ForProvider.URL

	if cr.Spec.ForProvider.ContentType != nil {
		config["content_type"] = *cr.Spec.ForProvider.ContentType
	} else {
		config["content_type"] = "json"
	}

	if cr.Spec.ForProvider.Secret != nil {
		config["secret"] = *cr.Spec.ForProvider.Secret
	}

	webhookType := "gitea"
	if cr.Spec.ForProvider.Type != nil {
		webhookType = *cr.Spec.ForProvider.Type
	}

	active := true
	if cr.Spec.ForProvider.Active != nil {
		active = *cr.Spec.ForProvider.Active
	}

	events := cr.Spec.ForProvider.Events
	if len(events) == 0 {
		events = []string{"push"}
	}

	return &giteaclients.CreateWebhookRequest{
		Type:   webhookType,
		Config: config,
		Events: events,
		Active: active,
	}
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Webhook)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotWebhook)
	}

	// Construct webhook identifier (repo/org + webhook ID)
	var webhookID int64
	if externalName := meta.GetExternalName(cr); externalName != "" {
		// Parse webhook ID from external name if it exists
		// Implementation depends on how we store the ID
		// TODO: implement external name parsing
		_ = externalName
	}

	if webhookID == 0 {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	var webhook *giteaclients.Webhook
	var err error

	// Determine webhook type and call appropriate method
	if c.isRepositoryWebhook(cr) {
		webhook, err = c.client.GetRepositoryWebhook(ctx, *cr.Spec.ForProvider.Owner, *cr.Spec.ForProvider.Repository, webhookID)
	} else if c.isOrganizationWebhook(cr) {
		// Organization webhooks don't have a Get method in current client
		// For now, assume it exists if we have an external name
		return managed.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: false, // Always update since we can't verify current state
		}, nil
	} else {
		return managed.ExternalObservation{}, errors.New("webhook must specify either repository (owner+repository) or organization")
	}

	if err != nil {
		// If webhook doesn't exist, that's not an error
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetWebhook)
	}

	// Update status with observed values
	cr.Status.AtProvider.ID = &webhook.ID
	if webhook.CreatedAt != "" {
		cr.Status.AtProvider.CreatedAt = &webhook.CreatedAt
	}
	if webhook.UpdatedAt != "" {
		cr.Status.AtProvider.UpdatedAt = &webhook.UpdatedAt
	}

	cr.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  c.isUpToDate(cr, webhook),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Webhook)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotWebhook)
	}

	req := c.buildWebhookRequest(cr)
	var webhook *giteaclients.Webhook
	var err error

	// Determine webhook type and call appropriate method
	if c.isRepositoryWebhook(cr) {
		webhook, err = c.client.CreateRepositoryWebhook(ctx, *cr.Spec.ForProvider.Owner, *cr.Spec.ForProvider.Repository, req)
	} else if c.isOrganizationWebhook(cr) {
		webhook, err = c.client.CreateOrganizationWebhook(ctx, *cr.Spec.ForProvider.Organization, req)
	} else {
		return managed.ExternalCreation{}, errors.New("webhook must specify either repository (owner+repository) or organization")
	}

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateWebhook)
	}

	// Set external name annotation to webhook ID
	meta.SetExternalName(cr, strconv.FormatInt(webhook.ID, 10))

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Webhook)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotWebhook)
	}

	// Get webhook ID from external name annotation
	webhookIDStr := meta.GetExternalName(cr)
	webhookID, err := strconv.ParseInt(webhookIDStr, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to parse webhook ID")
	}

	// Build update request
	config := make(map[string]string)
	config["url"] = cr.Spec.ForProvider.URL
	if cr.Spec.ForProvider.ContentType != nil {
		config["content_type"] = *cr.Spec.ForProvider.ContentType
	}
	if cr.Spec.ForProvider.Secret != nil {
		config["secret"] = *cr.Spec.ForProvider.Secret
	}

	events := cr.Spec.ForProvider.Events
	if len(events) == 0 {
		events = []string{"push"}
	}

	req := &giteaclients.UpdateWebhookRequest{
		Config: &config,
		Events: &events,
		Active: cr.Spec.ForProvider.Active,
	}

	// Determine webhook type and call appropriate method
	if c.isRepositoryWebhook(cr) {
		_, err = c.client.UpdateRepositoryWebhook(ctx, *cr.Spec.ForProvider.Owner, *cr.Spec.ForProvider.Repository, webhookID, req)
	} else if c.isOrganizationWebhook(cr) {
		_, err = c.client.UpdateOrganizationWebhook(ctx, *cr.Spec.ForProvider.Organization, webhookID, req)
	} else {
		return managed.ExternalUpdate{}, errors.New("webhook must specify either repository (owner+repository) or organization")
	}

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateWebhook)
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Webhook)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotWebhook)
	}

	// Get webhook ID from external name annotation
	webhookIDStr := meta.GetExternalName(cr)
	webhookID, err := strconv.ParseInt(webhookIDStr, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to parse webhook ID")
	}

	// Determine webhook type and call appropriate method
	if c.isRepositoryWebhook(cr) {
		err = c.client.DeleteRepositoryWebhook(ctx, *cr.Spec.ForProvider.Owner, *cr.Spec.ForProvider.Repository, webhookID)
	} else if c.isOrganizationWebhook(cr) {
		// Organization webhooks don't have a Delete method in current client - TODO: add this
		err = errors.New("organization webhook deletion not yet implemented")
	} else {
		return managed.ExternalDelete{}, errors.New("webhook must specify either repository (owner+repository) or organization")
	}

	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteWebhook)
}

// isUpToDate checks if the observed webhook matches the desired state
func (c *external) isUpToDate(cr *v1alpha1.Webhook, webhook *giteaclients.Webhook) bool {
	// Check if URL matches
	if cr.Spec.ForProvider.URL != webhook.Config["url"] {
		return false
	}

	// Check if active status matches
	if cr.Spec.ForProvider.Active == nil || *cr.Spec.ForProvider.Active != webhook.Active {
		return false
	}

	// Check if events match
	if len(cr.Spec.ForProvider.Events) != len(webhook.Events) {
		return false
	}

	// Add more field comparisons as needed
	return true
}
