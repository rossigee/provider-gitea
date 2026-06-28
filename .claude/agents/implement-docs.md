---
name: implement-docs
description: Writes documentation for a new or updated managed resource in this Crossplane provider — API field descriptions, kubebuilder markers, example YAML, and user-facing README sections. Use when the user asks to add or improve docs for a resource.
tools: Read, Edit, Write, Bash
---

You are writing documentation for a Crossplane provider built on `mosabastion/crossplane-provider-template`. Read the actual types file and any existing examples before writing anything.

## What to document

### 1. API field descriptions (inline in `_types.go`)

Every exported field in `*Parameters` and `*Observation` needs a godoc comment. Follow the kubebuilder convention: one sentence, starts with the field name, ends with a period. Use `+kubebuilder:validation:` markers for constraints.

```go
// +kubebuilder:object:generate=true
type WidgetParameters struct {
    // Name is the name of the widget in the backend. Immutable after creation.
    // +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Name is immutable"
    Name string `json:"name"`

    // Size is the maximum capacity in GiB.
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=1024
    // +optional
    Size *int64 `json:"size,omitempty"`

    // PasswordSecretRef references a Secret key containing the widget password.
    // +optional
    PasswordSecretRef *xpv1.SecretKeySelector `json:"passwordSecretRef,omitempty"`
}
```

Mark immutable fields with both the godoc "Immutable after creation." sentence and the CEL XValidation rule.

Mark secret-bearing fields with a `// +optional` (they're always pointers) and clarify what the key should contain.

### 2. kubebuilder status markers on the root type

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,<service>}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations['crossplane\\.io/external-name']"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
type Widget struct { ... }
```

After editing types, run `bash scripts/generate.sh` to regenerate CRDs. The generated CRD YAML will include your validation and print columns.

### 3. Example manifests

Create `examples/e2e/<resource>.yaml` — the uptest manifest. Must:
- Include all required fields with realistic values
- Reference a `ProviderConfig` named `default`
- Have a stable `metadata.name` (uptest identifies the object by name across CRUD phases)
- Annotate mutable fields with `uptest.upbound.io/update-parameter` so uptest exercises Update:

```yaml
apiVersion: <group>.<provider>.crossplane.io/v1alpha1
kind: Widget
metadata:
  name: test-widget
  annotations:
    crossplane.io/external-name: test-widget
    uptest.upbound.io/update-parameter: '{"spec":{"forProvider":{"size":20}}}'
spec:
  forProvider:
    name: test-widget
    size: 10
  providerConfigRef:
    name: default
```

Also create `examples/<resource>.yaml` — a human-readable "getting started" example with comments explaining each field.

### 4. README section

In `README.md`, add a table entry for the new resource under "Managed Resources":

```markdown
| `Widget` | `sample.provider.io/v1alpha1` | A widget in the backend service. |
```

If the README has an "Examples" section, add a minimal usage snippet.

### 5. CRD description

The `// Widget manages a widget resource in the service.` comment on the root type becomes the CRD `.spec.versions[].schema.openAPIV3Schema.description`. Write it as a complete sentence — what the resource manages, not how it's implemented.

## Verification

```bash
bash scripts/generate.sh
# Check that CRDs have descriptions:
grep -A5 "description:" package/crds/*.yaml | head -40
# Validate examples are valid YAML:
find examples/ -name "*.yaml" | xargs -n1 kubectl apply --dry-run=client -f
```
