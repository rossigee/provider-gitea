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

package pullrequest

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/pullrequest/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotPullRequest         = "managed resource is not a PullRequest custom resource"
	errGetPullRequest         = "failed to get pullrequest"
	errCreatePullRequest      = "failed to create pullrequest"
	errUpdatePullRequest      = "failed to update pullrequest"
	errDeletePullRequest      = "failed to delete pullrequest"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.PullRequest)
	if !ok {
		return nil, errors.New(errNotPullRequest)
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
	cr, ok := mg.(*v2.PullRequest)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPullRequest)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// TODO: Implement actual observation logic for PullRequest
	// This is a stub that marks resource as existing and up-to-date
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.PullRequest)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPullRequest)
	}

	// TODO: Implement creation logic for PullRequest
	externalID := cr.GetName()
	meta.SetExternalName(cr, externalID)

	return managed.ExternalCreation{}, errors.New("PullRequest controller not yet fully implemented")
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.PullRequest)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPullRequest)
	}

	// TODO: Implement update logic for PullRequest
	return managed.ExternalUpdate{}, errors.New("PullRequest controller not yet fully implemented")
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.PullRequest)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotPullRequest)
	}

	// TODO: Implement deletion logic for PullRequest
	return managed.ExternalDelete{}, errors.New("PullRequest controller not yet fully implemented")
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName(v2.PullRequestKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.PullRequestGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.PullRequest{}).
		Complete(r)
}
