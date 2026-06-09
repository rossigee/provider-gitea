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
	"strings"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/webhook/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotWebhook         = "managed resource is not a Webhook custom resource"
	errGetWebhook         = "failed to get webhook"
	errCreateWebhook      = "failed to create webhook"
	errUpdateWebhook      = "failed to update webhook"
	errDeleteWebhook      = "failed to delete webhook"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

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

type externalClient struct {
	client clients.Client
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotWebhook)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	webhookID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to parse webhook ID")
	}

	owner := ""
	repo := ""
	if cr.Spec.ForProvider.Owner != nil && cr.Spec.ForProvider.Repository != nil {
		owner = *cr.Spec.ForProvider.Owner
		repo = *cr.Spec.ForProvider.Repository
	}

	webhook, err := e.client.GetRepositoryWebhook(ctx, owner, repo, webhookID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetWebhook)
	}

	cr.Status.AtProvider = v2.WebhookObservation{
		ID: &webhook.ID,
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotWebhook)
	}

	owner := ""
	repo := ""
	if cr.Spec.ForProvider.Owner != nil && cr.Spec.ForProvider.Repository != nil {
		owner = *cr.Spec.ForProvider.Owner
		repo = *cr.Spec.ForProvider.Repository
	}

	webhookType := "gitea"
	if cr.Spec.ForProvider.Type != nil {
		webhookType = *cr.Spec.ForProvider.Type
	}

	config := map[string]string{
		"url": cr.Spec.ForProvider.URL,
	}
	if cr.Spec.ForProvider.ContentType != nil {
		config["content_type"] = *cr.Spec.ForProvider.ContentType
	}
	if cr.Spec.ForProvider.Secret != nil {
		config["secret"] = *cr.Spec.ForProvider.Secret
	}
	if cr.Spec.ForProvider.BranchFilter != nil {
		config["branch_filter"] = *cr.Spec.ForProvider.BranchFilter
	}

	active := true
	if cr.Spec.ForProvider.Active != nil {
		active = *cr.Spec.ForProvider.Active
	}

	createReq := &clients.CreateWebhookRequest{
		Type:   webhookType,
		Config: config,
		Events: cr.Spec.ForProvider.Events,
		Active: active,
	}

	webhook, err := e.client.CreateRepositoryWebhook(ctx, owner, repo, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateWebhook)
	}

	meta.SetExternalName(cr, strconv.FormatInt(webhook.ID, 10))
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotWebhook)
	}

	externalID := meta.GetExternalName(cr)
	webhookID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to parse webhook ID")
	}

	owner := ""
	repo := ""
	if cr.Spec.ForProvider.Owner != nil && cr.Spec.ForProvider.Repository != nil {
		owner = *cr.Spec.ForProvider.Owner
		repo = *cr.Spec.ForProvider.Repository
	}

	updateReq := &clients.UpdateWebhookRequest{}

	if cr.Spec.ForProvider.Active != nil {
		active := *cr.Spec.ForProvider.Active
		updateReq.Active = &active
	}
	if len(cr.Spec.ForProvider.Events) > 0 {
		events := cr.Spec.ForProvider.Events
		updateReq.Events = &events
	}

	_, err = e.client.UpdateRepositoryWebhook(ctx, owner, repo, webhookID, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateWebhook)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Webhook)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotWebhook)
	}

	externalID := meta.GetExternalName(cr)
	webhookID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to parse webhook ID")
	}

	owner := ""
	repo := ""
	if cr.Spec.ForProvider.Owner != nil && cr.Spec.ForProvider.Repository != nil {
		owner = *cr.Spec.ForProvider.Owner
		repo = *cr.Spec.ForProvider.Repository
	}

	err = e.client.DeleteRepositoryWebhook(ctx, owner, repo, webhookID)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteWebhook)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.WebhookKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.WebhookGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Webhook{}).
		Complete(r)
}
