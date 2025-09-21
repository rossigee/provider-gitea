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

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	"github.com/rossigee/provider-gitea/internal/controller/accesstoken"
	"github.com/rossigee/provider-gitea/internal/controller/action"
	"github.com/rossigee/provider-gitea/internal/controller/adminuser"
	"github.com/rossigee/provider-gitea/internal/controller/branchprotection"
	"github.com/rossigee/provider-gitea/internal/controller/deploykey"
	"github.com/rossigee/provider-gitea/internal/controller/githook"
	"github.com/rossigee/provider-gitea/internal/controller/issue"
	"github.com/rossigee/provider-gitea/internal/controller/label"
	"github.com/rossigee/provider-gitea/internal/controller/pullrequest"
	"github.com/rossigee/provider-gitea/internal/controller/release"
	"github.com/rossigee/provider-gitea/internal/controller/organization"
	"github.com/rossigee/provider-gitea/internal/controller/organizationmember"
	"github.com/rossigee/provider-gitea/internal/controller/organizationsecret"
	"github.com/rossigee/provider-gitea/internal/controller/organizationsettings"
	"github.com/rossigee/provider-gitea/internal/controller/repository"
	repositoryv2 "github.com/rossigee/provider-gitea/internal/controller/repository/v2"
	"github.com/rossigee/provider-gitea/internal/controller/repositorycollaborator"
	"github.com/rossigee/provider-gitea/internal/controller/repositorykey"
	"github.com/rossigee/provider-gitea/internal/controller/repositorysecret"
	"github.com/rossigee/provider-gitea/internal/controller/runner"
	"github.com/rossigee/provider-gitea/internal/controller/team"
	"github.com/rossigee/provider-gitea/internal/controller/user"
	"github.com/rossigee/provider-gitea/internal/controller/userkey"
	"github.com/rossigee/provider-gitea/internal/controller/webhook"
)

// Setup creates all Gitea controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		repository.Setup,
		repositoryv2.Setup, // V2 namespaced repository controller
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
		issue.Setup,
		pullrequest.Setup,
		release.Setup,
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
