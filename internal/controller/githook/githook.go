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

package githook

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/githook/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotGitHook        = "managed resource is not a GitHook custom resource"
	errGetGitHook        = "failed to get git hook"
	errCreateGitHook     = "failed to create git hook"
	errUpdateGitHook     = "failed to update git hook"
	errDeleteGitHook     = "failed to delete git hook"
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
	cr, ok := mg.(*v2.GitHook)
	if !ok {
		return nil, errors.New(errNotGitHook)
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

// contentHash returns a stable hash of hook content for drift detection.
func contentHash(content string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
}

// parseExternalName splits an "owner/name/hooktype" external name.
func parseExternalName(externalID string) (repository, hookType string, err error) {
	parts := strings.Split(externalID, "/")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid external-id format %q, expected owner/name/hooktype", externalID)
	}
	return parts[0] + "/" + parts[1], parts[2], nil
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "githook.observe",
		tracing.SpanAttrs("githook", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.GitHook)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotGitHook)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	repository, hookType, err := parseExternalName(externalID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	hook, err := e.client.GetGitHook(ctx, repository, hookType)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetGitHook)
	}

	// Update observed state
	hash := contentHash(cr.Spec.ForProvider.Content)
	cr.Status.AtProvider = v2.GitHookObservation{
		Name:        &hook.Name,
		ContentHash: &hash,
	}

	uo := hook.Content == cr.Spec.ForProvider.Content
	if cr.Spec.ForProvider.IsActive != nil {
		uo = uo && hook.IsActive == *cr.Spec.ForProvider.IsActive
	}

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "githook.create",
		tracing.SpanAttrs("githook", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.GitHook)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotGitHook)
	}

	fp := cr.Spec.ForProvider
	active := true
	if fp.IsActive != nil {
		active = *fp.IsActive
	}

	if _, err := e.client.CreateGitHook(ctx, fp.Repository, &clients.CreateGitHookRequest{
		HookType: fp.HookType,
		Content:  fp.Content,
		IsActive: active,
	}); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateGitHook)
	}

	// Set external name to owner/name/hooktype format for future lookups
	meta.SetExternalName(cr, fmt.Sprintf("%s/%s", fp.Repository, fp.HookType))

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "githook.update",
		tracing.SpanAttrs("githook", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.GitHook)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotGitHook)
	}

	repository, hookType, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	fp := cr.Spec.ForProvider
	active := true
	if fp.IsActive != nil {
		active = *fp.IsActive
	}

	if _, err := e.client.UpdateGitHook(ctx, repository, hookType, &clients.UpdateGitHookRequest{
		Content:  fp.Content,
		IsActive: active,
	}); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateGitHook)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "githook.delete",
		tracing.SpanAttrs("githook", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.GitHook)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotGitHook)
	}

	repository, hookType, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetGitHook(ctx, repository, hookType); err == nil {
		if err := e.client.DeleteGitHook(ctx, repository, hookType); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteGitHook)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteGitHook)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.GitHookKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.GitHookGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.GitHook{}).
		Complete(r)
}
