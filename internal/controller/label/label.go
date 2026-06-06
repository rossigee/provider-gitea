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

package label

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

	labelv2 "github.com/rossigee/provider-gitea/apis/label/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotLabel           = "managed resource is not a Label"
	errGetProviderConfig  = "cannot get ProviderConfig"
	errProviderNotReady   = "provider is not ready"
	errGetLabel           = "cannot get label"
	errCreateLabel        = "cannot create label"
	errUpdateLabel        = "cannot update label"
	errDeleteLabel        = "cannot delete label"
	errParseLabelID       = "cannot parse label ID"
	errParseRepository    = "cannot parse repository"
	controllerName        = "labels.label.gitea.m.crossplane.io"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(labelv2.LabelGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", "Label")),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(controllerName))),
		managed.WithPollInterval(o.PollInterval),
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&labelv2.Label{}).
		Complete(r)
}

type connector struct{ kube client.Client }
type external struct{ client giteaclients.Client }

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*labelv2.Label)
	if !ok {
		return nil, errors.New(errNotLabel)
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
	cr, ok := mg.(*labelv2.Label)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotLabel)
	}
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalObservation{}, errors.New(errParseRepository)
	}
	labelID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errParseLabelID)
	}
	label, err := e.client.GetLabel(ctx, parts[0], parts[1], labelID)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetLabel)
	}
	cr.Status.AtProvider = labelv2.LabelObservation{
		ID:  &label.ID,
		URL: &label.URL,
	}
	cr.Status.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: labelUpToDate(&cr.Spec.ForProvider, label),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*labelv2.Label)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotLabel)
	}
	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalCreation{}, errors.New(errParseRepository)
	}
	req := &giteaclients.CreateLabelRequest{
		Name:  cr.Spec.ForProvider.Name,
		Color: cr.Spec.ForProvider.Color,
	}
	if cr.Spec.ForProvider.Description != nil {
		req.Description = *cr.Spec.ForProvider.Description
	}
	label, err := e.client.CreateLabel(ctx, parts[0], parts[1], req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateLabel)
	}
	meta.SetExternalName(cr, strconv.FormatInt(label.ID, 10))
	cr.Status.SetConditions(xpv1.Creating())
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*labelv2.Label)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotLabel)
	}
	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalUpdate{}, errors.New(errParseRepository)
	}
	externalName := meta.GetExternalName(cr)
	labelID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errParseLabelID)
	}
	name := cr.Spec.ForProvider.Name
	color := cr.Spec.ForProvider.Color
	req := &giteaclients.UpdateLabelRequest{
		Name:  &name,
		Color: &color,
	}
	if cr.Spec.ForProvider.Description != nil {
		req.Description = cr.Spec.ForProvider.Description
	}
	if _, err := e.client.UpdateLabel(ctx, parts[0], parts[1], labelID, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateLabel)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*labelv2.Label)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotLabel)
	}
	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalDelete{}, errors.New(errParseRepository)
	}
	externalName := meta.GetExternalName(cr)
	labelID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errParseLabelID)
	}
	err = e.client.DeleteLabel(ctx, parts[0], parts[1], labelID)
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteLabel)
	}
	cr.Status.SetConditions(xpv1.Deleting())
	return managed.ExternalDelete{}, nil
}

func labelUpToDate(desired *labelv2.LabelParameters, actual *giteaclients.Label) bool {
	if desired.Name != actual.Name {
		return false
	}
	if desired.Color != actual.Color {
		return false
	}
	if desired.Description != nil && *desired.Description != actual.Description {
		return false
	}
	return true
}
