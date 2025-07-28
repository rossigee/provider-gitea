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

package organizationsecret

import (
	"context"

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

	"github.com/crossplane-contrib/provider-gitea/apis/organizationsecret/v1alpha1"
	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
	giteaclients "github.com/crossplane-contrib/provider-gitea/internal/clients"
	corev1 "k8s.io/api/core/v1"
)

const (
	errNotOrganizationSecret    = "managed resource is not an OrganizationSecret custom resource"
	errTrackPCUsage             = "cannot track ProviderConfig usage"
	errGetPC                    = "cannot get ProviderConfig"
	errGetCreds                 = "cannot get credentials"
	errNewClient                = "cannot create new Service"
	errCreateOrganizationSecret = "cannot create organization secret"
	errUpdateOrganizationSecret = "cannot update organization secret"
	errDeleteOrganizationSecret = "cannot delete organization secret"
	errGetOrganizationSecret    = "cannot get organization secret"
	errGetSecretData            = "cannot get secret data"
)

// Setup adds a controller that reconciles OrganizationSecret managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.OrganizationSecretKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.OrganizationSecretGroupVersionKind),
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
		For(&v1alpha1.OrganizationSecret{}).
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
	cr, ok := mg.(*v1alpha1.OrganizationSecret)
	if !ok {
		return nil, errors.New(errNotOrganizationSecret)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{
		Name:      cr.GetProviderConfigReference().Name,
		Namespace: cr.GetNamespace(),
	}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	client, err := giteaclients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: client, kube: c.kube}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client giteaclients.Client
	kube   client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.OrganizationSecret)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationSecret)
	}

	secretName := cr.Spec.ForProvider.SecretName

	// Gitea organization secrets API doesn't support GET operations (returns 405)
	// So we use a write-through approach: assume the secret needs to be created/updated
	// since we can't verify its existence or current value
	
	// Check if we have an external name - this indicates we've created it before
	resourceExists := meta.GetExternalName(cr) != ""
	
	// Use secretName as external name for this resource if not already set
	if meta.GetExternalName(cr) == "" {
		meta.SetExternalName(cr, secretName)
	}

	// Always mark as needing update since we can't verify the current state
	// This ensures secrets are always synchronized with the desired state
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   resourceExists,
		ResourceUpToDate: false, // Always update since we can't verify current state
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.OrganizationSecret)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationSecret)
	}

	cr.SetConditions(xpv1.Creating())

	org := cr.Spec.ForProvider.Organization
	secretName := cr.Spec.ForProvider.SecretName

	// Get secret data
	secretData, err := c.getSecretData(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetSecretData)
	}

	req := &giteaclients.CreateOrganizationSecretRequest{
		Data: secretData,
	}

	err = c.client.CreateOrganizationSecret(ctx, org, secretName, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateOrganizationSecret)
	}

	meta.SetExternalName(cr, secretName)

	// Publish the secret data as connection details for applications to use
	connectionDetails := managed.ConnectionDetails{
		"data": []byte(secretData),
	}

	return managed.ExternalCreation{
		ConnectionDetails: connectionDetails,
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.OrganizationSecret)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationSecret)
	}

	org := cr.Spec.ForProvider.Organization
	secretName := cr.Spec.ForProvider.SecretName

	// Get secret data
	secretData, err := c.getSecretData(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetSecretData)
	}

	req := &giteaclients.CreateOrganizationSecretRequest{
		Data: secretData,
	}

	err = c.client.UpdateOrganizationSecret(ctx, org, secretName, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateOrganizationSecret)
	}

	// Publish the secret data as connection details for applications to use
	connectionDetails := managed.ConnectionDetails{
		"data": []byte(secretData),
	}

	return managed.ExternalUpdate{
		ConnectionDetails: connectionDetails,
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.OrganizationSecret)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganizationSecret)
	}

	cr.SetConditions(xpv1.Deleting())

	org := cr.Spec.ForProvider.Organization
	secretName := cr.Spec.ForProvider.SecretName

	err := c.client.DeleteOrganizationSecret(ctx, org, secretName)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteOrganizationSecret)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// Nothing to disconnect as we're using HTTP client
	return nil
}

// getSecretData retrieves the secret data from either direct value or Kubernetes secret reference
func (c *external) getSecretData(ctx context.Context, cr *v1alpha1.OrganizationSecret) (string, error) {
	// Check if direct data is provided
	if cr.Spec.ForProvider.Data != nil {
		return *cr.Spec.ForProvider.Data, nil
	}

	// Check if dataFrom is provided
	if cr.Spec.ForProvider.DataFrom != nil {
		secretRef := cr.Spec.ForProvider.DataFrom.SecretKeyRef
		secret := &corev1.Secret{}
		key := types.NamespacedName{
			Namespace: secretRef.Namespace,
			Name:      secretRef.Name,
		}

		if err := c.kube.Get(ctx, key, secret); err != nil {
			return "", errors.Wrap(err, "failed to get secret")
		}

		data, ok := secret.Data[secretRef.Key]
		if !ok {
			return "", errors.Errorf("key %s not found in secret", secretRef.Key)
		}

		return string(data), nil
	}

	return "", errors.New("either data or dataFrom must be specified")
}