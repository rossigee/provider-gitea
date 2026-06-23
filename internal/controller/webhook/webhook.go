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

// Package webhook implements the Crossplane managed-resource reconciler for the
// Gitea Webhook resource. It mirrors the canonical repository controller
// (Available() in Observe, typed not-found classification, real drift
// detection, external-name-as-identity, a non-nil rate limiter) and adds the
// org-vs-repo variant split: a webhook lives under an organization when
// spec.forProvider.organization is set, otherwise under an owner/repository.
package webhook

import (
	"context"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
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

	"github.com/rossigee/provider-gitea/apis/v1beta1"
	v2 "github.com/rossigee/provider-gitea/apis/webhook/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotWebhook        = "managed resource is not a Webhook custom resource"
	errGetWebhook        = "failed to get webhook"
	errCreateWebhook     = "failed to create webhook"
	errUpdateWebhook     = "failed to update webhook"
	errDeleteWebhook     = "failed to delete webhook"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected numeric webhook id"
)

// Setup adds a controller that reconciles Webhook managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.WebhookKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}
	// Honour spec.managementPolicies (ObserveOnly, no-delete, pause, ...) when the
	// operator runs the provider with --enable-management-policies.
	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.WebhookGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.Webhook{}).
		// A non-nil rate limiter is mandatory: ratelimiter.Reconciler.When()
		// dereferences it every reconcile, so a nil limiter panics on the first
		// event (lesson #1). o.ForControllerRuntime() also carries
		// MaxConcurrentReconciles through; without WithOptions both are dropped.
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return nil, errors.New(errNotWebhook)
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

// external observes/creates/updates/deletes the backend webhook.
type external struct {
	client clients.Client
}

// isOrg reports whether this webhook is an organization webhook (vs. a
// repository webhook). The parent is set on spec.forProvider and is immutable;
// it selects the org-or-repo client variant used consistently across
// Observe/Create/Update/Delete.
func isOrg(cr *v2.Webhook) bool {
	return ptr.Deref(cr.Spec.ForProvider.Organization, "") != ""
}

