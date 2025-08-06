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

package testing

import (
	"github.com/rossigee/provider-gitea/apis/repository/v1alpha1"
)

// Simple parameter builders that avoid complex type issues

// RepositoryParameters creates basic repository parameters 
func (f *TestFixtures) RepositoryParameters() v1alpha1.RepositoryParameters {
	return v1alpha1.RepositoryParameters{
		Name:        f.TestRepo,
		Owner:       &f.TestOrg,
		Description: func() *string { s := "Test repository"; return &s }(),
		Private:     func() *bool { b := true; return &b }(),
	}
}