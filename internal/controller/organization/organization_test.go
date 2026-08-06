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

package organization

import (
	"testing"

	"github.com/rossigee/provider-gitea/apis/organization/v2"
	"github.com/rossigee/provider-gitea/internal/clients"
	"github.com/stretchr/testify/assert"
)

func TestIsOrganizationUpToDate(t *testing.T) {
	testCases := []struct {
		name   string
		crSpec v2.OrganizationParameters
		org    *clients.Organization
		want   bool
	}{
		{
			name: "up to date",
			crSpec: v2.OrganizationParameters{
				Username:    "test-org",
				Name:        strPtr("Test Org"),
				Description: strPtr("Test description"),
			},
			org: &clients.Organization{
				Username:    "test-org",
				FullName:    "Test Org",
				Description: "Test description",
			},
			want: true,
		},
		{
			name: "needs update - name",
			crSpec: v2.OrganizationParameters{
				Username: "test-org",
				Name:     strPtr("New Name"),
			},
			org: &clients.Organization{
				Username: "test-org",
				FullName: "Old Name",
			},
			want: false,
		},
		{
			name: "needs update - description",
			crSpec: v2.OrganizationParameters{
				Username:    "test-org",
				Description: strPtr("New description"),
			},
			org: &clients.Organization{
				Username:    "test-org",
				Description: "Old description",
			},
			want: false,
		},
		{
			name: "needs update - location",
			crSpec: v2.OrganizationParameters{
				Username: "test-org",
				Location: strPtr("New Location"),
			},
			org: &clients.Organization{
				Username: "test-org",
				Location: "Old Location",
			},
			want: false,
		},
		{
			name: "needs update - website",
			crSpec: v2.OrganizationParameters{
				Username: "test-org",
				Website:  strPtr("https://new-site.com"),
			},
			org: &clients.Organization{
				Username: "test-org",
				Website:  "https://old-site.com",
			},
			want: false,
		},
		{
			name: "needs update - visibility",
			crSpec: v2.OrganizationParameters{
				Username:   "test-org",
				Visibility: strPtr("private"),
			},
			org: &clients.Organization{
				Username:  "test-org",
				Visibility: "public",
			},
			want: false,
		},
		{
			name: "up to date - nil optional fields",
			crSpec: v2.OrganizationParameters{
				Username: "test-org",
			},
			org: &clients.Organization{
				Username: "test-org",
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cr := &v2.Organization{
				Spec: v2.OrganizationSpec{
					ForProvider: tc.crSpec,
				},
			}
			result := isOrganizationUpToDate(cr, tc.org)
			assert.Equal(t, tc.want, result)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
