# Add a Managed Resource

Adds a complete new managed resource to this provider: API types, client methods, controller, and unit tests. All patterns follow the template's mental model — read `dev/docs/09-lessons-learned.md` before starting.

## Before you begin

Confirm with the user:
- **Resource name** (e.g. `Repository`, `Policy`)
- **API group** (e.g. `repo`, `iam`) — becomes `apis/<group>/v1alpha1/`
- **External identifier**: what field uniquely identifies this resource in the backend? This becomes the `crossplane.io/external-name` annotation value.
- **Immutable fields**: fields the backend won't let you change after creation → add CEL rule `+kubebuilder:validation:XValidation:rule="self == oldSelf"`
- **CRUD endpoints**: verify against the backend's API spec before writing client code

## Step 1 — API types

Create `apis/<group>/v1alpha1/<resource>_types.go`. Every struct needs `// +kubebuilder:object:generate=true` — this is critical. Without it, `controller-gen` does not generate `DeepCopyInto` for that type, causing a compile error when the CI runs `scripts/generate.sh` before building.

```go
// +kubebuilder:object:generate=true

// <Resource>Parameters are the configurable fields of a <Resource>.
type <Resource>Parameters struct {
    // ExternalName is handled via annotation; omit it here.

    // RequiredField is a required field.
    // +kubebuilder:validation:Required
    RequiredField string `json:"requiredField"`

    // ImmutableField cannot be changed after creation.
    // +kubebuilder:validation:XValidation:rule="self == oldSelf",message="<field> is immutable"
    ImmutableField string `json:"immutableField"`

    // OptionalField is optional.
    // +kubebuilder:validation:Optional
    OptionalField *string `json:"optionalField,omitempty"`

    // SecretRef: credentials are ALWAYS *xpv1.SecretKeySelector, never inline strings.
    // Field name: <Purpose>SecretRef, type *xpv1.SecretKeySelector.
    // +kubebuilder:validation:Optional
    TokenSecretRef *xpv1.SecretKeySelector `json:"tokenSecretRef,omitempty"`
}

// +kubebuilder:object:generate=true

// <Resource>Observation are the observable fields of a <Resource>.
type <Resource>Observation struct {
    ID string `json:"id,omitempty"`
    // ...backend-reported fields only
}

// +kubebuilder:object:generate=true

// <Resource>Spec defines the desired state of a <Resource>.
type <Resource>Spec struct {
    xpv1.ManagedResourceSpec `json:",inline"`
    ForProvider              <Resource>Parameters `json:"forProvider"`
}

// +kubebuilder:object:generate=true

// <Resource>Status represents the observed state of a <Resource>.
type <Resource>Status struct {
    xpv1.ConditionedStatus `json:",inline"`
    AtProvider             <Resource>Observation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,<service>}

// <Resource> is a managed resource representing a <service> <resource>.
type <Resource> struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec   <Resource>Spec   `json:"spec"`
    Status <Resource>Status `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// <Resource>List contains a list of <Resource>.
type <Resource>List struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []<Resource> `json:"items"`
}
```

Also create `apis/<group>/v1alpha1/groupversion_info.go` if the group is new:

```go
package v1alpha1

import (
    "k8s.io/apimachinery/pkg/runtime/schema"
    "sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
    SchemeGroupVersion = schema.GroupVersion{Group: "<group>.<service>.crossplane.io", Version: "v1alpha1"}
    SchemeBuilder      = &scheme.Builder{GroupVersion: SchemeGroupVersion}
    AddToScheme        = SchemeBuilder.AddToScheme
)

var <Resource>GroupVersionKind = SchemeGroupVersion.WithKind("<Resource>")
```

**DO NOT** create `register.go` alongside `groupversion_info.go` — duplicate declarations won't compile.

Add `<Resource>` and `<Resource>List` to the `SchemeBuilder.Register(...)` call in `groupversion_info.go` or a new `AddToScheme` call in `apis/apis.go`.

Then add the hand-written managed methods in `apis/<group>/v1alpha1/<resource>_managed.go` — copy `widget_managed.go` and substitute the type. This is required because `angryjet` does not emit these for the v2 namespaced MR shape (lesson #20).

## Step 2 — Generate

```bash
bash scripts/generate.sh
```

Verify `apis/<group>/v1alpha1/zz_generated.deepcopy.go` contains `func (in *<Resource>Spec) DeepCopyInto(...)` and `func (in *<Resource>Status) DeepCopyInto(...)`. If those are missing, the `+kubebuilder:object:generate=true` markers are absent.

Verify `package/crds/` contains the new CRD YAML.

## Step 3 — Client

Create `internal/clients/admin/<resource>.go` (or add to an existing client file):

```go
// <Resource>Info is the internal representation.
type <Resource>Info struct {
    ID   string
    Name string
    // ...
}

// Get<Resource> returns (nil, nil) on not-found — NEVER return an error for 404.
func (c *Client) Get<Resource>(ctx context.Context, name string) (*<Resource>Info, error) {
    // call the backend
    // if status == 404: return nil, nil
    // if other error: return nil, err
    // else: return &<Resource>Info{...}, nil
}

// Create<Resource> creates the resource and returns the backend-assigned id.
func (c *Client) Create<Resource>(ctx context.Context, params <Resource>Params) (*<Resource>Info, error) { ... }

