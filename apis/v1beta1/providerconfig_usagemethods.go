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

package v1beta1

// Hand-written because angryjet does not emit the v2 typed-PCU methodset
// (GetProviderConfigReference / SetProviderConfigReference with a
// ProviderConfigReference value, and GetResourceReference /
// SetResourceReference with a TypedReference value).  The embedded
// xpv1.ProviderConfigUsage carries the storage fields; these shims satisfy
// resource.TypedProviderConfigUsage so that resource.NewProviderConfigUsageTracker
// accepts *ProviderConfigUsage as its prototype.

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// GetProviderConfigReference returns the typed ProviderConfig reference stored
// in the usage record. The embedded xpv1.Reference only holds a name, so the
// Kind field of the returned value will be empty when read back from storage;
// it is populated on every write via SetProviderConfigReference.
func (p *ProviderConfigUsage) GetProviderConfigReference() xpv1.ProviderConfigReference {
	return xpv1.ProviderConfigReference{Name: p.ProviderConfigReference.Name}
}

// SetProviderConfigReference stores the typed ProviderConfig reference.
func (p *ProviderConfigUsage) SetProviderConfigReference(ref xpv1.ProviderConfigReference) {
	p.ProviderConfigReference = xpv1.Reference{Name: ref.Name}
}

// GetResourceReference returns the managed resource reference stored in the
// usage record.
func (p *ProviderConfigUsage) GetResourceReference() xpv1.TypedReference {
	return p.ResourceReference
}

// SetResourceReference stores the managed resource reference in the usage record.
func (p *ProviderConfigUsage) SetResourceReference(ref xpv1.TypedReference) {
	p.ResourceReference = ref
}
