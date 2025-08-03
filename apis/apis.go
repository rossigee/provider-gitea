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

	giteav1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/repository/v1alpha1"
	orgv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/organization/v1alpha1"
	orgsecretv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/organizationsecret/v1alpha1"
	userv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/user/v1alpha1"
	webhookv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/webhook/v1alpha1"
	deploykeyv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/deploykey/v1alpha1"
	teamv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/team/v1alpha1"
	labelv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/label/v1alpha1"
	repositorycollaboratorv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/repositorycollaborator/v1alpha1"
	organizationsettingsv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/organizationsettings/v1alpha1"
	githookv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/githook/v1alpha1"
	branchprotectionv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/branchprotection/v1alpha1"
	repositorykeyv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/repositorykey/v1alpha1"
	accesstokenv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/accesstoken/v1alpha1"
	repositorysecretv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/repositorysecret/v1alpha1"
	userkeyv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/userkey/v1alpha1"
	organizationmemberv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/organizationmember/v1alpha1"
	actionv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/action/v1alpha1"
	runnerv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/runner/v1alpha1"
	adminuserv1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/adminuser/v1alpha1"
	v1alpha1 "github.com/crossplane-contrib/provider-gitea/apis/v1alpha1"
	v1beta1 "github.com/crossplane-contrib/provider-gitea/apis/v1beta1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		v1alpha1.SchemeBuilder.AddToScheme,
		v1beta1.SchemeBuilder.AddToScheme,
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
		branchprotectionv1alpha1.SchemeBuilder.AddToScheme,
		repositorykeyv1alpha1.SchemeBuilder.AddToScheme,
		accesstokenv1alpha1.SchemeBuilder.AddToScheme,
		repositorysecretv1alpha1.SchemeBuilder.AddToScheme,
		userkeyv1alpha1.SchemeBuilder.AddToScheme,
		organizationmemberv1alpha1.SchemeBuilder.AddToScheme,
		actionv1alpha1.SchemeBuilder.AddToScheme,
		runnerv1alpha1.SchemeBuilder.AddToScheme,
		adminuserv1alpha1.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}