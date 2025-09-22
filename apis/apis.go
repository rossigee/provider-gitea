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

// Package apis contains Kubernetes API for the Gitea provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	// v2 APIs (namespaced with .m. API group) - v2-only provider
	accesstokenv2 "github.com/rossigee/provider-gitea/apis/accesstoken/v2"
	actionv2 "github.com/rossigee/provider-gitea/apis/action/v2"
	adminuserv2 "github.com/rossigee/provider-gitea/apis/adminuser/v2"
	branchprotectionv2 "github.com/rossigee/provider-gitea/apis/branchprotection/v2"
	deploykeyv2 "github.com/rossigee/provider-gitea/apis/deploykey/v2"
	githookv2 "github.com/rossigee/provider-gitea/apis/githook/v2"
	issuev2 "github.com/rossigee/provider-gitea/apis/issue/v2"
	labelv2 "github.com/rossigee/provider-gitea/apis/label/v2"
	orgv2 "github.com/rossigee/provider-gitea/apis/organization/v2"
	organizationmemberv2 "github.com/rossigee/provider-gitea/apis/organizationmember/v2"
	orgsecretv2 "github.com/rossigee/provider-gitea/apis/organizationsecret/v2"
	organizationsettingsv2 "github.com/rossigee/provider-gitea/apis/organizationsettings/v2"
	pullrequestv2 "github.com/rossigee/provider-gitea/apis/pullrequest/v2"
	releasev2 "github.com/rossigee/provider-gitea/apis/release/v2"
	giteav2 "github.com/rossigee/provider-gitea/apis/repository/v2"
	repositorycollaboratorv2 "github.com/rossigee/provider-gitea/apis/repositorycollaborator/v2"
	repositorykeyv2 "github.com/rossigee/provider-gitea/apis/repositorykey/v2"
	repositorysecretv2 "github.com/rossigee/provider-gitea/apis/repositorysecret/v2"
	runnerv2 "github.com/rossigee/provider-gitea/apis/runner/v2"
	teamv2 "github.com/rossigee/provider-gitea/apis/team/v2"
	userv2 "github.com/rossigee/provider-gitea/apis/user/v2"
	userkeyv2 "github.com/rossigee/provider-gitea/apis/userkey/v2"
	webhookv2 "github.com/rossigee/provider-gitea/apis/webhook/v2"

	// Provider configuration APIs
	v1alpha1 "github.com/rossigee/provider-gitea/apis/v1alpha1"
	v1beta1 "github.com/rossigee/provider-gitea/apis/v1beta1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		// Provider configuration APIs
		v1alpha1.SchemeBuilder.AddToScheme,
		v1beta1.SchemeBuilder.AddToScheme,

		// v2 APIs (namespaced with .m. API group) - v2-only provider
		giteav2.SchemeBuilder.AddToScheme,
		orgv2.SchemeBuilder.AddToScheme,
		orgsecretv2.SchemeBuilder.AddToScheme,
		userv2.SchemeBuilder.AddToScheme,
		webhookv2.SchemeBuilder.AddToScheme,
		deploykeyv2.SchemeBuilder.AddToScheme,
		teamv2.SchemeBuilder.AddToScheme,
		labelv2.SchemeBuilder.AddToScheme,
		repositorycollaboratorv2.SchemeBuilder.AddToScheme,
		organizationsettingsv2.SchemeBuilder.AddToScheme,
		githookv2.SchemeBuilder.AddToScheme,
		issuev2.SchemeBuilder.AddToScheme,
		pullrequestv2.SchemeBuilder.AddToScheme,
		releasev2.SchemeBuilder.AddToScheme,
		branchprotectionv2.SchemeBuilder.AddToScheme,
		repositorykeyv2.SchemeBuilder.AddToScheme,
		accesstokenv2.SchemeBuilder.AddToScheme,
		repositorysecretv2.SchemeBuilder.AddToScheme,
		userkeyv2.SchemeBuilder.AddToScheme,
		organizationmemberv2.SchemeBuilder.AddToScheme,
		actionv2.SchemeBuilder.AddToScheme,
		adminuserv2.SchemeBuilder.AddToScheme,
		runnerv2.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
