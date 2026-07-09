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
	accesstokenv2 "github.com/rossigee/provider-gitea/apis/accesstoken/v1beta1"
	branchprotectionv2 "github.com/rossigee/provider-gitea/apis/branchprotection/v1beta1"
	githookv2 "github.com/rossigee/provider-gitea/apis/githook/v1beta1"
	labelv2 "github.com/rossigee/provider-gitea/apis/label/v1beta1"
	orgv2 "github.com/rossigee/provider-gitea/apis/organization/v1beta1"
	orgsecretv2 "github.com/rossigee/provider-gitea/apis/organizationsecret/v1beta1"
	organizationsettingsv2 "github.com/rossigee/provider-gitea/apis/organizationsettings/v1beta1"
	giteav2 "github.com/rossigee/provider-gitea/apis/repository/v1beta1"
	repositorycollaboratorv2 "github.com/rossigee/provider-gitea/apis/repositorycollaborator/v1beta1"
	repositorykeyv2 "github.com/rossigee/provider-gitea/apis/repositorykey/v1beta1"
	repositorysecretv2 "github.com/rossigee/provider-gitea/apis/repositorysecret/v1beta1"
	teamv2 "github.com/rossigee/provider-gitea/apis/team/v1beta1"
	teammembershipv2 "github.com/rossigee/provider-gitea/apis/teammembership/v1beta1"
	teamrepositoryv2 "github.com/rossigee/provider-gitea/apis/teamrepository/v1beta1"
	userv2 "github.com/rossigee/provider-gitea/apis/user/v1beta1"
	variablev2 "github.com/rossigee/provider-gitea/apis/variable/v1beta1"
	webhookv2 "github.com/rossigee/provider-gitea/apis/webhook/v1beta1"

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
		teamv2.SchemeBuilder.AddToScheme,
		labelv2.SchemeBuilder.AddToScheme,
		repositorycollaboratorv2.SchemeBuilder.AddToScheme,
		organizationsettingsv2.SchemeBuilder.AddToScheme,
		githookv2.SchemeBuilder.AddToScheme,
		branchprotectionv2.SchemeBuilder.AddToScheme,
		repositorykeyv2.SchemeBuilder.AddToScheme,
		accesstokenv2.SchemeBuilder.AddToScheme,
		repositorysecretv2.SchemeBuilder.AddToScheme,
		variablev2.SchemeBuilder.AddToScheme,
		teammembershipv2.SchemeBuilder.AddToScheme,
		teamrepositoryv2.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
