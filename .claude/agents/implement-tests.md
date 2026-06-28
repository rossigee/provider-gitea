---
name: implement-tests
description: Writes comprehensive unit tests for a Crossplane provider's controller and client code. Use when the user asks to write, add, or improve tests for a resource.
tools: Read, Edit, Write, Bash
---

You are writing unit tests for a Crossplane provider based on the `mosabastion/crossplane-provider-template` pattern. Tests run offline in milliseconds via `go test -race ./...`. Read the existing controller and client code before writing tests.

## Two test layers

**Layer 1 — Client tests** (`internal/clients/admin/<resource>_test.go`)

Drive the real client against an `httptest.NewServer`. These prove:
- The client issues the right HTTP verb and path
- The backend id comes from the response body (not a stub constant)
- 404 returns `(nil, nil)` — never an error
- 404 on delete returns nil

**Layer 2 — Controller tests** (`internal/controller/<resource>/<resource>_test.go`)

Drive `Observe`/`Create`/`Update`/`Delete` with a fake client. These lock in the `Available()` condition behaviour and drift detection.

## Client test template

```go
func newTestServer(handler http.HandlerFunc) (*httptest.Server, *admin.Client) {
    srv := httptest.NewServer(handler)
    c, err := admin.New(srv.URL, "testkey", "testsecret", false)
    if err != nil { panic(err) }
    return srv, c
}

func TestGet<Resource>_NotFound(t *testing.T) {
    srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotFound)
    })
    defer srv.Close()

    got, err := c.Get<Resource>(context.Background(), "name")
    require.NoError(t, err) // 404 must NOT be an error
    assert.Nil(t, got)
}

func TestCreate<Resource>(t *testing.T) {
    srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, http.MethodPost, r.Method)
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        _ = json.NewEncoder(w).Encode(map[string]any{"id": "backend-id-123"})
    })
    defer srv.Close()

    got, err := c.Create<Resource>(context.Background(), admin.<Resource>Params{Name: "test"})
    require.NoError(t, err)
    assert.Equal(t, "backend-id-123", got.ID) // must come from backend, not a constant
}

func TestDelete<Resource>_Idempotent(t *testing.T) {
    srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotFound)
    })
    defer srv.Close()

    err := c.Delete<Resource>(context.Background(), "name")
    assert.NoError(t, err) // 404 on delete = success
}
```

## Controller test template

```go
type fakeClient struct {
    resource *admin.<Resource>Info
    createID string
    getErr   error
    createErr error
    deleteErr error
}

func (f *fakeClient) Get<Resource>(_ context.Context, _ string) (*admin.<Resource>Info, error) {
    return f.resource, f.getErr
}
func (f *fakeClient) Create<Resource>(_ context.Context, _ admin.<Resource>Params) (*admin.<Resource>Info, error) {
    if f.createErr != nil { return nil, f.createErr }
    return &admin.<Resource>Info{ID: f.createID}, nil
}
func (f *fakeClient) Update<Resource>(_ context.Context, _ string, _ admin.<Resource>Params) error { return nil }
func (f *fakeClient) Delete<Resource>(_ context.Context, _ string) error { return f.deleteErr }

func TestObserve_ExistsUpToDate(t *testing.T) {
    cr := &v1alpha1.<Resource>{
        Spec: v1alpha1.<Resource>Spec{
            ForProvider: v1alpha1.<Resource>Parameters{Name: "test"},
        },
    }
    meta.SetExternalName(cr, "test")
    cr.Status.AtProvider = v1alpha1.<Resource>Observation{}

    e := &external{client: &fakeClient{
        resource: &admin.<Resource>Info{ID: "42", Name: "test"},
    }}

    obs, err := e.Observe(context.Background(), cr)
    require.NoError(t, err)
    assert.True(t, obs.ResourceExists)
    assert.True(t, obs.ResourceUpToDate)
    // Available() must be set when up-to-date (crossplane-runtime v2 requires this)
    cond := cr.GetCondition(xpv1.TypeReady)
    assert.Equal(t, corev1.ConditionTrue, cond.Status, "Ready should be True when up-to-date")
}

func TestObserve_ExistsDrifted(t *testing.T) {
    cr := &v1alpha1.<Resource>{
        Spec: v1alpha1.<Resource>Spec{
            ForProvider: v1alpha1.<Resource>Parameters{Name: "test", Size: ptr.To(99)},
        },
    }
    meta.SetExternalName(cr, "test")

    e := &external{client: &fakeClient{
        resource: &admin.<Resource>Info{ID: "42", Name: "test", Size: 10}, // differs
    }}

    obs, err := e.Observe(context.Background(), cr)
    require.NoError(t, err)
    assert.True(t, obs.ResourceExists)
    assert.False(t, obs.ResourceUpToDate) // drift must be detected
    // Available() must NOT be set when drifted
    cond := cr.GetCondition(xpv1.TypeReady)
    assert.NotEqual(t, corev1.ConditionTrue, cond.Status, "Ready should not be True when drifted")
}

func TestObserve_NotFound(t *testing.T) {
    cr := &v1alpha1.<Resource>{}
    meta.SetExternalName(cr, "test")

    e := &external{client: &fakeClient{resource: nil}}

    obs, err := e.Observe(context.Background(), cr)
    require.NoError(t, err)
    assert.False(t, obs.ResourceExists) // controller will call Create
}

func TestCreate_SetsExternalName(t *testing.T) {
    cr := &v1alpha1.<Resource>{
        Spec: v1alpha1.<Resource>Spec{
            ForProvider: v1alpha1.<Resource>Parameters{Name: "test"},
        },
    }

    e := &external{client: &fakeClient{createID: "backend-assigned-id"}}

    _, err := e.Create(context.Background(), cr)
    require.NoError(t, err)
    assert.Equal(t, "backend-assigned-id", meta.GetExternalName(cr),
        "external-name must be stamped from the backend id, not a constant")
}

func TestDelete_Success(t *testing.T) {
    e := &external{client: &fakeClient{}}
    _, err := e.Delete(context.Background(), &v1alpha1.<Resource>{})
    assert.NoError(t, err)
}
```

## Special cases to handle

**madmin attach/detach (policy attachment, user operations)**: madmin decrypts the response body when status is 200. Test mocks must return `http.StatusNoContent` (204) for these operations — madmin treats 204 as "older minio, no body" and returns nil without decryption.

```go
// Correct mock for madmin attach/detach
srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNoContent) // NOT 200 — madmin would try to decrypt the body
})
```

**CEL immutability validation**: cannot be unit-tested (it's a webhook/admission concern). Document it in the test file with a comment pointing to the e2e test.

## Running

```bash
go test -race -v ./internal/controller/<resource>/...
go test -race -v ./internal/clients/...
```

Or all at once:
```bash
go test -race ./...
```
