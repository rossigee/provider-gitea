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
)

// Setup creates all Gitea v2 controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	// NOTE: Controller implementations pending v2.3.2 API pattern clarification
	// See CONTROLLER_IMPLEMENTATION_GUIDE.md for detailed implementation plan
	//
	// Phase 1: Core Resources (7) - Repository, Organization, Team, DeployKey, User, Label, Webhook
	// Phase 2: Security Resources (7) - AccessToken, UserKey, RepositoryKey, RepositorySecret, OrganizationSecret, BranchProtection, OrganizationMember
	// Phase 3: CI/CD & Admin (8) - Action, Runner, AdminUser, OrganizationSettings, GitHook, RepositoryCollaborator, Issue, PullRequest, Release

	// TODO: Implement 22 resource controllers

	return nil
}