// Update<Resource> updates mutable fields.
func (c *Client) Update<Resource>(ctx context.Context, name string, params <Resource>Params) error { ... }

// Delete<Resource> is idempotent — return nil on 404.
func (c *Client) Delete<Resource>(ctx context.Context, name string) error {
    // if status == 404: return nil
    // else: return err (or nil on success)
}
```

Key rule: detect not-found by HTTP status code, not by string matching in an error message.

## Step 4 — Controller

Create `internal/controller/<resource>/<resource>.go`:

```go
const (
    errNot<Resource>  = "managed resource is not a <Resource>"
    errTrackPCUsage  = "cannot track ProviderConfig usage"
    errGet<Resource>  = "cannot get <resource>"
    // ...
)

type connector struct {
    kube  client.Client
    usage resource.ModernTracker
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
    cr, ok := mg.(*v1alpha1.<Resource>)
    if !ok { return nil, errors.New(errNot<Resource>) }
    if err := c.usage.Track(ctx, cr); err != nil { // pass cr (typed), not mg
        return nil, errors.Wrap(err, errTrackPCUsage)
    }
    cl, err := clients.NewClientFromProviderConfig(ctx, c.kube, cr)
    return &external{client: cl}, err
}

type external struct{ client *admin.Client }

func (e *external) Disconnect(_ context.Context) error { return nil }

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
    cr := mg.(*v1alpha1.<Resource>)
    name := meta.GetExternalName(cr) // always key off external-name, not atProvider

    obj, err := e.client.Get<Resource>(ctx, name)
    if err != nil { return managed.ExternalObservation{}, errors.Wrap(err, errGet<Resource>) }
    if obj == nil { return managed.ExternalObservation{ResourceExists: false}, nil }

    cr.Status.AtProvider.ID = obj.ID
    // ... populate other observed fields

    upToDate := isUpToDate(cr, obj)
    if upToDate {
        cr.SetConditions(xpv1.Available()) // required in v2 — runtime no longer auto-sets this
    }
    return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
    cr := mg.(*v1alpha1.<Resource>)
    obj, err := e.client.Create<Resource>(ctx, ...)
    if err != nil { return managed.ExternalCreation{}, err }
    meta.SetExternalName(cr, obj.ID) // stamp the backend id
    return managed.ExternalCreation{}, nil // no ExternalNameAssigned field in v2
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
    cr := mg.(*v1alpha1.<Resource>)
    return managed.ExternalUpdate{}, e.client.Update<Resource>(ctx, meta.GetExternalName(cr), ...)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
    // signature is (ExternalDelete, error) in v2 — not just error
    cr := mg.(*v1alpha1.<Resource>)
    return managed.ExternalDelete{}, e.client.Delete<Resource>(ctx, meta.GetExternalName(cr))
}
```

Wire in `internal/controller/controller.go`:

```go
func Setup(mgr ctrl.Manager, o controller.Options) error {
    // ...existing controllers...
    return errors.Join(
        widget.Setup(mgr, o),
        <resource>.Setup(mgr, o),
    )
}
```

## Step 5 — Unit tests

Create `internal/controller/<resource>/<resource>_test.go`. Use `httptest.NewServer` via the client constructor — test the controller's `Observe`/`Create`/`Update`/`Delete` methods against a real HTTP handler, not mocks:

- `Observe` with existing resource → `ResourceExists:true`, `Ready=Available` when up-to-date
- `Observe` with drifted resource → `ResourceExists:true`, `ResourceUpToDate:false`
- `Observe` with missing resource → `ResourceExists:false`
- `Create` → backend called, external-name stamped
- `Delete` → backend called, nil returned on success
- `Delete` with 404 response → nil (idempotent)

See `internal/controller/widget/widget_test.go` for the exact test helper pattern.

## Step 6 — Add example

Create `examples/e2e/<resource>.yaml` with minimal valid `forProvider`. This drives the uptest e2e: apply → wait Ready → delete → assert gone.

## Checklist

- [ ] `+kubebuilder:object:generate=true` on `<Resource>Parameters`, `<Resource>Observation`, `<Resource>Spec`, `<Resource>Status`
- [ ] `zz_generated.deepcopy.go` has `(*<Resource>Spec).DeepCopyInto` and `(*<Resource>Status).DeepCopyInto` after generate
- [ ] `Get<Resource>` returns `(nil, nil)` on 404
- [ ] `Delete<Resource>` returns nil on 404
- [ ] `Create` sets `meta.SetExternalName(cr, id)` and returns `managed.ExternalCreation{}`
- [ ] `Delete` returns `(managed.ExternalDelete{}, err)` — not just `error`
- [ ] `Observe` calls `cr.SetConditions(xpv1.Available())` when up-to-date
- [ ] `connector.Track(ctx, cr)` called with typed `cr`, not `mg`
- [ ] Secret fields use `*xpv1.SecretKeySelector`, not inline strings
- [ ] New group registered in `apis/apis.go`
- [ ] `<resource>_managed.go` hand-written (angryjet does not generate v2 methods)
- [ ] `bash scripts/generate.sh` passes
- [ ] `go build ./...` passes
- [ ] Unit tests pass
