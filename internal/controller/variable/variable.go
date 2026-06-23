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

// Package variable implements the Crossplane managed-resource reconciler for
// the Gitea Actions Variable resource. A Variable is non-secret: its value is
// readable from a GET, so unlike OrganizationSecret/RepositorySecret this
// controller does REAL drift detection against the live value. It supports both
// scopes via the spec: if forProvider.repository is set the variable is
// repo-scoped (/repos/{owner}/{repo}/actions/variables/{name}), otherwise it is
// org-scoped (/orgs/{org}/actions/variables/{name}). Identity is the
// external-name (the variable name), pinned from spec on Create; the parent
// (org or owner/repo) is read from the immutable spec every reconcile.
package variable

import (
	"context"
	"strings"

	"github.com/pkg/errors"
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

	"github.com/rossigee/provider-gitea/apis/v1beta1"
	v2 "github.com/rossigee/provider-gitea/apis/variable/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotVariable       = "managed resource is not a Variable custom resource"
	errGetVariable       = "failed to get variable"
	errCreateVariable    = "failed to create variable"
	errUpdateVariable    = "failed to update variable"
	errDeleteVariable    = "failed to delete variable"
	errGetProviderConfig = "failed to get provider config"
	errExternalName      = "invalid external-name, expected variable name"
	errScope             = "exactly one of forProvider.organization or forProvider.repository must be set"
	errRepository        = "invalid repository, expected owner/name"
)

// Setup adds a controller that reconciles Variable managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.VariableKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}
	// Honour spec.managementPolicies (ObserveOnly, no-delete, pause, ...) when the
	// operator runs the provider with --enable-management-policies.
	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.VariableGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.Variable{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube client.Client
}

// Connect builds a Gitea API client from the resource's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.Variable)
	if !ok {
		return nil, errors.New(errNotVariable)
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

// external observes/creates/updates/deletes the backend variable.
type external struct {
	client clients.Client
}

// isRepoScoped reports whether the variable is repo-scoped (repository set) or
// org-scoped, and validates that exactly one scope field is set.
func isRepoScoped(cr *v2.Variable) (repoScoped bool, err error) {
	hasOrg := cr.Spec.ForProvider.Organization != nil && *cr.Spec.ForProvider.Organization != ""
	hasRepo := cr.Spec.ForProvider.Repository != nil && *cr.Spec.ForProvider.Repository != ""
	if hasOrg == hasRepo {
		// both set or neither set
		return false, errors.New(errScope)
	}
	return hasRepo, nil
}

// splitRepository parses the owner/name repo-scope parent.
func splitRepository(cr *v2.Variable) (owner, repo string, ok bool) {
	if cr.Spec.ForProvider.Repository == nil {
		return "", "", false
	}
	parts := strings.Split(*cr.Spec.ForProvider.Repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// getVariable fetches the live variable from the correct scoped endpoint.
func (e *external) getVariable(ctx context.Context, cr *v2.Variable, name string) (*clients.Variable, error) {
	repoScoped, err := isRepoScoped(cr)
	if err != nil {
		return nil, err
	}
	if repoScoped {
		owner, repo, ok := splitRepository(cr)
		if !ok {
			return nil, errors.New(errRepository)
		}
		return e.client.GetRepositoryVariable(ctx, owner, repo, name)
	}
	return e.client.GetOrganizationVariable(ctx, *cr.Spec.ForProvider.Organization, name)
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Variable)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotVariable)
	}

	// Identity is the external-name (the variable name); the parent scope is read
	// from spec every reconcile (lesson #14). Empty external-name -> not created
	// yet; don't issue a GET.
	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	v, err := e.getVariable(ctx, cr, name)
	if err != nil {
		// Classify not-found off the typed HTTP status, never a string match
		// (lesson #3). A real failure (auth/network/5xx) must surface.
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetVariable)
	}

	obs := v2.VariableObservation{}
	if v.ID != 0 {
		id := v.ID
		obs.ID = &id
	}
	cr.Status.AtProvider = obs

	// crossplane-runtime v2 no longer auto-sets Available() (lesson #2/#6). Set it
	// on the exists path; drift is signalled via ResourceUpToDate.
	cr.SetConditions(xpv1.Available())

	// Variables are readable, so we compare the live value to spec for REAL drift
	// detection (unlike secrets, which are write-only).
	upToDate := v.Data == cr.Spec.ForProvider.Value

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Variable)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotVariable)
	}
	cr.SetConditions(xpv1.Creating())

	repoScoped, err := isRepoScoped(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	req := &clients.VariableRequest{Value: cr.Spec.ForProvider.Value}
	if repoScoped {
		owner, repo, ok := splitRepository(cr)
		if !ok {
			return managed.ExternalCreation{}, errors.New(errRepository)
		}
		if err := e.client.CreateRepositoryVariable(ctx, owner, repo, cr.Spec.ForProvider.Name, req); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateVariable)
		}
	} else {
		if err := e.client.CreateOrganizationVariable(ctx, *cr.Spec.ForProvider.Organization, cr.Spec.ForProvider.Name, req); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateVariable)
		}
	}

	// The name IS the identity; pin it from spec after a successful create so
	// Observe/Update/Delete resolve from the annotation (lesson #3/#7/#14).
	meta.SetExternalName(cr, cr.Spec.ForProvider.Name)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Variable)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotVariable)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalUpdate{}, errors.New(errExternalName)
	}

	repoScoped, err := isRepoScoped(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	req := &clients.VariableRequest{Value: cr.Spec.ForProvider.Value}
	if repoScoped {
		owner, repo, ok := splitRepository(cr)
		if !ok {
			return managed.ExternalUpdate{}, errors.New(errRepository)
		}
		if err := e.client.UpdateRepositoryVariable(ctx, owner, repo, name, req); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateVariable)
		}
	} else {
		if err := e.client.UpdateOrganizationVariable(ctx, *cr.Spec.ForProvider.Organization, name, req); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateVariable)
		}
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Variable)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotVariable)
	}
	cr.SetConditions(xpv1.Deleting())

	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalDelete{}, errors.New(errExternalName)
	}

	repoScoped, err := isRepoScoped(cr)
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	if repoScoped {
		owner, repo, ok := splitRepository(cr)
		if !ok {
			return managed.ExternalDelete{}, errors.New(errRepository)
		}
		err = e.client.DeleteRepositoryVariable(ctx, owner, repo, name)
	} else {
		err = e.client.DeleteOrganizationVariable(ctx, *cr.Spec.ForProvider.Organization, name)
	}
	// An already-absent variable is a successful delete (idempotent, lesson #16).
	if err != nil && clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteVariable)
}

func (e *external) Disconnect(_ context.Context) error {
	// No persistent connection to tear down for the HTTP client.
	return nil
}
