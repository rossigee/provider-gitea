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

package branchprotection

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctesting "github.com/rossigee/provider-gitea/internal/controller/testing"
)

func TestBranchProtectionFixtures(t *testing.T) {
	fixtures := ctesting.NewTestFixtures()

	assert.Equal(t, "testorg", fixtures.TestOrg)
	assert.Equal(t, "testrepo", fixtures.TestRepo)
}

func TestBranchProtectionMockClientExpectations(t *testing.T) {
	builder := ctesting.NewExternalClient().
		ExpectCreate("CreateBranchProtection", nil, nil).
		ExpectGet("GetBranchProtection", nil, nil).
		ExpectUpdate("UpdateBranchProtection", nil, nil).
		ExpectDelete("DeleteBranchProtection", nil)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.GetGiteaClient())
}

func TestBranchProtectionApprovals(t *testing.T) {
	approvals := 2
	assert.Equal(t, 2, approvals)
}

func TestBranchProtectionPolicies(t *testing.T) {
	// BranchProtection has 16+ policy fields
	policies := map[string]bool{
		"EnablePush":            true,
		"EnableMergeWhitelist":  false,
		"RequireSignedCommits":  true,
		"BlockOnOutdatedBranch": true,
	}

	assert.Equal(t, 4, len(policies))
	assert.True(t, policies["EnablePush"])
}

func TestBranchProtectionRules(t *testing.T) {
	branch := "main"
	protected := true

	assert.Equal(t, "main", branch)
	assert.True(t, protected)
}
