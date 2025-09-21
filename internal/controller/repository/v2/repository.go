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

package v2

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v2 "github.com/rossigee/provider-gitea/apis/repository/v2"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRepositoryV2    = "managed resource is not a Repository v2 custom resource"
	errTrackPCUsage       = "cannot track ProviderConfig usage"
	errGetPC              = "cannot get ProviderConfig"
	errGetCreds           = "cannot get credentials"
	errNewClient          = "cannot create new Service"
	errCreateRepositoryV2 = "cannot create repository v2"
	errUpdateRepositoryV2 = "cannot update repository v2"
	errDeleteRepositoryV2 = "cannot delete repository v2"
	errGetRepositoryV2    = "cannot get repository v2"
)

// Setup adds a controller that reconciles Repository v2 managed resources (namespaced).
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v2.RepositoryKind + "V2")

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v2.RepositoryGroupVersionKind),
		managed.WithExternalConnecter(&connectorV2{
			kube:  mgr.GetClient(),
			usage: resource.TrackerFn(func(ctx context.Context, mg resource.Managed) error { return nil }),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v2.Repository{}).
		Complete(r)
}

// A connectorV2 is expected to produce an ExternalClient when its Connect method
// is called. V2 supports namespaced resources.
type connectorV2 struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient with v2 namespaced support:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig (namespaced or cluster-scoped).
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorV2) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v2.Repository)
	if !ok {
		return nil, errors.New(errNotRepositoryV2)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// V2 Enhancement: Support namespace-scoped ProviderConfig
	pc := &v1beta1.ProviderConfig{}
	providerConfigRef := cr.GetProviderConfigReference()

	// Try namespaced ProviderConfig first, then fall back to cluster-scoped
	pcNamespace := cr.GetNamespace()
	if pcNamespace == "" {
		pcNamespace = "default" // fallback for cluster-scoped resources
	}

	err := c.kube.Get(ctx, types.NamespacedName{
		Name:      providerConfigRef.Name,
		Namespace: pcNamespace,
	}, pc)

	if err != nil {
		// Fallback to cluster-scoped ProviderConfig for backward compatibility
		if err := c.kube.Get(ctx, types.NamespacedName{Name: providerConfigRef.Name}, pc); err != nil {
			return nil, errors.Wrap(err, errGetPC)
		}
	}

	client, err := giteaclients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &externalV2{service: client}, nil
}

// An externalV2 observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type externalV2 struct {
	service giteaclients.Client
}

func (c *externalV2) Disconnect(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

func (c *externalV2) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepositoryV2)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalObservation{}, nil
	}

	owner := ""
	if cr.Spec.ForProvider.Owner != nil {
		owner = *cr.Spec.ForProvider.Owner
	}

	repo, err := c.service.GetRepository(ctx, owner, name)
	if err != nil {
		// Repository doesn't exist
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepositoryV2)
	}

	// V2 Enhancement: Rich observability
	cr.Status.AtProvider.ID = &repo.ID
	cr.Status.AtProvider.FullName = &repo.FullName
	cr.Status.AtProvider.HTMLURL = &repo.HTMLURL
	cr.Status.AtProvider.SSHURL = &repo.SSHURL
	cr.Status.AtProvider.CloneURL = &repo.CloneURL

	// V2 enhancements - convert int to *int64 and string to *string
	size := int64(repo.Size)
	cr.Status.AtProvider.Size = &size
	if repo.Language != "" {
		cr.Status.AtProvider.Language = &repo.Language
	}

	// Check if repository is up to date
	upToDate := true
	if cr.Spec.ForProvider.Description != nil && repo.Description != *cr.Spec.ForProvider.Description {
		upToDate = false
	}
	if cr.Spec.ForProvider.Private != nil && repo.Private != *cr.Spec.ForProvider.Private {
		upToDate = false
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (c *externalV2) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepositoryV2)
	}

	cr.SetConditions(xpv1.Creating())

	owner := ""
	if cr.Spec.ForProvider.Owner != nil {
		owner = *cr.Spec.ForProvider.Owner
	}

	createOpts := &giteaclients.CreateRepositoryRequest{
		Name: cr.Spec.ForProvider.Name,
	}

	if cr.Spec.ForProvider.Description != nil {
		createOpts.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Private != nil {
		createOpts.Private = *cr.Spec.ForProvider.Private
	}
	if cr.Spec.ForProvider.AutoInit != nil {
		createOpts.AutoInit = *cr.Spec.ForProvider.AutoInit
	}

	var repo *giteaclients.Repository
	var err error

	if owner != "" {
		repo, err = c.service.CreateOrganizationRepository(ctx, owner, createOpts)
	} else {
		repo, err = c.service.CreateRepository(ctx, createOpts)
	}
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepositoryV2)
	}

	meta.SetExternalName(cr, repo.Name)

	return managed.ExternalCreation{}, nil
}

func (c *externalV2) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepositoryV2)
	}

	name := meta.GetExternalName(cr)
	owner := ""
	if cr.Spec.ForProvider.Owner != nil {
		owner = *cr.Spec.ForProvider.Owner
	}

	updateOpts := &giteaclients.UpdateRepositoryRequest{}

	updateOpts.Description = cr.Spec.ForProvider.Description
	updateOpts.Private = cr.Spec.ForProvider.Private

	_, err := c.service.UpdateRepository(ctx, owner, name, updateOpts)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepositoryV2)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *externalV2) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v2.Repository)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepositoryV2)
	}

	cr.SetConditions(xpv1.Deleting())

	name := meta.GetExternalName(cr)
	owner := ""
	if cr.Spec.ForProvider.Owner != nil {
		owner = *cr.Spec.ForProvider.Owner
	}

	err := c.service.DeleteRepository(ctx, owner, name)
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepositoryV2)
	}

	return managed.ExternalDelete{}, nil
}