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

package release

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/release/v1alpha1"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	giteaclients "github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNotRelease    = "managed resource is not a Release custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errNewClient     = "cannot create new Service"
	errCreateRelease = "cannot create release"
	errUpdateRelease = "cannot update release"
	errDeleteRelease = "cannot delete release"
	errGetRelease    = "cannot get release"
	errAssetUpload   = "cannot upload release asset"
)

// Setup adds a controller that reconciles Release managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ReleaseKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ReleaseGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.TrackerFn(func(ctx context.Context, mg resource.Managed) error { return nil }),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.Release{}).
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
	cr, ok := mg.(*v1alpha1.Release)
	if !ok {
		return nil, errors.New(errNotRelease)
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
	cr, ok := mg.(*v1alpha1.Release)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRelease)
	}

	externalName := meta.GetExternalName(cr)
	
	// If no external name is set, try to find by tag name
	if externalName == "" {
		// Try to find release by tag name
		release, err := c.client.GetReleaseByTag(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, cr.Spec.ForProvider.TagName)
		if err != nil {
			if giteaclients.IsNotFound(err) {
				return managed.ExternalObservation{
					ResourceExists: false,
				}, nil
			}
			return managed.ExternalObservation{}, errors.Wrap(err, errGetRelease)
		}

		// Found by tag, set external name to ID
		meta.SetExternalName(cr, fmt.Sprintf("%d", release.ID))
		externalName = fmt.Sprintf("%d", release.ID)
	}

	// Parse the external name as the release ID
	releaseID, err := strconv.ParseInt(externalName, 10, 64)
	if err != nil {
		// If external name is not a valid number, treat as not created yet
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Get the release from Gitea
	release, err := c.client.GetRelease(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, releaseID)
	if err != nil {
		if giteaclients.IsNotFound(err) {
			// Release doesn't exist, mark for recreation
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRelease)
	}

	// Update observed state
	cr.Status.AtProvider = generateReleaseObservation(release)

	// Check if resource needs update
	upToDate := isReleaseUpToDate(cr, release)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        upToDate,
		ResourceLateInitialized: false,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Release)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRelease)
	}

	// Create release in Gitea
	createOpts := &giteaclients.CreateReleaseOptions{
		TagName:         cr.Spec.ForProvider.TagName,
		TargetCommitish: stringValueOrEmpty(cr.Spec.ForProvider.TargetCommitish),
		Name:            stringValueOrEmpty(cr.Spec.ForProvider.Name),
		Body:            stringValueOrEmpty(cr.Spec.ForProvider.Body),
		Draft:           boolValue(cr.Spec.ForProvider.Draft),
		Prerelease:      boolValue(cr.Spec.ForProvider.Prerelease),
		GenerateNotes:   boolValue(cr.Spec.ForProvider.GenerateNotes),
	}

	release, err := c.client.CreateRelease(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, createOpts)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRelease)
	}

	// Set external name annotation
	meta.SetExternalName(cr, fmt.Sprintf("%d", release.ID))

	// Handle asset uploads if specified
	if len(cr.Spec.ForProvider.Assets) > 0 {
		if err := c.uploadAssets(ctx, cr, release.ID); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errAssetUpload)
		}
	}

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Release)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRelease)
	}

	releaseID, err := strconv.ParseInt(meta.GetExternalName(cr), 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRelease)
	}

	// Update release in Gitea
	updateOpts := &giteaclients.UpdateReleaseOptions{
		TargetCommitish: cr.Spec.ForProvider.TargetCommitish,
		Name:            cr.Spec.ForProvider.Name,
		Body:            cr.Spec.ForProvider.Body,
		Draft:           cr.Spec.ForProvider.Draft,
		Prerelease:      cr.Spec.ForProvider.Prerelease,
		GenerateNotes:   cr.Spec.ForProvider.GenerateNotes,
	}
	
	// Set TagName only if not empty
	if cr.Spec.ForProvider.TagName != "" {
		updateOpts.TagName = &cr.Spec.ForProvider.TagName
	}

	_, err = c.client.UpdateRelease(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, releaseID, updateOpts)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRelease)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Release)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRelease)
	}

	releaseID, err := strconv.ParseInt(meta.GetExternalName(cr), 10, 64)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRelease)
	}

	// Delete the release
	err = c.client.DeleteRelease(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, releaseID)
	if err != nil && !giteaclients.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRelease)
	}

	return managed.ExternalDelete{}, nil
}

