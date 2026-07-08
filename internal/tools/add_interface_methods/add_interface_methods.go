package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resource maps resource names to their Go type names
var resources = map[string]string{
	"accesstoken":            "AccessToken",
	"action":                 "Action",
	"adminuser":              "AdminUser",
	"branchprotection":       "BranchProtection",
	"deploykey":              "DeployKey",
	"githook":                "GitHook",
	"issue":                  "Issue",
	"label":                  "Label",
	"organization":           "Organization",
	"organizationmember":     "OrganizationMember",
	"organizationsecret":     "OrganizationSecret",
	"organizationsettings":   "OrganizationSettings",
	"pullrequest":            "PullRequest",
	"release":                "Release",
	"repository":             "Repository",
	"repositorycollaborator": "RepositoryCollaborator",
	"repositorykey":          "RepositoryKey",
	"repositorysecret":       "RepositorySecret",
	"runner":                 "Runner",
	"team":                   "Team",
	"user":                   "User",
	"userkey":                "UserKey",
	"webhook":                "Webhook",
}

const methodsTemplate = `
// GetCondition returns the condition for the given ConditionType if it exists, otherwise returns nil.
func (r *%s) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return r.Status.GetCondition(ct)
}

// SetConditions sets the supplied conditions, replacing any existing conditions of the same type.
func (r *%s) SetConditions(c ...xpv1.Condition) {
	r.Status.SetConditions(c...)
}

// GetManagementPolicies returns the management policies for this resource.
func (r *%s) GetManagementPolicies() xpv1.ManagementPolicies {
	return r.Spec.ManagementPolicies
}

// SetManagementPolicies sets the management policies for this resource.
func (r *%s) SetManagementPolicies(p xpv1.ManagementPolicies) {
	r.Spec.ManagementPolicies = p
}
`

func main() {
	if len(os.Args) > 1 && os.Args[1] == "check" {
		checkAllResources()
		return
	}

	generateAllMethods()
}

func checkAllResources() {
	fmt.Println("Checking which resources need wrapper methods...")
	fmt.Println("")

	for name, typeName := range resources {
		typesPath := filepath.Join("apis", name, "v2", "types.go")
		content, err := os.ReadFile(typesPath)
		if err != nil {
			fmt.Printf("❌ %s: File not found\n", typeName)
			continue
		}

		contentStr := string(content)
		if strings.Contains(contentStr, fmt.Sprintf("func (r *%s) GetCondition", typeName)) {
			fmt.Printf("✅ %s: Methods already exist\n", typeName)
		} else {
			fmt.Printf("⏳ %s: Needs methods\n", typeName)
		}
	}
}

func generateAllMethods() {
	fmt.Println("Generating wrapper methods for resource.Managed interface...")
	fmt.Println("")

	succeeded := 0
	skipped := 0
	failed := 0

	for name, typeName := range resources {
		typesPath := filepath.Join("apis", name, "v2", "types.go")
		content, err := os.ReadFile(typesPath)
		if err != nil {
			fmt.Printf("❌ %s: Cannot read file\n", typeName)
			failed++
			continue
		}

		contentStr := string(content)

		// Check if methods already exist
		if strings.Contains(contentStr, fmt.Sprintf("func (r *%s) GetCondition", typeName)) {
			fmt.Printf("⊘ %s: Methods already exist\n", typeName)
			skipped++
			continue
		}

		// Find the init() function and insert methods before it
		initIdx := strings.Index(contentStr, "\nfunc init() {")
		if initIdx == -1 {
			fmt.Printf("❌ %s: Cannot find init() function\n", typeName)
			failed++
			continue
		}

		methods := fmt.Sprintf(methodsTemplate, typeName, typeName, typeName, typeName)
		newContent := contentStr[:initIdx] + "\n" + methods + "\n" + contentStr[initIdx+1:]

		err = os.WriteFile(typesPath, []byte(newContent), 0644)
		if err != nil {
			fmt.Printf("❌ %s: Cannot write file: %v\n", typeName, err)
			failed++
			continue
		}

		fmt.Printf("✅ %s: Methods added\n", typeName)
		succeeded++
	}

	fmt.Println("")
	fmt.Printf("Results: %d added, %d skipped, %d failed\n", succeeded, skipped, failed)

	if failed == 0 {
		fmt.Println("\n✅ All resources updated successfully!")
	}
}
