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

package repositorykey

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane-contrib/provider-gitea/apis/repositorykey/v1alpha1"
	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
	giteaclients "github.com/crossplane-contrib/provider-gitea/internal/clients"
)

const (
	errNotRepositoryKey = "managed resource is not a RepositoryKey custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errNewClient        = "cannot create new Service"
	errGetRepositoryKey = "cannot get repository key"
	errCreateRepositoryKey = "cannot create repository key"
	errUpdateRepositoryKey = "cannot update repository key"
	errDeleteRepositoryKey = "cannot delete repository key"
)

// Setup adds a controller that reconciles RepositoryKey managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.RepositoryKeyKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RepositoryKeyGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.RepositoryKey{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.RepositoryKey)
	if !ok {
		return nil, errors.New(errNotRepositoryKey)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	client, err := giteaclients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: client}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client giteaclients.Client
}

func (c *external) Disconnect(ctx context.Context) error {
	// No persistent connection to disconnect
	return nil
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.RepositoryKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepositoryKey)
	}

	// External name format: repository/key_id (e.g., "myorg/myrepo/123")
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		// No external name set yet, resource doesn't exist
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Parse key ID from external name (repository/key_id format)
	var keyID int64
	var err error
	if _, err = fmt.Sscanf(externalName, cr.Spec.ForProvider.Repository+"/%d", &keyID); err != nil {
		// If we can't parse the ID, assume the resource doesn't exist yet
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	key, err := c.client.GetRepositoryKey(ctx, cr.Spec.ForProvider.Repository, keyID)
	if err != nil {
		// If key doesn't exist, it needs to be created
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.RepositoryKeyObservation{
		ID:          &key.ID,
		Title:       &key.Title,
		Fingerprint: &key.Fingerprint,
		ReadOnly:    &key.ReadOnly,
		CreatedAt:   &key.CreatedAt,
		URL:         &key.URL,
		Repository:  &cr.Spec.ForProvider.Repository,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, key),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.RepositoryKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepositoryKey)
	}

	cr.SetConditions(xpv1.Creating())

	req := c.buildCreateRequest(cr)

	key, err := c.client.CreateRepositoryKey(ctx, cr.Spec.ForProvider.Repository, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepositoryKey)
	}

	// Set external name to repository/key_id format
	externalName := fmt.Sprintf("%s/%d", cr.Spec.ForProvider.Repository, key.ID)
	meta.SetExternalName(cr, externalName)

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.RepositoryKey)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepositoryKey)
	}

	// Parse key ID from external name
	externalName := meta.GetExternalName(cr)
	var keyID int64
	if _, err := fmt.Sscanf(externalName, cr.Spec.ForProvider.Repository+"/%d", &keyID); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot parse key ID from external name")
	}

	req := c.buildUpdateRequest(cr)

	_, err := c.client.UpdateRepositoryKey(ctx, cr.Spec.ForProvider.Repository, keyID, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepositoryKey)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.RepositoryKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepositoryKey)
	}

	// Parse key ID from external name
	externalName := meta.GetExternalName(cr)
	var keyID int64
	if _, err := fmt.Sscanf(externalName, cr.Spec.ForProvider.Repository+"/%d", &keyID); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot parse key ID from external name")
	}

	err := c.client.DeleteRepositoryKey(ctx, cr.Spec.ForProvider.Repository, keyID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepositoryKey)
	}

	return managed.ExternalDelete{}, nil
}

// buildCreateRequest creates a create request from the CR spec
func (c *external) buildCreateRequest(cr *v1alpha1.RepositoryKey) *giteaclients.CreateRepositoryKeyRequest {
	req := &giteaclients.CreateRepositoryKeyRequest{
		Title: cr.Spec.ForProvider.Title,
		Key:   cr.Spec.ForProvider.Key,
	}

	if cr.Spec.ForProvider.ReadOnly != nil {
		req.ReadOnly = cr.Spec.ForProvider.ReadOnly
	}

	return req
}

// buildUpdateRequest creates an update request from the CR spec
func (c *external) buildUpdateRequest(cr *v1alpha1.RepositoryKey) *giteaclients.UpdateRepositoryKeyRequest {
	req := &giteaclients.UpdateRepositoryKeyRequest{}

	// Only title can be updated for deploy keys
	req.Title = &cr.Spec.ForProvider.Title

	if cr.Spec.ForProvider.ReadOnly != nil {
		req.ReadOnly = cr.Spec.ForProvider.ReadOnly
	}

	return req
}

// isUpToDate checks if the repository key is up to date with the desired state
func (c *external) isUpToDate(cr *v1alpha1.RepositoryKey, key *giteaclients.RepositoryKey) bool {
	if cr.Spec.ForProvider.Title != key.Title {
		return false
	}
	if cr.Spec.ForProvider.ReadOnly != nil && *cr.Spec.ForProvider.ReadOnly != key.ReadOnly {
		return false
	}
	// Note: SSH key content cannot be updated once created

	return true
}