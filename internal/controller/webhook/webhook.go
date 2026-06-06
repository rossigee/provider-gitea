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

	webhookv2 "github.com/rossigee/provider-gitea/apis/webhook/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotWebhook         = "managed resource is not a Webhook"
	errGetProviderConfig  = "cannot get ProviderConfig"
	errProviderNotReady   = "provider is not ready"
	errGetWebhook         = "cannot get webhook"
	errCreateWebhook      = "cannot create webhook"
	errUpdateWebhook      = "cannot update webhook"
	errDeleteWebhook      = "cannot delete webhook"
	errParseWebhookID     = "cannot parse webhook ID"
	errParseRepository    = "cannot parse repository"
	errMissingScope       = "webhook must have either repository or organization scope"
	controllerName        = "webhooks.webhook.gitea.m.crossplane.io"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(webhookv2.WebhookGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", "Webhook")),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(controllerName))),
		managed.WithPollInterval(o.PollInterval),
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&webhookv2.Webhook{}).
		Complete(r)
}

type connector struct{ kube client.Client }
type external struct{ client giteaclients.Client }

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*webhookv2.Webhook)
	if !ok {
		return nil, errors.New(errNotWebhook)
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
	cr, ok := mg.(*webhookv2.Webhook)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotWebhook)
	}
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	webhookID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errParseWebhookID)
	}
	var hook *giteaclients.Webhook
	if cr.Spec.ForProvider.Repository != nil {
		parts := strings.Split(*cr.Spec.ForProvider.Repository, "/")
		if len(parts) != 2 {
			return managed.ExternalObservation{}, errors.New(errParseRepository)
		}
		hook, err = e.client.GetRepositoryWebhook(ctx, parts[0], parts[1], webhookID)
	} else if cr.Spec.ForProvider.Organization != nil {
		hook, err = e.client.GetOrganizationWebhook(ctx, *cr.Spec.ForProvider.Organization, webhookID)
	} else {
		return managed.ExternalObservation{}, errors.New(errMissingScope)
	}
	if err != nil {
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetWebhook)
	}
	cr.Status.AtProvider = webhookv2.WebhookObservation{
		ID: &hook.ID,
	}
	cr.Status.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: webhookUpToDate(&cr.Spec.ForProvider, hook),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*webhookv2.Webhook)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotWebhook)
	}
	config := make(map[string]string)
	config["url"] = cr.Spec.ForProvider.URL
	if cr.Spec.ForProvider.Secret != nil {
		config["secret"] = *cr.Spec.ForProvider.Secret
	}
	if cr.Spec.ForProvider.ContentType != nil {
		config["content_type"] = *cr.Spec.ForProvider.ContentType
	}
	req := &giteaclients.CreateWebhookRequest{
		Config: config,
		Events: cr.Spec.ForProvider.Events,
	}
	if cr.Spec.ForProvider.Active != nil {
		req.Active = *cr.Spec.ForProvider.Active
	}
	var hook *giteaclients.Webhook
	var err error
	if cr.Spec.ForProvider.Repository != nil {
		parts := strings.Split(*cr.Spec.ForProvider.Repository, "/")
		if len(parts) != 2 {
			return managed.ExternalCreation{}, errors.New(errParseRepository)
		}
		hook, err = e.client.CreateRepositoryWebhook(ctx, parts[0], parts[1], req)
	} else if cr.Spec.ForProvider.Organization != nil {
		hook, err = e.client.CreateOrganizationWebhook(ctx, *cr.Spec.ForProvider.Organization, req)
	} else {
		return managed.ExternalCreation{}, errors.New(errMissingScope)
	}
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateWebhook)
	}
	meta.SetExternalName(cr, strconv.FormatInt(hook.ID, 10))
	cr.Status.SetConditions(xpv1.Creating())
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*webhookv2.Webhook)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotWebhook)
	}
	externalName := meta.GetExternalName(cr)
	webhookID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errParseWebhookID)
	}
	config := make(map[string]string)
	config["url"] = cr.Spec.ForProvider.URL
	if cr.Spec.ForProvider.Secret != nil {
		config["secret"] = *cr.Spec.ForProvider.Secret
	}
	if cr.Spec.ForProvider.ContentType != nil {
		config["content_type"] = *cr.Spec.ForProvider.ContentType
	}
	req := &giteaclients.UpdateWebhookRequest{
		Config: &config,
		Events: &cr.Spec.ForProvider.Events,
	}
	if cr.Spec.ForProvider.Active != nil {
		req.Active = cr.Spec.ForProvider.Active
	}
	if cr.Spec.ForProvider.Repository != nil {
		parts := strings.Split(*cr.Spec.ForProvider.Repository, "/")
		if len(parts) != 2 {
			return managed.ExternalUpdate{}, errors.New(errParseRepository)
		}
		if _, err := e.client.UpdateRepositoryWebhook(ctx, parts[0], parts[1], webhookID, req); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateWebhook)
		}
	} else if cr.Spec.ForProvider.Organization != nil {
		if _, err := e.client.UpdateOrganizationWebhook(ctx, *cr.Spec.ForProvider.Organization, webhookID, req); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateWebhook)
		}
	} else {
		return managed.ExternalUpdate{}, errors.New(errMissingScope)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*webhookv2.Webhook)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotWebhook)
	}
	externalName := meta.GetExternalName(cr)
	webhookID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errParseWebhookID)
	}
	var deleteErr error
	if cr.Spec.ForProvider.Repository != nil {
		parts := strings.Split(*cr.Spec.ForProvider.Repository, "/")
		if len(parts) != 2 {
			return managed.ExternalDelete{}, errors.New(errParseRepository)
		}
		deleteErr = e.client.DeleteRepositoryWebhook(ctx, parts[0], parts[1], webhookID)
	} else if cr.Spec.ForProvider.Organization != nil {
		deleteErr = e.client.DeleteOrganizationWebhook(ctx, *cr.Spec.ForProvider.Organization, webhookID)
	} else {
		return managed.ExternalDelete{}, errors.New(errMissingScope)
	}
	if deleteErr != nil && !giteaclients.IsNotFound(deleteErr) {
		return managed.ExternalDelete{}, errors.Wrap(deleteErr, errDeleteWebhook)
	}
	cr.Status.SetConditions(xpv1.Deleting())
	return managed.ExternalDelete{}, nil
}

func webhookUpToDate(desired *webhookv2.WebhookParameters, actual *giteaclients.Webhook) bool {
	if desired.URL != actual.URL {
		return false
	}
	if desired.Active != nil && *desired.Active != actual.Active {
		return false
	}
	return true
}
