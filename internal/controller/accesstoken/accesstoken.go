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

// Package accesstoken implements the Crossplane managed-resource reconciler for
// the Gitea AccessToken resource. It follows the canonical pattern documented
// in repository.go and crossplane-provider-template
// dev/docs/09-lessons-learned.md.
package accesstoken

import (
	"context"
	"strconv"

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

	v2 "github.com/rossigee/provider-gitea/apis/accesstoken/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotAccessToken    = "managed resource is not an AccessToken custom resource"
	errGetToken          = "failed to get access token"
	errCreateToken       = "failed to create access token"
	errDeleteToken       = "failed to delete access token"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected the numeric token id"
)

// Setup adds a controller that reconciles AccessToken managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.AccessTokenKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.AccessTokenGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.AccessToken{}).
		// A non-nil rate limiter is mandatory (lesson #1).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return nil, errors.New(errNotAccessToken)
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

// external observes/creates/deletes the backend access token.
type external struct {
	client clients.Client
}

// tokenID parses the numeric token id carried by the crossplane.io/external-name
// annotation. It is authoritative for Observe/Delete (lesson #14). A zero/empty
// value means "not created yet" — never GET id 0 (lesson #7).
func tokenID(cr *v2.AccessToken) (int64, bool) {
	name := meta.GetExternalName(cr)
	if name == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(name, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAccessToken)
	}

	id, ok := tokenID(cr)
	if !ok {
		// No usable external-name yet -> not created. Don't try to GET it.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	token, err := e.client.GetAccessToken(ctx, cr.Spec.ForProvider.Username, id)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetToken)
	}

	cr.Status.AtProvider = v2.AccessTokenObservation{
		ID:             &token.ID,
		Name:           &token.Name,
		Scopes:         token.Scopes,
		TokenLastEight: &token.TokenLastEight,
		Username:       &cr.Spec.ForProvider.Username,
	}

	// Set Available on the exists path (lesson #2/#6).
	cr.SetConditions(xpv1.Available())

	// An access token is immutable in practice: the secret VALUE can never be
	// read back, name/scopes are fixed at create time on Gitea, so the resource
	// is treated as always up-to-date once it exists to avoid false drift.
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAccessToken)
	}
	cr.SetConditions(xpv1.Creating())

	req := &clients.CreateAccessTokenRequest{
		Name:   cr.Spec.ForProvider.Name,
		Scopes: cr.Spec.ForProvider.Scopes,
	}
	token, err := e.client.CreateAccessToken(ctx, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateToken)
	}

	// Pin the authoritative numeric id from the create response as the
	// external-name so Observe/Delete resolve identity from the annotation
	// (lesson #3/#14).
	meta.SetExternalName(cr, strconv.FormatInt(token.ID, 10))

	// Surface the one-time secret token value as a connection detail; it is the
	// only chance to capture it (Gitea never returns it again).
	conn := managed.ConnectionDetails{}
	if token.Token != "" {
		conn["token"] = []byte(token.Token)
	}

	return managed.ExternalCreation{ConnectionDetails: conn}, nil
}

func (e *external) Update(_ context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// An access token has no meaningful in-place update: its scopes/name are
	// fixed at create time and the secret value is write-once. Observe always
	// reports up-to-date, so Update is never driven by drift; keep it a no-op.
	if _, ok := mg.(*v2.AccessToken); !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAccessToken)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAccessToken)
	}
	cr.SetConditions(xpv1.Deleting())

	id, ok := tokenID(cr)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteAccessToken(ctx, cr.Spec.ForProvider.Username, id)
	// An already-absent token is a successful delete (lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteToken)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
