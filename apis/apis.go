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

	// v1alpha1 APIs (cluster-scoped)
	accesstokenv1alpha1 "github.com/rossigee/provider-gitea/apis/accesstoken/v1alpha1"
	actionv1alpha1 "github.com/rossigee/provider-gitea/apis/action/v1alpha1"
	adminuserv1alpha1 "github.com/rossigee/provider-gitea/apis/adminuser/v1alpha1"
	branchprotectionv1alpha1 "github.com/rossigee/provider-gitea/apis/branchprotection/v1alpha1"
	deploykeyv1alpha1 "github.com/rossigee/provider-gitea/apis/deploykey/v1alpha1"
	githookv1alpha1 "github.com/rossigee/provider-gitea/apis/githook/v1alpha1"
	issuev1alpha1 "github.com/rossigee/provider-gitea/apis/issue/v1alpha1"
	labelv1alpha1 "github.com/rossigee/provider-gitea/apis/label/v1alpha1"
	pullrequestv1alpha1 "github.com/rossigee/provider-gitea/apis/pullrequest/v1alpha1"
	releasev1alpha1 "github.com/rossigee/provider-gitea/apis/release/v1alpha1"
	orgv1alpha1 "github.com/rossigee/provider-gitea/apis/organization/v1alpha1"
	organizationmemberv1alpha1 "github.com/rossigee/provider-gitea/apis/organizationmember/v1alpha1"
	orgsecretv1alpha1 "github.com/rossigee/provider-gitea/apis/organizationsecret/v1alpha1"
	organizationsettingsv1alpha1 "github.com/rossigee/provider-gitea/apis/organizationsettings/v1alpha1"
	giteav1alpha1 "github.com/rossigee/provider-gitea/apis/repository/v1alpha1"
	repositorycollaboratorv1alpha1 "github.com/rossigee/provider-gitea/apis/repositorycollaborator/v1alpha1"
	repositorykeyv1alpha1 "github.com/rossigee/provider-gitea/apis/repositorykey/v1alpha1"
	repositorysecretv1alpha1 "github.com/rossigee/provider-gitea/apis/repositorysecret/v1alpha1"
	runnerv1alpha1 "github.com/rossigee/provider-gitea/apis/runner/v1alpha1"
	teamv1alpha1 "github.com/rossigee/provider-gitea/apis/team/v1alpha1"
	userv1alpha1 "github.com/rossigee/provider-gitea/apis/user/v1alpha1"
	userkeyv1alpha1 "github.com/rossigee/provider-gitea/apis/userkey/v1alpha1"
	webhookv1alpha1 "github.com/rossigee/provider-gitea/apis/webhook/v1alpha1"

	// v2 APIs (namespaced with .m. API group)
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

		// v1alpha1 APIs (cluster-scoped) - backward compatibility
		giteav1alpha1.SchemeBuilder.AddToScheme,
		orgv1alpha1.SchemeBuilder.AddToScheme,
		orgsecretv1alpha1.SchemeBuilder.AddToScheme,
		userv1alpha1.SchemeBuilder.AddToScheme,
		webhookv1alpha1.SchemeBuilder.AddToScheme,
		deploykeyv1alpha1.SchemeBuilder.AddToScheme,
		teamv1alpha1.SchemeBuilder.AddToScheme,
		labelv1alpha1.SchemeBuilder.AddToScheme,
		repositorycollaboratorv1alpha1.SchemeBuilder.AddToScheme,
		organizationsettingsv1alpha1.SchemeBuilder.AddToScheme,
		githookv1alpha1.SchemeBuilder.AddToScheme,
		issuev1alpha1.SchemeBuilder.AddToScheme,
		pullrequestv1alpha1.SchemeBuilder.AddToScheme,
		releasev1alpha1.SchemeBuilder.AddToScheme,
		branchprotectionv1alpha1.SchemeBuilder.AddToScheme,
		repositorykeyv1alpha1.SchemeBuilder.AddToScheme,
		accesstokenv1alpha1.SchemeBuilder.AddToScheme,
		repositorysecretv1alpha1.SchemeBuilder.AddToScheme,
		userkeyv1alpha1.SchemeBuilder.AddToScheme,
		organizationmemberv1alpha1.SchemeBuilder.AddToScheme,
		actionv1alpha1.SchemeBuilder.AddToScheme,
		adminuserv1alpha1.SchemeBuilder.AddToScheme,
		runnerv1alpha1.SchemeBuilder.AddToScheme,

		// v2 APIs (namespaced with .m. API group) - native v2 support
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
