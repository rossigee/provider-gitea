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

package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
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

	"github.com/crossplane-contrib/provider-gitea/apis/action/v1alpha1"
	"github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
	"github.com/crossplane-contrib/provider-gitea/internal/clients"
)

const (
	errNotAction    = "managed resource is not a Action custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create new Service"
)

// Setup adds a controller that reconciles Action managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ActionKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ActionGroupVersionKind),
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
		For(&v1alpha1.Action{}).
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
	cr, ok := mg.(*v1alpha1.Action)
	if !ok {
		return nil, errors.New(errNotAction)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	client, err := clients.NewClient(ctx, pc, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: client}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client clients.Client
}

func (c *external) Disconnect(ctx context.Context) error {
	// No persistent connection to disconnect
	return nil
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Action)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAction)
	}

	// Get external name from annotations
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{}, nil
	}

	// Parse external name format: repository/workflow-name
	repo, workflowName, err := parseExternalName(externalName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot parse external name")
	}

	action, err := c.client.GetAction(ctx, repo, workflowName)
	if clients.IsNotFound(err) {
		return managed.ExternalObservation{}, nil
	}
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get action")
	}

	// Update observation
	current := cr.Spec.ForProvider.DeepCopy()
	cr.Status.AtProvider = generateObservation(action)

	// Resource exists and is available
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        isUpToDate(&cr.Spec.ForProvider, action),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(&cr.Spec.ForProvider, current),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Action)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAction)
	}

	cr.Status.SetConditions(xpv1.Creating())

	// Build create request
	req := &clients.CreateActionRequest{
		WorkflowName: cr.Spec.ForProvider.WorkflowName,
		WorkflowFile: cr.Spec.ForProvider.Content,
		Path:         fmt.Sprintf(".github/workflows/%s", cr.Spec.ForProvider.WorkflowName),
	}

	if cr.Spec.ForProvider.CommitMessage != nil {
		req.Message = *cr.Spec.ForProvider.CommitMessage
	}
	if cr.Spec.ForProvider.Branch != nil {
		req.Branch = *cr.Spec.ForProvider.Branch
	}

	action, err := c.client.CreateAction(ctx, cr.Spec.ForProvider.Repository, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create action")
	}

	// Set external name
	externalName := fmt.Sprintf("%s/%s", cr.Spec.ForProvider.Repository, action.WorkflowName)
	meta.SetExternalName(cr, externalName)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Action)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAction)
	}

	// Get external name
	externalName := meta.GetExternalName(cr)
	repo, workflowName, err := parseExternalName(externalName)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot parse external name")
	}

	// Build update request
	req := &clients.UpdateActionRequest{}

	if cr.Spec.ForProvider.Content != "" {
		req.WorkflowFile = &cr.Spec.ForProvider.Content
	}
	if cr.Spec.ForProvider.CommitMessage != nil {
		req.Message = cr.Spec.ForProvider.CommitMessage
	}
	if cr.Spec.ForProvider.Branch != nil {
		req.Branch = cr.Spec.ForProvider.Branch
	}

	_, err = c.client.UpdateAction(ctx, repo, workflowName, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update action")
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Action)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAction)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	// Get external name
	externalName := meta.GetExternalName(cr)
	repo, workflowName, err := parseExternalName(externalName)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot parse external name")
	}

	err = c.client.DeleteAction(ctx, repo, workflowName)
	if clients.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete action")
}

// Helper functions

func parseExternalName(externalName string) (repo, workflowName string, err error) {
	parts := strings.Split(externalName, "/")
	if len(parts) != 3 {
		return "", "", errors.New("external name must be in format 'owner/repo/workflow-name'")
	}
	return fmt.Sprintf("%s/%s", parts[0], parts[1]), parts[2], nil
}

func generateObservation(action *clients.Action) v1alpha1.ActionObservation {
	obs := v1alpha1.ActionObservation{
		WorkflowName: &action.WorkflowName,
		State:        &action.State,
		BadgeURL:     &action.Badge,
		CreatedAt:    &action.CreatedAt,
		UpdatedAt:    &action.UpdatedAt,
	}

	if action.LastRun != nil {
		obs.LastRun = &v1alpha1.ActionLastRun{
			ID:         &action.LastRun.ID,
			RunNumber:  &action.LastRun.Number,
			Status:     &action.LastRun.Status,
			Conclusion: &action.LastRun.Conclusion,
			Event:      &action.LastRun.Event,
			CreatedAt:  &action.LastRun.StartedAt,
			UpdatedAt:  &action.LastRun.UpdatedAt,
		}
	}

	return obs
}

func isUpToDate(spec *v1alpha1.ActionParameters, action *clients.Action) bool {
	return spec.Content == action.WorkflowFile.Content
}
