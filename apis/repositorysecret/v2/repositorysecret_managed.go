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

package v2

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// These methods complete crossplane-runtime's resource.Managed interface for
// RepositorySecret: the ProviderConfig + connection-secret accessors. Without
// GetWriteConnectionSecretToReference the runtime's connection publisher fails
// every reconcile with "managed resource does not implement connection
// details" (it type-switches on resource.LocalConnectionSecretOwner).
// crossplane-tools' angryjet does not emit these for the v2 namespaced shape,
// so they are hand-written (mirroring crossplane-provider-template).
// The Conditioned + ManagementPolicies accessors live in the types file.

// GetProviderConfigReference of this RepositorySecret.
func (mg *RepositorySecret) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// SetProviderConfigReference of this RepositorySecret.
func (mg *RepositorySecret) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// GetWriteConnectionSecretToReference of this RepositorySecret.
func (mg *RepositorySecret) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetWriteConnectionSecretToReference of this RepositorySecret.
func (mg *RepositorySecret) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
