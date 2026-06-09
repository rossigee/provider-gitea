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

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/label/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotLabel         = "managed resource is not a Label custom resource"
	errGetLabel         = "failed to get label"
	errCreateLabel      = "failed to create label"
	errUpdateLabel      = "failed to update label"
	errDeleteLabel      = "failed to delete label"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.Label)
	if !ok {
		return nil, errors.New(errNotLabel)
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
	cr, ok := mg.(*v2.Label)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotLabel)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	labelID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to parse label ID")
	}

	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalObservation{}, errors.New("invalid repository format")
	}

	label, err := e.client.GetLabel(ctx, parts[0], parts[1], labelID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetLabel)
	}

	cr.Status.AtProvider = v2.LabelObservation{
		ID:  &label.ID,
		URL: &label.URL,
	}

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Label)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotLabel)
	}

	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalCreation{}, errors.New("invalid repository format")
	}

	createReq := &clients.CreateLabelRequest{
		Name:  cr.Spec.ForProvider.Name,
		Color: cr.Spec.ForProvider.Color,
	}

	if cr.Spec.ForProvider.Description != nil {
		createReq.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Exclusive != nil {
		createReq.Exclusive = *cr.Spec.ForProvider.Exclusive
	}

	label, err := e.client.CreateLabel(ctx, parts[0], parts[1], createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateLabel)
	}

	meta.SetExternalName(cr, strconv.FormatInt(label.ID, 10))
	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Label)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotLabel)
	}

	externalID := meta.GetExternalName(cr)
	labelID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to parse label ID")
	}

	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalUpdate{}, errors.New("invalid repository format")
	}

	updateReq := &clients.UpdateLabelRequest{}

	if cr.Spec.ForProvider.Description != nil {
		updateReq.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Color != "" {
		color := cr.Spec.ForProvider.Color
		updateReq.Color = &color
	}
	if cr.Spec.ForProvider.Exclusive != nil {
		updateReq.Exclusive = cr.Spec.ForProvider.Exclusive
	}

	_, err = e.client.UpdateLabel(ctx, parts[0], parts[1], labelID, updateReq)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateLabel)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Label)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotLabel)
	}

	externalID := meta.GetExternalName(cr)
	labelID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to parse label ID")
	}

	parts := strings.Split(cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 {
		return managed.ExternalDelete{}, errors.New("invalid repository format")
	}

	err = e.client.DeleteLabel(ctx, parts[0], parts[1], labelID)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteLabel)
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.LabelKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.LabelGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Label{}).
		Complete(r)
}
