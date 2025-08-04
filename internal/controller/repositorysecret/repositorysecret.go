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

package repositorysecret

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/repositorysecret/v1alpha1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRepositorySecret    = "managed resource is not a RepositorySecret custom resource"
	errTrackPCUsage           = "cannot track ProviderConfig usage"
	errGetPC                  = "cannot get ProviderConfig"
	errGetCreds               = "cannot get credentials"
	errNewClient              = "cannot create new Service"
	errGetSecret              = "cannot get secret from Kubernetes"
	errGetRepositorySecret    = "cannot get repository secret"
	errCreateRepositorySecret = "cannot create repository secret"
	errUpdateRepositorySecret = "cannot update repository secret"
	errDeleteRepositorySecret = "cannot delete repository secret"
)

// Setup adds a controller that reconciles RepositorySecret managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.RepositorySecretKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RepositorySecretGroupVersionKind),
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
		For(&v1alpha1.RepositorySecret{}).
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
	cr, ok := mg.(*v1alpha1.RepositorySecret)
	if !ok {
		return nil, errors.New(errNotRepositorySecret)
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

	return &external{client: client, kube: c.kube}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client giteaclients.Client
	kube   client.Client
}

func (c *external) Disconnect(ctx context.Context) error {
	// No persistent connection to disconnect
	return nil
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.RepositorySecret)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepositorySecret)
	}

	// External name format: repository/secret_name (e.g., "myorg/myrepo/CI_TOKEN")
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		externalName = fmt.Sprintf("%s/%s", cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.SecretName)
		meta.SetExternalName(cr, externalName)
	}

	secret, err := c.client.GetRepositorySecret(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.SecretName)
	if err != nil {
		// If secret doesn't exist, it needs to be created
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.RepositorySecretObservation{
		SecretName: &secret.Name,
		CreatedAt:  &secret.CreatedAt,
		Repository: &cr.Spec.ForProvider.Repository,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true, // Secrets are considered up-to-date if they exist
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.RepositorySecret)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepositorySecret)
	}

	cr.SetConditions(xpv1.Creating())

	// Get secret value from Kubernetes secret
	secretValue, err := c.getSecretValue(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetSecret)
	}

	req := &giteaclients.CreateRepositorySecretRequest{
		Data: secretValue,
	}

	err = c.client.CreateRepositorySecret(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.SecretName, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepositorySecret)
	}

	// Set external name to repository/secret_name format
	externalName := fmt.Sprintf("%s/%s", cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.SecretName)
	meta.SetExternalName(cr, externalName)

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.RepositorySecret)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepositorySecret)
	}

	// Get secret value from Kubernetes secret
	secretValue, err := c.getSecretValue(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetSecret)
	}

	req := &giteaclients.UpdateRepositorySecretRequest{
		Data: secretValue,
	}

	err = c.client.UpdateRepositorySecret(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.SecretName, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepositorySecret)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.RepositorySecret)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepositorySecret)
	}

	err := c.client.DeleteRepositorySecret(ctx, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.SecretName)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepositorySecret)
	}

	return managed.ExternalDelete{}, nil
}

// getSecretValue retrieves the secret value from the referenced Kubernetes secret
func (c *external) getSecretValue(ctx context.Context, cr *v1alpha1.RepositorySecret) (string, error) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Namespace: cr.Spec.ForProvider.ValueSecretRef.Namespace,
		Name:      cr.Spec.ForProvider.ValueSecretRef.Name,
	}

	if err := c.kube.Get(ctx, key, secret); err != nil {
		return "", errors.Wrap(err, "failed to get secret")
	}

	value, ok := secret.Data[cr.Spec.ForProvider.ValueSecretRef.Key]
	if !ok {
		return "", errors.Errorf("key %s not found in secret", cr.Spec.ForProvider.ValueSecretRef.Key)
	}

	return string(value), nil
}
