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
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSuite provides a simple test orchestration framework
type TestSuite struct {
	fixtures *TestFixtures
	t        *testing.T
}

// NewTestSuite creates a new test suite
func NewTestSuite(t *testing.T) *TestSuite {
	return &TestSuite{
		fixtures: NewTestFixtures(),
		t:        t,
	}
}

// WithFixtures sets custom test fixtures
func (s *TestSuite) WithFixtures(fixtures *TestFixtures) *TestSuite {
	s.fixtures = fixtures
	return s
}

// AssertNoError is a test helper for asserting no error occurred
func (s *TestSuite) AssertNoError(err error) {
	s.t.Helper()
	assert.NoError(s.t, err)
}

// AssertError is a test helper for asserting an error occurred
func (s *TestSuite) AssertError(err error) {
	s.t.Helper()
	assert.Error(s.t, err)
}

// AssertErrorContains is a test helper for asserting an error contains specific text
func (s *TestSuite) AssertErrorContains(err error, text string) {
	s.t.Helper()
	assert.Error(s.t, err)
	assert.Contains(s.t, err.Error(), text)
}