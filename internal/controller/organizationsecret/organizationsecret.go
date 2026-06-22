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

// Package organizationsecret implements the Crossplane managed-resource
// reconciler for the Gitea OrganizationSecret resource. It mirrors the canonical
// reference controller (internal/controller/repository/repository.go). See
// crossplane-provider-template dev/docs/09-lessons-learned.md.
package organizationsecret

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/organizationsecret/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotOrganizationSecret    = "managed resource is not an OrganizationSecret custom resource"
	errGetOrganizationSecret    = "failed to get organizationsecret"
	errCreateOrganizationSecret = "failed to create organizationsecret"
	errUpdateOrganizationSecret = "failed to update organizationsecret"
	errDeleteOrganizationSecret = "failed to delete organizationsecret"
	errGetProviderConfig        = "failed to get provider config"
	errExternalName             = "invalid external-name, expected secret name"
)

// Setup adds a controller that reconciles OrganizationSecret managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.OrganizationSecretKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.OrganizationSecretGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.OrganizationSecret{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.OrganizationSecret)
	if !ok {
		return nil, errors.New(errNotOrganizationSecret)
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

type external struct {
	client clients.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.OrganizationSecret)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationSecret)
	}

	// Identity is the external-name (the secret name); the parent organization
	// is read from spec every reconcile (lesson #14). Empty external-name -> not
	// created yet; don't issue a GET.
	secretName := meta.GetExternalName(cr)
	if secretName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	obs, err := e.client.GetOrganizationSecret(ctx, cr.Spec.ForProvider.Organization, secretName)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure (auth/network/5xx) must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetOrganizationSecret)
	}

	cr.Status.AtProvider = v2.OrganizationSecretObservation{
		CreatedAt: &obs.CreatedAt,
		UpdatedAt: &obs.UpdatedAt,
	}

	// crossplane-runtime v2 no longer auto-sets Available() (lesson #2/#6).
	cr.SetConditions(xpv1.Available())

	// The secret VALUE is write-only — Gitea never returns it on GET — so it
	// cannot be diffed for drift. Treat the secret as always up-to-date; the
	// only mutable field (the value) is pushed unconditionally on Update.
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.OrganizationSecret)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationSecret)
	}
	cr.SetConditions(xpv1.Creating())

	req := &clients.CreateOrganizationSecretRequest{}
	if cr.Spec.ForProvider.Data != nil {
		req.Data = *cr.Spec.ForProvider.Data
	}
	if err := e.client.CreateOrganizationSecret(ctx, cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.SecretName, req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateOrganizationSecret)
	}

	// For this resource the name IS the identity; pin it from spec after a
	// successful create so Observe/Update/Delete resolve from the annotation.
	meta.SetExternalName(cr, cr.Spec.ForProvider.SecretName)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.OrganizationSecret)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationSecret)
	}

	secretName := meta.GetExternalName(cr)
	if secretName == "" {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	req := &clients.CreateOrganizationSecretRequest{}
	if cr.Spec.ForProvider.Data != nil {
		req.Data = *cr.Spec.ForProvider.Data
	}
	if err := e.client.UpdateOrganizationSecret(ctx, cr.Spec.ForProvider.Organization, secretName, req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganizationSecret)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.OrganizationSecret)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganizationSecret)
	}
	cr.SetConditions(xpv1.Deleting())

	secretName := meta.GetExternalName(cr)
	if secretName == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteOrganizationSecret(ctx, cr.Spec.ForProvider.Organization, secretName)
	// An already-absent secret is a successful delete (idempotent, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteOrganizationSecret)
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}
