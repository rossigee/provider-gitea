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

// Package repository implements the Crossplane managed-resource reconciler for
// the Gitea Repository resource. It is the canonical reference for every other
// controller in this provider — the lessons baked in here (Available() in
// Observe, typed not-found classification, real drift detection,
// external-name-as-identity, a non-nil rate limiter) are documented in
// crossplane-provider-template dev/docs/09-lessons-learned.md.
package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v2 "github.com/rossigee/provider-gitea/apis/repository/v1beta1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRepository     = "managed resource is not a Repository custom resource"
	errGetRepository     = "failed to get repository"
	errCreateRepository  = "failed to create repository"
	errUpdateRepository  = "failed to update repository"
	errDeleteRepository  = "failed to delete repository"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected owner/name"
	errTrackUsage        = "cannot track ProviderConfig usage"
)

// Setup adds a controller that reconciles Repository managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.RepositoryKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithPollJitterHook(o.PollInterval/10),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}
	// Honour spec.managementPolicies (ObserveOnly, no-delete, pause, ...) when the
	// operator runs the provider with --enable-management-policies.
	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RepositoryGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.Repository{}).
		// A non-nil rate limiter is mandatory: ratelimiter.Reconciler.When()
		// dereferences it every reconcile, so a nil limiter panics on the first
		// event (lesson #1). o.ForControllerRuntime() also carries
		// MaxConcurrentReconciles through; without WithOptions both are dropped.
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube  client.Client
	usage resource.ModernTracker
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.Repository)
	if !ok {
		return nil, errors.New(errNotRepository)
	}

	if err := c.usage.Track(ctx, cr); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
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

// external observes/creates/updates/deletes the backend repository.
type external struct {
	client clients.Client
}

// splitExternalName parses the owner/name identity carried by the
// crossplane.io/external-name annotation. The annotation is authoritative for
// Observe, Update AND Delete — status.atProvider is not guaranteed to persist
// between reconciles, so keying identity off it strands deletes (lesson #14).
func splitExternalName(cr *v2.Repository) (owner, name string, ok bool) {
	parts := strings.Split(meta.GetExternalName(cr), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepository)
	}

	owner, name, ok := splitExternalName(cr)
	if !ok {
		// No usable external-name yet -> not created. Don't try to GET it.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	repo, err := e.client.GetRepository(ctx, owner, name)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure (auth/network/5xx) must surface so we
		// don't spuriously recreate an existing repository.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepository)
	}

	cr.Status.AtProvider = v2.RepositoryObservation{
		ID:       &repo.ID,
		FullName: &repo.FullName,
		HTMLURL:  &repo.HTMLURL,
		SSHURL:   &repo.SSHURL,
		CloneURL: &repo.CloneURL,
		Language: &repo.Language,
	}

	upToDate := repositoryUpToDate(cr, repo)

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

// repositoryUpToDate reports whether the observed backend repository matches the
// managed fields of the desired spec. Only fields this provider actually pushes
// on Update are compared; a nil desired pointer means "do not manage", so it
// never causes drift.
func repositoryUpToDate(cr *v2.Repository, observed *clients.Repository) bool {
	p := cr.Spec.ForProvider
	if p.Description != nil && *p.Description != observed.Description {
		return false
	}
	if p.Private != nil && *p.Private != observed.Private {
		return false
	}
	if p.Template != nil && *p.Template != observed.Template {
		return false
	}
	if p.Archived != nil && *p.Archived != observed.Archived {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}
	cr.SetConditions(xpv1.Creating())

	owner := ptr.Deref(cr.Spec.ForProvider.Owner, "")

	createReq := &clients.CreateRepositoryRequest{Name: cr.Spec.ForProvider.Name}
	if cr.Spec.ForProvider.Description != nil {
		createReq.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Private != nil {
		createReq.Private = *cr.Spec.ForProvider.Private
	}
	if cr.Spec.ForProvider.AutoInit != nil {
		createReq.AutoInit = *cr.Spec.ForProvider.AutoInit
	}
	if cr.Spec.ForProvider.Template != nil {
		createReq.Template = *cr.Spec.ForProvider.Template
	}
	if cr.Spec.ForProvider.DefaultBranch != nil {
		createReq.DefaultBranch = *cr.Spec.ForProvider.DefaultBranch
	}
	if cr.Spec.ForProvider.TrustModel != nil {
		createReq.TrustModel = *cr.Spec.ForProvider.TrustModel
	}

	var repo *clients.Repository
	var err error
	if owner != "" {
		repo, err = e.client.CreateOrganizationRepository(ctx, owner, createReq)
	} else {
		repo, err = e.client.CreateRepository(ctx, createReq)
	}
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepository)
	}

	// Capture the authoritative owner/name from the backend response (re-read,
	// not a guess) and pin it as the external name so every later
	// Observe/Update/Delete resolves identity from the annotation (lessons
	// #3/#7/#14). Gitea always returns the owner on create.
	resolvedOwner := owner
	if repo.Owner != nil && repo.Owner.Username != "" {
		resolvedOwner = repo.Owner.Username
	}
	meta.SetExternalName(cr, fmt.Sprintf("%s/%s", resolvedOwner, repo.Name))

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepository)
	}

	owner, name, ok := splitExternalName(cr)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	updateReq := &clients.UpdateRepositoryRequest{
		Description:   cr.Spec.ForProvider.Description,
		Private:       cr.Spec.ForProvider.Private,
		Template:      cr.Spec.ForProvider.Template,
		Archived:      cr.Spec.ForProvider.Archived,
		DefaultBranch: cr.Spec.ForProvider.DefaultBranch,
	}

	if _, err := e.client.UpdateRepository(ctx, owner, name, updateReq); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepository)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepository)
	}
	cr.SetConditions(xpv1.Deleting())

	owner, name, ok := splitExternalName(cr)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	err := e.client.DeleteRepository(ctx, owner, name)
	// Treat an already-absent repository as a successful delete so the
	// finalizer can release (idempotent delete, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepository)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
