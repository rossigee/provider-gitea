---
name: implement-managed-resource
description: Implements a complete Crossplane managed resource in this provider — API types with correct markers, client CRUD methods, controller (Observe/Create/Update/Delete), and unit tests. Use this when the user asks to add or implement a new resource kind.
tools: Read, Edit, Write, Bash
---

You are implementing a managed resource for a Crossplane provider based on the `mosabastion/crossplane-provider-template` pattern. You have full context of the lessons learned from real provider implementations. Work through the following steps in order, reading existing files before editing them.

## Non-negotiable rules (failures cause CI breaks or runtime errors)

**1. `+kubebuilder:object:generate=true` on every non-root type**
Every `*Parameters`, `*Observation`, `*Spec`, `*Status` struct MUST have `// +kubebuilder:object:generate=true` immediately before it. Without this, `controller-gen` skips generating `DeepCopyInto` for that type. The root type's deepcopy then calls `in.Spec.DeepCopyInto(&out.Spec)` which resolves to the embedded `ManagedResourceSpec.DeepCopyInto(*ManagedResourceSpec)` — a type mismatch that breaks the release CI build which regenerates before compiling.

**2. `Get*` returns `(nil, nil)` on 404 — never an error**
A 404 from the backend means "resource doesn't exist" which is normal state the controller handles via `Observe → ResourceExists: false → Create`. Surfacing it as an error causes spurious reconcile failures.

**3. `Delete*` is idempotent — return nil on 404**
The finalizer must release. If the backend already deleted the resource, that's success.

**4. `Create` stamps external-name from the backend id**
```go
meta.SetExternalName(cr, obj.ID) // NOT a constant or spec field
return managed.ExternalCreation{}, nil // no ExternalNameAssigned field in crossplane-runtime v2
```

**5. `Delete` returns `(managed.ExternalDelete, error)` — not just `error`**
The signature changed in crossplane-runtime v2. Returning just `error` is a compile error.

**6. `Observe` calls `cr.SetConditions(xpv1.Available())` when up-to-date**
crossplane-runtime v2 no longer auto-sets `Available()`. Without this, `Ready` stays stuck on `Creating` forever even though the resource exists and reconciles correctly.

**7. `connector.Track(ctx, cr)` — pass the typed `cr`, not the `mg` interface**
`cr` implements `resource.ModernManaged`; `mg` may not satisfy the ModernTracker's expectations.

**8. Secret fields are always `*xpv1.SecretKeySelector`, never inline strings**
Field name: `<Purpose>SecretRef`, type `*xpv1.SecretKeySelector` from `xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"`. Never redeclare a local SecretKeySelector struct. Resolve in controller via `clients.ResolveSecretValue(ctx, kube, sel)`.

**9. No `register.go` alongside `groupversion_info.go`**
They redeclare the same vars and won't compile. Everything belongs in `groupversion_info.go`.

**10. `<resource>_managed.go` must be hand-written (angryjet doesn't handle v2 namespaced MR)**
Copy `apis/sample/v1alpha1/widget_managed.go` and substitute the type name. All eight methods are required — the runtime type-switches on `LocalConnectionSecretOwner` at runtime; missing methods cause `errManagedNotImplemented` on every reconcile.

## Implementation steps

### Step 1: Read existing code

Before creating anything, read:
- The existing type file in the same API group (if any) to understand naming conventions
- `internal/clients/` to understand the client structure
- `internal/controller/widget/widget.go` for the exact controller pattern
- `apis/sample/v1alpha1/widget_managed.go` for the managed methods template

### Step 2: API types (`apis/<group>/v1alpha1/<resource>_types.go`)

Structure:
```go
// +kubebuilder:object:generate=true
type <Resource>Parameters struct { ... }

// +kubebuilder:object:generate=true
type <Resource>Observation struct { ... }

// +kubebuilder:object:generate=true
type <Resource>Spec struct {
    xpv1.ManagedResourceSpec `json:",inline"`
    ForProvider              <Resource>Parameters `json:"forProvider"`
}

// +kubebuilder:object:generate=true
type <Resource>Status struct {
    xpv1.ConditionedStatus `json:",inline"`
    AtProvider             <Resource>Observation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,<service>}
type <Resource> struct { ... }

// +kubebuilder:object:root=true
type <Resource>List struct { ... }
```

### Step 3: Managed methods (`apis/<group>/v1alpha1/<resource>_managed.go`)

Copy `widget_managed.go` verbatim, replace `Widget`→`<Resource>`. All eight methods required.

### Step 4: Register the new type

In `apis/<group>/v1alpha1/groupversion_info.go` add to `SchemeBuilder.Register(...)`:
```go
SchemeBuilder.Register(&<Resource>{}, &<Resource>List{})
```

If this is a new group, also add `<group>v1alpha1.AddToScheme` to `apis/apis.go`.

### Step 5: Generate

```bash
bash scripts/generate.sh
```

Verify the generated deepcopy has `(*<Resource>Spec).DeepCopyInto` and `(*<Resource>Status).DeepCopyInto`. If missing, the markers in step 2 are wrong.

### Step 6: Client (`internal/clients/admin/<resource>.go`)

```go
func (c *Client) Get<Resource>(ctx context.Context, id string) (*<Resource>Info, error) {
    // 404 → return nil, nil
}
func (c *Client) Create<Resource>(ctx context.Context, p <Resource>Params) (*<Resource>Info, error) { ... }
func (c *Client) Update<Resource>(ctx context.Context, id string, p <Resource>Params) error { ... }
func (c *Client) Delete<Resource>(ctx context.Context, id string) error {
    // 404 → return nil
}
```

### Step 7: Controller (`internal/controller/<resource>/<resource>.go`)

Follow the widget.go pattern exactly. Key points:
- `Observe`: read by `meta.GetExternalName(cr)`, not `cr.Status.AtProvider.ID`
- `Observe`: set `cr.Status.AtProvider` from live state before returning
- `Observe`: call `cr.SetConditions(xpv1.Available())` only when `upToDate == true`
- `Create`: call `meta.SetExternalName(cr, id)`, return `managed.ExternalCreation{}`
- `Delete`: return `managed.ExternalDelete{}, err`

Wire in `internal/controller/controller.go`.

### Step 8: Unit tests (`internal/controller/<resource>/<resource>_test.go`)

Write at minimum:
- `TestObserve_ExistsUpToDate` → `ResourceExists:true`, `ResourceUpToDate:true`, `Ready=Available`
- `TestObserve_ExistsDrifted` → `ResourceExists:true`, `ResourceUpToDate:false`, Ready not Available
- `TestObserve_NotFound` → `ResourceExists:false`
- `TestCreate_SetsExternalName` → external-name stamped from backend id
- `TestDelete_Idempotent` → no error when 404

### Step 9: Example (`examples/e2e/<resource>.yaml`)

Minimal valid MR manifest for uptest.

### Step 10: Verify

```bash
bash scripts/generate.sh
go build ./...
go test -race ./internal/...
```
