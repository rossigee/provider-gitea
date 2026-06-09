package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type ResourceInfo struct {
	Name       string
	Type       string
	Package    string
	APIVersion string
}

const controllerTemplate = `/*
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

package {{.Package}}

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-gitea/apis/{{.Package}}/{{.APIVersion}}"
	"github.com/rossigee/provider-gitea/apis/v1beta1"
	"github.com/rossigee/provider-gitea/internal/clients"
)

const (
	errNot{{.Type}}         = "managed resource is not a {{.Type}} custom resource"
	errGet{{.Type}}         = "failed to get {{.Name}}"
	errCreate{{.Type}}      = "failed to create {{.Name}}"
	errUpdate{{.Type}}      = "failed to update {{.Name}}"
	errDelete{{.Type}}      = "failed to delete {{.Name}}"
	errGetProviderConfig = "failed to get provider config"
)

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*{{.APIVersion}}.{{.Type}})
	if !ok {
		return nil, errors.New(errNot{{.Type}})
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

	return &externalClient{client: conn}, nil
}

type externalClient struct {
	client clients.Client
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*{{.APIVersion}}.{{.Type}})
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNot{{.Type}})
	}

	externalID := meta.GetExternalName(cr)
	if externalID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// TODO: Implement actual observation logic for {{.Type}}
	// This is a stub that marks resource as existing and up-to-date
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*{{.APIVersion}}.{{.Type}})
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNot{{.Type}})
	}

	// TODO: Implement creation logic for {{.Type}}
	externalID := cr.GetName()
	meta.SetExternalName(cr, externalID)

	return managed.ExternalCreation{}, errors.New("{{.Type}} controller not yet fully implemented")
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mg.(*{{.APIVersion}}.{{.Type}})
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNot{{.Type}})
	}

	// TODO: Implement update logic for {{.Type}}
	return managed.ExternalUpdate{}, errors.New("{{.Type}} controller not yet fully implemented")
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, ok := mg.(*{{.APIVersion}}.{{.Type}})
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNot{{.Type}})
	}

	// TODO: Implement deletion logic for {{.Type}}
	return managed.ExternalDelete{}, errors.New("{{.Type}} controller not yet fully implemented")
}

func (e *externalClient) Disconnect(ctx context.Context) error {
	return nil
}

func Setup(mgr ctrl.Manager, o xpv1.Options) error {
	name := managed.ControllerName({{.APIVersion}}.{{.Type}}Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind({{.APIVersion}}.{{.Type}}GroupVersionKind),
		managed.WithExternalConnector(&connector{kube: mgr.GetClient()}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&{{.APIVersion}}.{{.Type}}{}).
		Complete(r)
}
`

func main() {
	resources := []ResourceInfo{
		{"organization", "Organization", "organization", "v2"},
		{"team", "Team", "team", "v2"},
		{"deploykey", "DeployKey", "deploykey", "v2"},
		{"user", "User", "user", "v2"},
		{"label", "Label", "label", "v2"},
		{"webhook", "Webhook", "webhook", "v2"},
		{"accesstoken", "AccessToken", "accesstoken", "v2"},
		{"userkey", "UserKey", "userkey", "v2"},
		{"repositorykey", "RepositoryKey", "repositorykey", "v2"},
		{"repositorysecret", "RepositorySecret", "repositorysecret", "v2"},
		{"organizationsecret", "OrganizationSecret", "organizationsecret", "v2"},
		{"branchprotection", "BranchProtection", "branchprotection", "v2"},
		{"organizationmember", "OrganizationMember", "organizationmember", "v2"},
		{"action", "Action", "action", "v2"},
		{"runner", "Runner", "runner", "v2"},
		{"adminuser", "AdminUser", "adminuser", "v2"},
		{"organizationsettings", "OrganizationSettings", "organizationsettings", "v2"},
		{"githook", "GitHook", "githook", "v2"},
		{"repositorycollaborator", "RepositoryCollaborator", "repositorycollaborator", "v2"},
		{"issue", "Issue", "issue", "v2"},
		{"pullrequest", "PullRequest", "pullrequest", "v2"},
	}

	tmpl, err := template.New("controller").Parse(controllerTemplate)
	if err != nil {
		fmt.Printf("Template error: %v\n", err)
		return
	}

	fmt.Println("Generating controller stubs...")

	succeeded := 0
	for _, r := range resources {
		dir := filepath.Join("internal", "controller", r.Package)
		os.MkdirAll(dir, 0755)

		filePath := filepath.Join(dir, strings.ToLower(r.Package)+".go")
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("❌ %s: %v\n", r.Type, err)
			continue
		}
		defer file.Close()

		if err := tmpl.Execute(file, r); err != nil {
			fmt.Printf("❌ %s: %v\n", r.Type, err)
			continue
		}

		fmt.Printf("✅ %s\n", r.Type)
		succeeded++
	}

	fmt.Printf("\n✅ Generated %d controllers\n", succeeded)
}
