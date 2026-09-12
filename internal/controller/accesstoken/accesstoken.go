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

package accesstoken

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/accesstoken/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotAccessToken    = "managed resource is not an AccessToken custom resource"
	errGetAccessToken    = "failed to get access token"
	errCreateAccessToken = "failed to create access token"
	errUpdateAccessToken = "failed to update access token"
	errDeleteAccessToken = "failed to delete access token"
	errGetProviderConfig = "failed to get provider config"
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
	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return nil, errors.New(errNotAccessToken)
	}

	// Get provider config reference from spec
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

// parseExternalName splits a "username/id" external name.
func parseExternalName(externalID string) (string, int64, error) {
	parts := strings.Split(externalID, "/")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid external-id format %q, expected username/id", externalID)
	}
	var id int64
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
		return "", 0, fmt.Errorf("invalid external-id %q: %w", externalID, err)
	}
	return parts[0], id, nil
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "accesstoken.observe",
		tracing.SpanAttrs("accesstoken", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAccessToken)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	username, id, err := parseExternalName(externalID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	token, err := e.client.GetAccessToken(ctx, username, id)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetAccessToken)
	}

	// Update observed state
	cr.Status.AtProvider = v2.AccessTokenObservation{
		ID:             &token.ID,
		Name:           &token.Name,
		Scopes:         token.Scopes,
		TokenLastEight: &token.TokenLastEight,
		Username:       &username,
	}

	uo := isAccessTokenUpToDate(cr, token)

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func isAccessTokenUpToDate(cr *v2.AccessToken, token *clients.AccessToken) bool {
	if len(cr.Spec.ForProvider.Scopes) != len(token.Scopes) {
		return false
	}
	want := make(map[string]struct{}, len(cr.Spec.ForProvider.Scopes))
	for _, s := range cr.Spec.ForProvider.Scopes {
		want[s] = struct{}{}
	}
	for _, s := range token.Scopes {
		if _, ok := want[s]; !ok {
			return false
		}
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "accesstoken.create",
		tracing.SpanAttrs("accesstoken", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAccessToken)
	}

	token, err := e.client.CreateAccessToken(ctx, cr.Spec.ForProvider.Username, &clients.CreateAccessTokenRequest{
		Name:   cr.Spec.ForProvider.Name,
		Scopes: cr.Spec.ForProvider.Scopes,
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAccessToken)
	}

	// Set external name to username/id format for future lookups
	meta.SetExternalName(cr, fmt.Sprintf("%s/%d", cr.Spec.ForProvider.Username, token.ID))

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "accesstoken.update",
		tracing.SpanAttrs("accesstoken", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAccessToken)
	}

	username, id, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	name := cr.Spec.ForProvider.Name
	_, err = e.client.UpdateAccessToken(ctx, username, id, &clients.UpdateAccessTokenRequest{
		Name:   &name,
		Scopes: cr.Spec.ForProvider.Scopes,
	})
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAccessToken)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "accesstoken.delete",
		tracing.SpanAttrs("accesstoken", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.AccessToken)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAccessToken)
	}

	username, id, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetAccessToken(ctx, username, id); err == nil {
		if err := e.client.DeleteAccessToken(ctx, username, id); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAccessToken)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAccessToken)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.AccessTokenKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.AccessTokenGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.AccessToken{}).
		Complete(r)
}