// parseID parses the numeric webhook id carried by the
// crossplane.io/external-name annotation. The annotation is authoritative for
// Observe, Update AND Delete — status.atProvider is not guaranteed to persist
// between reconciles, so keying identity off it strands deletes (lesson #14).
func parseID(cr *v2.Webhook) (int64, bool) {
	name := meta.GetExternalName(cr)
	if name == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(name, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// getWebhook fetches the backend webhook via the org-or-repo variant.
func (e *external) getWebhook(ctx context.Context, cr *v2.Webhook, id int64) (*clients.Webhook, error) {
	if isOrg(cr) {
		return e.client.GetOrganizationWebhook(ctx, ptr.Deref(cr.Spec.ForProvider.Organization, ""), id)
	}
	return e.client.GetRepositoryWebhook(ctx,
		ptr.Deref(cr.Spec.ForProvider.Owner, ""),
		ptr.Deref(cr.Spec.ForProvider.Repository, ""),
		id)
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotWebhook)
	}

	id, ok := parseID(cr)
	if !ok {
		// No usable external-name yet -> not created. Don't try to GET it.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	wh, err := e.getWebhook(ctx, cr, id)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure (auth/network/5xx) must surface so we
		// don't spuriously recreate an existing webhook.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetWebhook)
	}

	cr.Status.AtProvider = v2.WebhookObservation{
		ID: &wh.ID,
	}
	if wh.CreatedAt != "" {
		cr.Status.AtProvider.CreatedAt = &wh.CreatedAt
	}
	if wh.UpdatedAt != "" {
		cr.Status.AtProvider.UpdatedAt = &wh.UpdatedAt
	}

	upToDate := webhookUpToDate(cr, wh)

	// crossplane-runtime v2's managed reconciler no longer auto-sets
	// Available(): it only marks ReconcileSuccess. Readiness is the provider's
	// job (lesson #2/#6). Set Available on the exists path; drift is signalled
	// via ResourceUpToDate (surfaced as Synced), never by withholding Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// webhookUpToDate reports whether the observed backend webhook matches the
// mutable managed fields of the desired spec. Only fields this provider pushes
// on Update are compared; a nil/empty desired value means "do not manage", so
// it never causes drift.
func webhookUpToDate(cr *v2.Webhook, observed *clients.Webhook) bool {
	p := cr.Spec.ForProvider
	if p.Active != nil && *p.Active != observed.Active {
		return false
	}
	if len(p.Events) > 0 && !reflect.DeepEqual(p.Events, observed.Events) {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotWebhook)
	}
	cr.SetConditions(xpv1.Creating())

	createReq := &clients.CreateWebhookRequest{
		Type:   ptr.Deref(cr.Spec.ForProvider.Type, ""),
		Events: cr.Spec.ForProvider.Events,
		Active: ptr.Deref(cr.Spec.ForProvider.Active, false),
		Config: webhookConfig(cr),
	}

	var wh *clients.Webhook
	var err error
	if isOrg(cr) {
		wh, err = e.client.CreateOrganizationWebhook(ctx, ptr.Deref(cr.Spec.ForProvider.Organization, ""), createReq)
	} else {
		wh, err = e.client.CreateRepositoryWebhook(ctx,
			ptr.Deref(cr.Spec.ForProvider.Owner, ""),
			ptr.Deref(cr.Spec.ForProvider.Repository, ""),
			createReq)
	}
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateWebhook)
	}

	// Pin the backend-assigned id as the external name so every later
	// Observe/Update/Delete resolves identity from the annotation (lessons
	// #3/#7/#14).
	meta.SetExternalName(cr, strconv.FormatInt(wh.ID, 10))

	return managed.ExternalCreation{}, nil
}

// webhookConfig assembles the Gitea webhook config map from the spec fields the
// API expects inside config (url/content_type/secret).
func webhookConfig(cr *v2.Webhook) map[string]string {
	cfg := map[string]string{"url": cr.Spec.ForProvider.URL}
	if cr.Spec.ForProvider.ContentType != nil {
		cfg["content_type"] = *cr.Spec.ForProvider.ContentType
	}
	if cr.Spec.ForProvider.Secret != nil {
		cfg["secret"] = *cr.Spec.ForProvider.Secret
	}
	return cfg
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotWebhook)
	}

	id, ok := parseID(cr)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	cfg := webhookConfig(cr)
	updateReq := &clients.UpdateWebhookRequest{
		Config: &cfg,
		Events: &cr.Spec.ForProvider.Events,
		Active: cr.Spec.ForProvider.Active,
	}

	var err error
	if isOrg(cr) {
		_, err = e.client.UpdateOrganizationWebhook(ctx, ptr.Deref(cr.Spec.ForProvider.Organization, ""), id, updateReq)
	} else {
		_, err = e.client.UpdateRepositoryWebhook(ctx,
			ptr.Deref(cr.Spec.ForProvider.Owner, ""),
			ptr.Deref(cr.Spec.ForProvider.Repository, ""),
			id, updateReq)
	}
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateWebhook)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotWebhook)
	}
	cr.SetConditions(xpv1.Deleting())

	id, ok := parseID(cr)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	var err error
	if isOrg(cr) {
		err = e.client.DeleteOrganizationWebhook(ctx, ptr.Deref(cr.Spec.ForProvider.Organization, ""), id)
	} else {
		err = e.client.DeleteRepositoryWebhook(ctx,
			ptr.Deref(cr.Spec.ForProvider.Owner, ""),
			ptr.Deref(cr.Spec.ForProvider.Repository, ""),
			id)
	}
	// Treat an already-absent webhook as a successful delete so the finalizer
	// can release (idempotent delete, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteWebhook)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