// uploadAssets handles uploading assets to a release
func (c *external) uploadAssets(ctx context.Context, cr *v1alpha1.Release, releaseID int64) error {
	for _, asset := range cr.Spec.ForProvider.Assets {
		var content []byte
		var err error

		// Handle content from inline base64 or URL
		if asset.Content != nil {
			content, err = base64.StdEncoding.DecodeString(*asset.Content)
			if err != nil {
				return errors.Wrapf(err, "failed to decode base64 content for asset %s", asset.Name)
			}
		} else if asset.URL != nil {
			// In a full implementation, we would fetch content from URL
			// For now, we'll skip URL-based assets
			continue
		} else {
			// No content source specified, skip
			continue
		}

		contentType := "application/octet-stream"
		if asset.ContentType != nil {
			contentType = *asset.ContentType
		}

		_, err = c.client.CreateReleaseAttachment(ctx, cr.Spec.ForProvider.Owner, cr.Spec.ForProvider.Repository, releaseID, asset.Name, contentType, content)
		if err != nil {
			return errors.Wrapf(err, "failed to upload asset %s", asset.Name)
		}
	}

	return nil
}

// generateReleaseObservation creates ReleaseObservation from Gitea release
func generateReleaseObservation(release *giteaclients.Release) v1alpha1.ReleaseObservation {
	obs := v1alpha1.ReleaseObservation{
		ID:              release.ID,
		TagName:         release.TagName,
		TargetCommitish: release.TargetCommitish,
		Name:            release.Name,
		Body:            release.Body,
		URL:             release.URL,
		HTMLURL:         release.HTMLURL,
		TarballURL:      release.TarballURL,
		ZipballURL:      release.ZipballURL,
		UploadURL:       release.UploadURL,
		Draft:           release.Draft,
		Prerelease:      release.Prerelease,
		CreatedAt:       release.CreatedAt,
		PublishedAt:     release.PublishedAt,
	}

	// Add author info if available
	if release.Author != nil {
		obs.Author = release.Author.Username
	}

	// Convert assets
	for _, asset := range release.Assets {
		obs.Assets = append(obs.Assets, v1alpha1.ReleaseAssetObservation{
			ID:                 asset.ID,
			Name:               asset.Name,
			Size:               asset.Size,
			DownloadCount:      asset.DownloadCount,
			ContentType:        asset.ContentType,
			BrowserDownloadURL: asset.BrowserDownloadURL,
			CreatedAt:          asset.CreatedAt,
			UpdatedAt:          asset.UpdatedAt,
		})
	}

	return obs
}

// isReleaseUpToDate checks if the release is up to date with desired state
func isReleaseUpToDate(cr *v1alpha1.Release, release *giteaclients.Release) bool {
	// Check tag name
	if cr.Spec.ForProvider.TagName != release.TagName {
		return false
	}

	// Check name
	if cr.Spec.ForProvider.Name != nil && *cr.Spec.ForProvider.Name != release.Name {
		return false
	}

	// Check body
	if cr.Spec.ForProvider.Body != nil && *cr.Spec.ForProvider.Body != release.Body {
		return false
	}

	// Check draft status
	if cr.Spec.ForProvider.Draft != nil && *cr.Spec.ForProvider.Draft != release.Draft {
		return false
	}

	// Check prerelease status
	if cr.Spec.ForProvider.Prerelease != nil && *cr.Spec.ForProvider.Prerelease != release.Prerelease {
		return false
	}

	// TODO: Add more detailed comparison for assets and other fields

	return true
}

// Utility functions for handling pointer values
func stringValueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func boolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}