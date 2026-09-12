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

package deploykey

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
	v2 "github.com/rossigee/provider-gitea/apis/deploykey/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotDeployKey      = "managed resource is not a DeployKey custom resource"
	errGetDeployKey      = "failed to get deploy key"
	errCreateDeployKey   = "failed to create deploy key"
	errDeleteDeployKey   = "failed to delete deploy key"
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
	cr, ok := mg.(*v2.DeployKey)
	if !ok {
		return nil, errors.New(errNotDeployKey)
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

// parseExternalName splits an "owner/repo/id" external name.
func parseExternalName(externalID string) (owner, repo string, id int64, err error) {
	parts := strings.Split(externalID, "/")
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("invalid external-id format %q, expected owner/repo/id", externalID)
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &id); err != nil {
		return "", "", 0, fmt.Errorf("invalid external-id %q: %w", externalID, err)
	}
	return parts[0], parts[1], id, nil
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "deploykey.observe",
		tracing.SpanAttrs("deploykey", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.DeployKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDeployKey)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	owner, repo, id, err := parseExternalName(externalID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	key, err := e.client.GetDeployKey(ctx, owner, repo, id)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetDeployKey)
	}

	// Update observed state
	cr.Status.AtProvider = v2.DeployKeyObservation{
		ID:          &key.ID,
		URL:         &key.URL,
		Fingerprint: &key.Fingerprint,
		CreatedAt:   &key.CreatedAt,
	}

	// Deploy keys are immutable in Gitea (no update endpoint); title/key drift
	// requires replacement, which Crossplane handles via deletion + creation
	// when the external name no longer matches. Report up-to-date when the
	// stored key material matches.
	uo := key.Title == cr.Spec.ForProvider.Title && key.Key == cr.Spec.ForProvider.Key

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "deploykey.create",
		tracing.SpanAttrs("deploykey", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.DeployKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDeployKey)
	}

	fp := cr.Spec.ForProvider
	readOnly := true
	if fp.ReadOnly != nil {
		readOnly = *fp.ReadOnly
	}

	key, err := e.client.CreateDeployKey(ctx, fp.Owner, fp.Repository, &clients.CreateDeployKeyRequest{
		Title:    fp.Title,
		Key:      fp.Key,
		ReadOnly: readOnly,
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateDeployKey)
	}

	// Set external name to owner/repo/id format for future lookups
	meta.SetExternalName(cr, fmt.Sprintf("%s/%s/%d", fp.Owner, fp.Repository, key.ID))

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// Gitea has no deploy-key update endpoint; drift is handled by reporting
	// not-up-to-date so the runtime recreates the key.
	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "deploykey.delete",
		tracing.SpanAttrs("deploykey", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.DeployKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotDeployKey)
	}

	owner, repo, id, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetDeployKey(ctx, owner, repo, id); err == nil {
		if err := e.client.DeleteDeployKey(ctx, owner, repo, id); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteDeployKey)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteDeployKey)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.DeployKeyKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.DeployKeyGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.DeployKey{}).
		Complete(r)
}
