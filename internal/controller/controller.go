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
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/rossigee/provider-gitea/internal/controller/organization"
	"github.com/rossigee/provider-gitea/internal/controller/providerconfig"
	"github.com/rossigee/provider-gitea/internal/controller/repository"
	"github.com/rossigee/provider-gitea/internal/controller/repositorykey"
	"github.com/rossigee/provider-gitea/internal/controller/repositorysecret"
	"github.com/rossigee/provider-gitea/internal/controller/user"
	"github.com/rossigee/provider-gitea/internal/controller/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	if err := providerconfig.Setup(mgr); err != nil {
		return err
	}
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		repository.Setup,
		repositorykey.Setup,
		repositorysecret.Setup,
		organization.Setup,
		user.Setup,
		webhook.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}

	return nil
}
