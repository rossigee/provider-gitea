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

// Package features provides feature flags for the Gitea provider.
package features

import "github.com/crossplane/crossplane-runtime/pkg/feature"

// Feature flags.
const (
	// EnableAlphaExternalSecretStores enables alpha support for
	// External Secret Stores. See the below design document for more details:
	// https://github.com/crossplane/crossplane/blob/master/design/design-doc-external-secret-stores.md
	EnableAlphaExternalSecretStores feature.Flag = "EnableAlphaExternalSecretStores"

	// EnableAlphaManagementPolicies enables alpha support for
	// Management Policies. See the below design document for more details:
	// https://github.com/crossplane/crossplane/blob/master/design/one-pager-ignore-changes.md
	EnableAlphaManagementPolicies feature.Flag = "EnableAlphaManagementPolicies"
)