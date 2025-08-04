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

package label

import (
	"context"
	"strconv"
	"strings"

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

	"github.com/crossplane-contrib/provider-gitea/apis/label/v1alpha1"
	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
	giteaclients "github.com/crossplane-contrib/provider-gitea/internal/clients"
)

const (
	errNotLabel     = "managed resource is not a Label custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create new Service"
	errCreateLabel  = "cannot create label"
	errUpdateLabel  = "cannot update label"
	errDeleteLabel  = "cannot delete label"
	errGetLabel     = "cannot get label"
	errParseLabelID = "cannot parse label ID"
	errParseRepo    = "cannot parse repository (expected owner/repo format)"
)

// Setup adds a controller that reconciles Label managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.LabelKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.LabelGroupVersionKind),
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
		For(&v1alpha1.Label{}).
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
	cr, ok := mg.(*v1alpha1.Label)
	if !ok {
		return nil, errors.New(errNotLabel)
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
	cr, ok := mg.(*v1alpha1.Label)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotLabel)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Parse repository owner/name
	repoParts := strings.SplitN(cr.Spec.ForProvider.Repository, "/", 2)
	if len(repoParts) != 2 {
		return managed.ExternalObservation{ResourceExists: false}, errors.New(errParseRepo)
	}
	owner, repo := repoParts[0], repoParts[1]

	// Parse label ID from external name
	labelID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, errors.Wrap(err, errParseLabelID)
	}

	label, err := c.client.GetLabel(ctx, owner, repo, labelID)
	if err != nil {
		// If label doesn't exist, return that it needs to be created
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.LabelObservation{
		ID:  &label.ID,
		URL: &label.URL,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: c.isUpToDate(cr, label),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Label)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotLabel)
	}

	cr.SetConditions(xpv1.Creating())

	// Parse repository owner/name
	repoParts := strings.SplitN(cr.Spec.ForProvider.Repository, "/", 2)
	if len(repoParts) != 2 {
		return managed.ExternalCreation{}, errors.New(errParseRepo)
	}
	owner, repo := repoParts[0], repoParts[1]

	req := &giteaclients.CreateLabelRequest{
		Name:  cr.Spec.ForProvider.Name,
		Color: cr.Spec.ForProvider.Color,
	}

	if cr.Spec.ForProvider.Description != nil {
		req.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Exclusive != nil {
		req.Exclusive = *cr.Spec.ForProvider.Exclusive
	}

	label, err := c.client.CreateLabel(ctx, owner, repo, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateLabel)
	}

	// Set external name to the label ID
	meta.SetExternalName(cr, strconv.FormatInt(label.ID, 10))

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Label)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotLabel)
	}

	// Parse repository owner/name
	repoParts := strings.SplitN(cr.Spec.ForProvider.Repository, "/", 2)
	if len(repoParts) != 2 {
		return managed.ExternalUpdate{}, errors.New(errParseRepo)
	}
	owner, repo := repoParts[0], repoParts[1]

	externalName := meta.GetExternalName(cr)
	labelID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errParseLabelID)
	}

	req := &giteaclients.UpdateLabelRequest{}

	if cr.Spec.ForProvider.Name != "" {
		req.Name = &cr.Spec.ForProvider.Name
	}
	if cr.Spec.ForProvider.Color != "" {
		req.Color = &cr.Spec.ForProvider.Color
	}
	if cr.Spec.ForProvider.Description != nil {
		req.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Exclusive != nil {
		req.Exclusive = cr.Spec.ForProvider.Exclusive
	}

	_, err = c.client.UpdateLabel(ctx, owner, repo, labelID, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateLabel)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Label)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotLabel)
	}

	cr.SetConditions(xpv1.Deleting())

	// Parse repository owner/name
	repoParts := strings.SplitN(cr.Spec.ForProvider.Repository, "/", 2)
	if len(repoParts) != 2 {
		return managed.ExternalDelete{}, errors.New(errParseRepo)
	}
	owner, repo := repoParts[0], repoParts[1]

	externalName := meta.GetExternalName(cr)
	labelID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errParseLabelID)
	}

	err = c.client.DeleteLabel(ctx, owner, repo, labelID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteLabel)
	}

	return managed.ExternalDelete{}, nil
}

// isUpToDate checks if the label is up to date with the desired state
func (c *external) isUpToDate(cr *v1alpha1.Label, label *giteaclients.Label) bool {
	if cr.Spec.ForProvider.Name != label.Name {
		return false
	}
	if cr.Spec.ForProvider.Color != label.Color {
		return false
	}
	if cr.Spec.ForProvider.Description != nil && *cr.Spec.ForProvider.Description != label.Description {
		return false
	}
	if cr.Spec.ForProvider.Exclusive != nil && *cr.Spec.ForProvider.Exclusive != label.Exclusive {
		return false
	}

	return true
}
