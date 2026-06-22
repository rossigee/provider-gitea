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

// Package release implements the Crossplane managed-resource reconciler for the
// Gitea Release resource. It follows the canonical shape of the repository
// controller: Available() in Observe, typed not-found classification, real
// drift detection, external-name-as-identity, and a non-nil rate limiter.
//
// Releases are ID-keyed: the backend assigns an int64 id on create, and the
// parent (owner, repo) is taken from cr.Spec.ForProvider (immutable). The
// external-name is the decimal release id.
package release

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-gitea/apis/release/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRelease        = "managed resource is not a Release custom resource"
	errGetRelease        = "failed to get release"
	errCreateRelease     = "failed to create release"
	errUpdateRelease     = "failed to update release"
	errDeleteRelease     = "failed to delete release"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected numeric release id"
)

// Setup adds a controller that reconciles Release managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.ReleaseKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.ReleaseGroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.Release{}).
		// A non-nil rate limiter is mandatory: ratelimiter.Reconciler.When()
		// dereferences it every reconcile, so a nil limiter panics on the first
		// event (lesson #1). o.ForControllerRuntime() also carries
		// MaxConcurrentReconciles through; without WithOptions both are dropped.
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.Release)
	if !ok {
		return nil, errors.New(errNotRelease)
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

// external observes/creates/updates/deletes the backend release.
type external struct {
	client clients.Client
}

// releaseID parses the numeric release id carried by the
// crossplane.io/external-name annotation. The annotation is authoritative for
// Observe, Update AND Delete — status.atProvider is not guaranteed to persist
// between reconciles, so keying identity off it strands deletes (lesson #14).
func releaseID(cr *v2.Release) (id int64, ok bool) {
	name := meta.GetExternalName(cr)
	if name == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(name, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Release)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRelease)
	}

	id, ok := releaseID(cr)
	if !ok {
		// No usable external-name yet -> not created. Don't try to GET it.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	owner := cr.Spec.ForProvider.Owner
	repo := cr.Spec.ForProvider.Repository

	rel, err := e.client.GetRelease(ctx, owner, repo, id)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure (auth/network/5xx) must surface so we
		// don't spuriously recreate an existing release.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRelease)
	}

	cr.Status.AtProvider = v2.ReleaseObservation{
		ID:              rel.ID,
		TagName:         rel.TagName,
		TargetCommitish: rel.TargetCommitish,
		Name:            rel.Name,
		Body:            rel.Body,
		URL:             rel.URL,
		HTMLURL:         rel.HTMLURL,
		TarballURL:      rel.TarballURL,
		ZipballURL:      rel.ZipballURL,
		Draft:           rel.Draft,
		Prerelease:      rel.Prerelease,
	}

	upToDate := releaseUpToDate(cr, rel)

	// crossplane-runtime v2's managed reconciler no longer auto-sets
	// Available(): it only marks ReconcileSuccess. Readiness is the provider's
	// job (lesson #2/#6). Set Available on the exists path; drift is signalled
	// via ResourceUpToDate (surfaced as Synced), never by withholding Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// releaseUpToDate reports whether the observed backend release matches the
// managed fields of the desired spec. Only fields this provider actually pushes
// on Update are compared; a nil desired pointer means "do not manage", so it
// never causes drift.
func releaseUpToDate(cr *v2.Release, observed *clients.Release) bool {
	p := cr.Spec.ForProvider
	if p.Name != nil && *p.Name != observed.Name {
		return false
	}
	if p.Body != nil && *p.Body != observed.Body {
		return false
	}
	if p.Draft != nil && *p.Draft != observed.Draft {
		return false
	}
	if p.Prerelease != nil && *p.Prerelease != observed.Prerelease {
		return false
	}
	if p.TargetCommitish != nil && *p.TargetCommitish != observed.TargetCommitish {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Release)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRelease)
	}
	cr.SetConditions(xpv1.Creating())

	owner := cr.Spec.ForProvider.Owner
	repo := cr.Spec.ForProvider.Repository

	createReq := &clients.CreateReleaseOptions{
		TagName:         cr.Spec.ForProvider.TagName,
		TargetCommitish: ptr.Deref(cr.Spec.ForProvider.TargetCommitish, ""),
		Name:            ptr.Deref(cr.Spec.ForProvider.Name, ""),
		Body:            ptr.Deref(cr.Spec.ForProvider.Body, ""),
		Draft:           ptr.Deref(cr.Spec.ForProvider.Draft, false),
		Prerelease:      ptr.Deref(cr.Spec.ForProvider.Prerelease, false),
		GenerateNotes:   ptr.Deref(cr.Spec.ForProvider.GenerateNotes, false),
	}

	rel, err := e.client.CreateRelease(ctx, owner, repo, createReq)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRelease)
	}

	// Capture the backend-assigned id (re-read, not a guess) and pin it as the
	// external name so every later Observe/Update/Delete resolves identity from
	// the annotation (lessons #3/#7/#14).
	meta.SetExternalName(cr, strconv.FormatInt(rel.ID, 10))

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Release)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRelease)
	}

	id, ok := releaseID(cr)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	owner := cr.Spec.ForProvider.Owner
	repo := cr.Spec.ForProvider.Repository

	updateReq := &clients.UpdateReleaseOptions{
		TagName:         &cr.Spec.ForProvider.TagName,
		TargetCommitish: cr.Spec.ForProvider.TargetCommitish,
		Name:            cr.Spec.ForProvider.Name,
		Body:            cr.Spec.ForProvider.Body,
		Draft:           cr.Spec.ForProvider.Draft,
		Prerelease:      cr.Spec.ForProvider.Prerelease,
		GenerateNotes:   cr.Spec.ForProvider.GenerateNotes,
	}

	if _, err := e.client.UpdateRelease(ctx, owner, repo, id, updateReq); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRelease)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Release)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRelease)
	}
	cr.SetConditions(xpv1.Deleting())

	id, ok := releaseID(cr)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	owner := cr.Spec.ForProvider.Owner
	repo := cr.Spec.ForProvider.Repository

	err := e.client.DeleteRelease(ctx, owner, repo, id)
	// Treat an already-absent release as a successful delete so the finalizer
	// can release (idempotent delete, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRelease)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
