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

package release

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v2 "github.com/rossigee/provider-gitea/apis/release/v2"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"

	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/rossigee/provider-gitea/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotRelease        = "managed resource is not a Release custom resource"
	errGetRelease        = "failed to get release"
	errCreateRelease     = "failed to create release"
	errUpdateRelease     = "failed to update release"
	errDeleteRelease     = "failed to delete release"
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
	cr, ok := mg.(*v2.Release)
	if !ok {
		return nil, errors.New(errNotRelease)
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
	_, span := tracing.StartSpan(ctx, "release.observe",
		tracing.SpanAttrs("release", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v2.Release)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRelease)
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	owner, repo, id, err := parseExternalName(externalID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	release, err := e.client.GetRelease(ctx, owner, repo, id)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRelease)
	}

	// Update observed state
	cr.Status.AtProvider = v2.ReleaseObservation{
		ID:              release.ID,
		TagName:         release.TagName,
		TargetCommitish: release.TargetCommitish,
		Name:            release.Name,
		Body:            release.Body,
		URL:             release.URL,
		HTMLURL:         release.HTMLURL,
		TarballURL:      release.TarballURL,
		ZipballURL:      release.ZipballURL,
		UploadURL:       release.UploadURL,
		Draft:           release.Draft,
		Prerelease:      release.Prerelease,
		CreatedAt:       release.CreatedAt,
		PublishedAt:     release.PublishedAt,
	}

	uo := isReleaseUpToDate(cr, release)

	// Only set Available() - Crossplane runtime handles Synced condition automatically
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: uo,
	}, nil
}

func isReleaseUpToDate(cr *v2.Release, release *clients.Release) bool {
	fp := cr.Spec.ForProvider
	if fp.TagName != release.TagName {
		return false
	}
	if fp.Name != nil && *fp.Name != release.Name {
		return false
	}
	if fp.Body != nil && *fp.Body != release.Body {
		return false
	}
	if fp.Draft != nil && *fp.Draft != release.Draft {
		return false
	}
	if fp.Prerelease != nil && *fp.Prerelease != release.Prerelease {
		return false
	}
	return true
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "release.create",
		tracing.SpanAttrs("release", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v2.Release)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRelease)
	}

	fp := cr.Spec.ForProvider
	req := &clients.CreateReleaseOptions{
		TagName: fp.TagName,
	}
	if fp.TargetCommitish != nil {
		req.TargetCommitish = *fp.TargetCommitish
	}
	if fp.Name != nil {
		req.Name = *fp.Name
	}
	if fp.Body != nil {
		req.Body = *fp.Body
	}
	if fp.Draft != nil {
		req.Draft = *fp.Draft
	}
	if fp.Prerelease != nil {
		req.Prerelease = *fp.Prerelease
	}
	if fp.GenerateNotes != nil {
		req.GenerateNotes = *fp.GenerateNotes
	}

	release, err := e.client.CreateRelease(ctx, fp.Owner, fp.Repository, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRelease)
	}

	// Upload inline assets (base64 content only; URL assets require download which is out of scope).
	for _, asset := range fp.Assets {
		if asset.Content == nil {
			continue
		}
		content, err := base64.StdEncoding.DecodeString(*asset.Content)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateRelease)
		}
		contentType := "application/octet-stream"
		if asset.ContentType != nil {
			contentType = *asset.ContentType
		}
		if _, err := e.client.CreateReleaseAttachment(ctx, fp.Owner, fp.Repository, release.ID, asset.Name, contentType, content); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateRelease)
		}
	}

	// Set external name to owner/repo/id format for future lookups
	meta.SetExternalName(cr, fmt.Sprintf("%s/%s/%d", fp.Owner, fp.Repository, release.ID))

	return managed.ExternalCreation{}, nil
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "release.update",
		tracing.SpanAttrs("release", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v2.Release)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRelease)
	}

	owner, repo, id, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	fp := cr.Spec.ForProvider
	tag := fp.TagName
	_, err = e.client.UpdateRelease(ctx, owner, repo, id, &clients.UpdateReleaseOptions{
		TagName:         &tag,
		TargetCommitish: fp.TargetCommitish,
		Name:            fp.Name,
		Body:            fp.Body,
		Draft:           fp.Draft,
		Prerelease:      fp.Prerelease,
		GenerateNotes:   fp.GenerateNotes,
	})
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRelease)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "release.delete",
		tracing.SpanAttrs("release", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v2.Release)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRelease)
	}

	owner, repo, id, err := parseExternalName(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, nil
	}

	// Gracefully handle already-deleted external resource.
	if _, err := e.client.GetRelease(ctx, owner, repo, id); err == nil {
		if err := e.client.DeleteRelease(ctx, owner, repo, id); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRelease)
		}
		return managed.ExternalDelete{}, nil
	} else if !strings.Contains(err.Error(), "404") {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRelease)
	}
	return managed.ExternalDelete{}, nil
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.ReleaseKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.ReleaseGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v2.Release{}).
		Complete(r)
}
