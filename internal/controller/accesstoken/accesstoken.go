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

package accesstoken

import (
	"context"
	"fmt"
	"reflect"

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

	"github.com/crossplane-contrib/provider-gitea/apis/accesstoken/v1alpha1"
	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
	giteaclients "github.com/crossplane-contrib/provider-gitea/internal/clients"
)

const (
	errNotAccessToken = "managed resource is not a AccessToken custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
	errNewClient      = "cannot create new Service"
	errGetAccessToken = "cannot get access token"
	errCreateAccessToken = "cannot create access token"
	errUpdateAccessToken = "cannot update access token"
	errDeleteAccessToken = "cannot delete access token"
)

// Setup adds a controller that reconciles AccessToken managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.AccessTokenKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.AccessTokenGroupVersionKind),
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
		For(&v1alpha1.AccessToken{}).
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
	cr, ok := mg.(*v1alpha1.AccessToken)
	if !ok {
		return nil, errors.New(errNotAccessToken)
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
	cr, ok := mg.(*v1alpha1.AccessToken)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAccessToken)
	}

	// External name format: username/token_id (e.g., "myuser/123")
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		// No external name set yet, resource doesn't exist
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Parse token ID from external name (username/token_id format)
	var tokenID int64
	var err error
	if _, err = fmt.Sscanf(externalName, cr.Spec.ForProvider.Username+"/%d", &tokenID); err != nil {
		// If we can't parse the ID, assume the resource doesn't exist yet
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	token, err := c.client.GetAccessToken(ctx, cr.Spec.ForProvider.Username, tokenID)
	if err != nil {
		// If token doesn't exist, it needs to be created
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.AccessTokenObservation{
		ID:             &token.ID,
		Name:           &token.Name,
		Scopes:         token.Scopes,
		TokenLastEight: &token.TokenLastEight,
		Username:       &cr.Spec.ForProvider.Username,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, token),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.AccessToken)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAccessToken)
	}

	cr.SetConditions(xpv1.Creating())

	req := c.buildCreateRequest(cr)

	token, err := c.client.CreateAccessToken(ctx, cr.Spec.ForProvider.Username, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAccessToken)
	}

	// Set external name to username/token_id format
	externalName := fmt.Sprintf("%s/%d", cr.Spec.ForProvider.Username, token.ID)
	meta.SetExternalName(cr, externalName)

	// Store the actual token value in a connection secret
	connectionDetails := managed.ConnectionDetails{
		"token": []byte(token.Token),
	}

	return managed.ExternalCreation{
		ConnectionDetails: connectionDetails,
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.AccessToken)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAccessToken)
	}

	// Parse token ID from external name
	externalName := meta.GetExternalName(cr)
	var tokenID int64
	if _, err := fmt.Sscanf(externalName, cr.Spec.ForProvider.Username+"/%d", &tokenID); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot parse token ID from external name")
	}

	req := c.buildUpdateRequest(cr)

	_, err := c.client.UpdateAccessToken(ctx, cr.Spec.ForProvider.Username, tokenID, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAccessToken)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.AccessToken)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAccessToken)
	}

	// Parse token ID from external name
	externalName := meta.GetExternalName(cr)
	var tokenID int64
	if _, err := fmt.Sscanf(externalName, cr.Spec.ForProvider.Username+"/%d", &tokenID); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot parse token ID from external name")
	}

	err := c.client.DeleteAccessToken(ctx, cr.Spec.ForProvider.Username, tokenID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAccessToken)
	}

	return managed.ExternalDelete{}, nil
}

// buildCreateRequest creates a create request from the CR spec
func (c *external) buildCreateRequest(cr *v1alpha1.AccessToken) *giteaclients.CreateAccessTokenRequest {
	return &giteaclients.CreateAccessTokenRequest{
		Name:   cr.Spec.ForProvider.Name,
		Scopes: cr.Spec.ForProvider.Scopes,
	}
}

// buildUpdateRequest creates an update request from the CR spec
func (c *external) buildUpdateRequest(cr *v1alpha1.AccessToken) *giteaclients.UpdateAccessTokenRequest {
	return &giteaclients.UpdateAccessTokenRequest{
		Name:   &cr.Spec.ForProvider.Name,
		Scopes: cr.Spec.ForProvider.Scopes,
	}
}

// isUpToDate checks if the access token is up to date with the desired state
func (c *external) isUpToDate(cr *v1alpha1.AccessToken, token *giteaclients.AccessToken) bool {
	if cr.Spec.ForProvider.Name != token.Name {
		return false
	}
	if !reflect.DeepEqual(cr.Spec.ForProvider.Scopes, token.Scopes) {
		return false
	}

	return true
}