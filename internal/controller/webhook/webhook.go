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
	"fmt"
	"strconv"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	"github.com/rossigee/provider-gitea/apis/webhook/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"
)

const (
	errNotWebhook        = "managed resource is not a Webhook custom resource"
	errGetWebhook        = "failed to get webhook"
	errCreateWebhook     = "failed to create webhook"
	errUpdateWebhook     = "failed to update webhook"
	errDeleteWebhook     = "failed to delete webhook"
	errGetProviderConfig = "failed to get provider config"
	errInvalidExternalID = "invalid external-id format, expected owner/repo/id or org/id"
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

	return &externalClient{client: conn}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it matches the managed resource's desired state.
type externalClient struct {
	client clients.Client
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "webhook.observe",
		tracing.SpanAttrs("webhook", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotWebhook)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	owner, repo, webhookID, isOrg, err := parseExternalID(externalID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	var webhook *clients.Webhook
	if isOrg {
		webhook, err = e.client.GetOrganizationWebhook(ctx, owner, webhookID)
	} else {
		webhook, err = e.client.GetRepositoryWebhook(ctx, owner, repo, webhookID)
	}
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetWebhook)
	}

	cr.Status.AtProvider = v2.WebhookObservation{
		ID:        &webhook.ID,
		CreatedAt: &webhook.CreatedAt,
		UpdatedAt: &webhook.UpdatedAt,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: isWebhookUpToDate(cr, webhook)}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "webhook.create",
		tracing.SpanAttrs("webhook", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotWebhook)
	}

	owner := ""
	repo := ""

	if cr.Spec.ForProvider.Organization != nil {
		owner = *cr.Spec.ForProvider.Organization
	} else if cr.Spec.ForProvider.Owner != nil {
		owner = *cr.Spec.ForProvider.Owner
	}

	if cr.Spec.ForProvider.Repository != nil {
		repo = *cr.Spec.ForProvider.Repository
	}

	if owner == "" {
		return managed.ExternalCreation{}, errors.New("either organization, owner, or repository is required")
	}

	webhookType := "gitea"
	if cr.Spec.ForProvider.Type != nil {
		webhookType = *cr.Spec.ForProvider.Type
	}

	active := true
	if cr.Spec.ForProvider.Active != nil {
		active = *cr.Spec.ForProvider.Active
	}

	events := []string{"push"}
	if len(cr.Spec.ForProvider.Events) > 0 {
		events = cr.Spec.ForProvider.Events
	}

	contentType := "json"
	if cr.Spec.ForProvider.ContentType != nil {
		contentType = *cr.Spec.ForProvider.ContentType
	}

	createReq := &clients.CreateWebhookRequest{
		Type: webhookType,
		Config: map[string]string{
			"url":          cr.Spec.ForProvider.URL,
			"content_type": contentType,
		},
		Events: events,
		Active: active,
	}

	if cr.Spec.ForProvider.Secret != nil {
		createReq.Config["secret"] = *cr.Spec.ForProvider.Secret
	}

	var webhook *clients.Webhook
	var err error

	if repo != "" {
		webhook, err = e.client.CreateRepositoryWebhook(ctx, owner, repo, createReq)
	} else {
		webhook, err = e.client.CreateOrganizationWebhook(ctx, owner, createReq)
	}

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateWebhook)
	}

	externalID := buildExternalID(owner, repo, webhook.ID, repo == "")
	meta.SetExternalName(cr, externalID)

	cr.Status.AtProvider = v2.WebhookObservation{
		ID:        &webhook.ID,
		CreatedAt: &webhook.CreatedAt,
		UpdatedAt: &webhook.UpdatedAt,
	}

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "webhook.update",
		tracing.SpanAttrs("webhook", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotWebhook)
	}

	externalID := meta.GetExternalName(cr)
	owner, repo, webhookID, isOrg, err := parseExternalID(externalID)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errInvalidExternalID)
	}

	active := cr.Spec.ForProvider.Active
	events := cr.Spec.ForProvider.Events

	updateReq := &clients.UpdateWebhookRequest{}
	if active != nil {
		updateReq.Active = active
	}
	if len(events) > 0 {
		updateReq.Events = &events
	}

	if isOrg {
		_, err = e.client.UpdateOrganizationWebhook(ctx, owner, webhookID, updateReq)
	} else {
		_, err = e.client.UpdateRepositoryWebhook(ctx, owner, repo, webhookID, updateReq)
	}

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateWebhook)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "webhook.delete",
		tracing.SpanAttrs("webhook", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotWebhook)
	}

	externalID := meta.GetExternalName(cr)
	owner, repo, webhookID, isOrg, err := parseExternalID(externalID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errInvalidExternalID)
	}

	var errDel error
	if isOrg {
		errDel = e.client.DeleteOrganizationWebhook(ctx, owner, webhookID)
	} else {
		errDel = e.client.DeleteRepositoryWebhook(ctx, owner, repo, webhookID)
	}

	return managed.ExternalDelete{}, errors.Wrap(errDel, errDeleteWebhook)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func parseExternalID(externalID string) (owner, repo string, id int64, isOrg bool, err error) {
	parts := strings.Split(externalID, "/")
	if len(parts) < 2 {
		return "", "", 0, false, errors.New(errInvalidExternalID)
	}

	if len(parts) == 3 {
		owner = parts[0]
		repo = parts[1]
		id, err = strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return "", "", 0, false, errors.New(errInvalidExternalID)
		}
		return owner, repo, id, false, nil
	}

	if len(parts) == 2 {
		owner = parts[0]
		id, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return "", "", 0, false, errors.New(errInvalidExternalID)
		}
		return owner, "", id, true, nil
	}

	return "", "", 0, false, errors.New(errInvalidExternalID)
}

func buildExternalID(owner, repo string, id int64, isOrg bool) string {
	if isOrg {
		return fmt.Sprintf("%s/%d", owner, id)
	}
	return fmt.Sprintf("%s/%s/%d", owner, repo, id)
}

func isWebhookUpToDate(cr *v2.Webhook, webhook *clients.Webhook) bool {
	if cr.Spec.ForProvider.URL != webhook.Config["url"] {
		return false
	}
	if cr.Spec.ForProvider.Active != nil && *cr.Spec.ForProvider.Active != webhook.Active {
		return false
	}
	return true
}

// Setup adds a controller that reconciles Webhook managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.WebhookKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.WebhookGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Webhook{}).
		Complete(r)
}
