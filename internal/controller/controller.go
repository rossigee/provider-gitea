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

// Package controller wires up every managed-resource controller this provider
// runs. New resource kinds add their Setup call to the slice below.
package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	"github.com/rossigee/provider-gitea/internal/controller/accesstoken"
	"github.com/rossigee/provider-gitea/internal/controller/branchprotection"
	"github.com/rossigee/provider-gitea/internal/controller/githook"
	"github.com/rossigee/provider-gitea/internal/controller/label"
	"github.com/rossigee/provider-gitea/internal/controller/organization"
	"github.com/rossigee/provider-gitea/internal/controller/organizationsecret"
	"github.com/rossigee/provider-gitea/internal/controller/organizationsettings"
	"github.com/rossigee/provider-gitea/internal/controller/repository"
	"github.com/rossigee/provider-gitea/internal/controller/repositorycollaborator"
	"github.com/rossigee/provider-gitea/internal/controller/repositorykey"
	"github.com/rossigee/provider-gitea/internal/controller/repositorysecret"
	"github.com/rossigee/provider-gitea/internal/controller/team"
	"github.com/rossigee/provider-gitea/internal/controller/user"
	"github.com/rossigee/provider-gitea/internal/controller/webhook"
)

// Setup registers every controller this provider runs.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		accesstoken.Setup,
		branchprotection.Setup,
		githook.Setup,
		label.Setup,
		organization.Setup,
		organizationsecret.Setup,
		organizationsettings.Setup,
		repository.Setup,
		repositorycollaborator.Setup,
		repositorykey.Setup,
		repositorysecret.Setup,
		team.Setup,
		user.Setup,
		webhook.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}

	return nil
}
