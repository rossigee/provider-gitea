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

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/pkg/controller"

	"github.com/crossplane-contrib/provider-gitea/internal/controller/accesstoken"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/action"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/adminuser"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/branchprotection"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/deploykey"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/githook"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/label"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/organization"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/organizationmember"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/organizationsecret"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/organizationsettings"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/repository"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/repositorycollaborator"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/repositorykey"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/repositorysecret"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/runner"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/team"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/user"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/userkey"
	"github.com/crossplane-contrib/provider-gitea/internal/controller/webhook"
)

// Setup creates all Gitea controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		repository.Setup,
		organization.Setup,
		organizationsecret.Setup,
		user.Setup,
		webhook.Setup,
		deploykey.Setup,
		team.Setup,
		label.Setup,
		repositorycollaborator.Setup,
		organizationsettings.Setup,
		githook.Setup,
		branchprotection.Setup,
		repositorykey.Setup,
		accesstoken.Setup,
		repositorysecret.Setup,
		userkey.Setup,
		organizationmember.Setup,
		action.Setup,
		runner.Setup,
		adminuser.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
