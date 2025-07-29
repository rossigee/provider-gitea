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
	"strconv"

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

	"github.com/crossplane-contrib/provider-gitea/apis/deploykey/v1alpha1"
	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
	giteaclients "github.com/crossplane-contrib/provider-gitea/internal/clients"
)

const (
	errNotDeployKey    = "managed resource is not a DeployKey custom resource"
	errTrackPCUsage       = "cannot track ProviderConfig usage"
	errGetPC              = "cannot get ProviderConfig"
	errGetCreds           = "cannot get credentials"
	errNewClient          = "cannot create new Service"
	errCreateDeployKey = "cannot create deploy key"
	errUpdateDeployKey = "cannot update deploy key"
	errDeleteDeployKey = "cannot delete deploy key"
	errGetDeployKey    = "cannot get deploy key"
)

// Setup adds a controller that reconciles DeployKey managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.DeployKeyKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.DeployKeyGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.DeployKey{}).
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
	cr, ok := mg.(*v1alpha1.DeployKey)
	if !ok {
		return nil, errors.New(errNotDeployKey)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	giteaClient, err := giteaclients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: giteaClient}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client giteaclients.Client
}

func (c *external) Disconnect(_ context.Context) error {
	return nil
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.DeployKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDeployKey)
	}

	// Construct deploy key identifier (repo + key ID)
	var keyID int64
	if externalName := meta.GetExternalName(cr); externalName != "" {
		// Parse key ID from external name if it exists
		// Implementation depends on how we store the ID
	}

	if keyID == 0 {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	deployKey, err := c.client.GetDeployKey(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, keyID)
	if err != nil {
		// If deploy key doesn't exist, that's not an error
		if giteaclients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetDeployKey)
	}

	// Update status with observed values
	cr.Status.AtProvider.ID = &deployKey.ID
	cr.Status.AtProvider.Fingerprint = &deployKey.Fingerprint
	if deployKey.CreatedAt != "" {
		cr.Status.AtProvider.CreatedAt = &deployKey.CreatedAt
	}

	cr.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  c.isUpToDate(cr, deployKey),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.DeployKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDeployKey)
	}

	req := &giteaclients.CreateDeployKeyRequest{
		Title:    cr.Spec.ForProvider.Title,
		Key:      cr.Spec.ForProvider.Key,
		ReadOnly: *cr.Spec.ForProvider.ReadOnly,
	}
	
	deployKey, err := c.client.CreateDeployKey(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateDeployKey)
	}

	// Set external name annotation to key ID
	meta.SetExternalName(cr, strconv.FormatInt(deployKey.ID, 10))

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mg.(*v1alpha1.DeployKey)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotDeployKey)
	}

	// Deploy keys typically cannot be updated in place
	// The common pattern is to delete and recreate them
	// For now, return success without changes
	// TODO: Implement delete-and-recreate pattern if needed
	
	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.DeployKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotDeployKey)
	}

	// Get deploy key ID from external name annotation
	keyIDStr := meta.GetExternalName(cr)
	keyID, err := strconv.ParseInt(keyIDStr, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to parse deploy key ID")
	}

	err = c.client.DeleteDeployKey(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, keyID)
	return managed.ExternalDelete{}, errors.Wrap(err, errDeleteDeployKey)
}

// isUpToDate checks if the observed deploy key matches the desired state
func (c *external) isUpToDate(cr *v1alpha1.DeployKey, deployKey *giteaclients.DeployKey) bool {
	// Check if title matches
	if cr.Spec.ForProvider.Title != deployKey.Title {
		return false
	}

	// Check if read only status matches
	if cr.Spec.ForProvider.ReadOnly == nil || *cr.Spec.ForProvider.ReadOnly != deployKey.ReadOnly {
		return false
	}

	// Add more field comparisons as needed
	return true
}