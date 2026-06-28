# Write Unit Tests

Writes unit tests for controller and client code following this template's testing patterns. See `dev/docs/07-testing.md` for the full philosophy.

## What to test

Unit tests live in `internal/controller/<resource>/` and `internal/clients/`. They run in milliseconds offline via `go test -race ./...`.

The two highest-value tests (per `dev/docs/09-lessons-learned.md` #2 and #3):

1. **Client test** — drives the real client against an `httptest` server. Proves the client issues the right verbs/paths, parses the backend id (not a stub constant), and returns `(nil, nil)` on 404.

2. **Controller test** — drives `Observe`/`Create`/`Update`/`Delete` with a mock client. Locks in the `Available()` condition behaviour (lesson #2) and that drift is detected correctly (lesson #3).

## Client test pattern

```go
func TestGet<Resource>_NotFound(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/name") {
            w.WriteHeader(http.StatusNotFound)
            return
        }
        t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
    }))
    defer srv.Close()

    client, err := admin.New(srv.URL, "access", "secret", false)
    require.NoError(t, err)

    got, err := client.Get<Resource>(context.Background(), "name")
    require.NoError(t, err) // 404 must NOT surface as an error
    assert.Nil(t, got)
}

func TestCreate<Resource>(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, http.MethodPost, r.Method)
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]string{"id": "backend-assigned-id"})
    }))
    defer srv.Close()

    client, _ := admin.New(srv.URL, "access", "secret", false)
    got, err := client.Create<Resource>(context.Background(), admin.<Resource>Params{Name: "test"})
    require.NoError(t, err)
    assert.Equal(t, "backend-assigned-id", got.ID) // id must come from backend, not a stub
}

func TestDelete<Resource>_Idempotent(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotFound) // already gone
    }))
    defer srv.Close()

    client, _ := admin.New(srv.URL, "access", "secret", false)
    err := client.Delete<Resource>(context.Background(), "name")
    assert.NoError(t, err) // 404 on delete must be nil, not an error
}
```

## Controller test pattern

Use a lightweight fake client struct; do not mock the HTTP transport at this layer:

```go
type fakeClient struct {
    resource *admin.<Resource>Info
    err      error
}
func (f *fakeClient) Get<Resource>(_ context.Context, _ string) (*admin.<Resource>Info, error) {
    return f.resource, f.err
}
// ...implement other methods

func TestObserve_Exists_UpToDate(t *testing.T) {
    cr := &v1alpha1.<Resource>{
        Spec: v1alpha1.<Resource>Spec{
            ForProvider: v1alpha1.<Resource>Parameters{Name: "test", Size: ptr.To(10)},
        },
    }
    meta.SetExternalName(cr, "test")

    e := &external{client: &fakeClient{
        resource: &admin.<Resource>Info{ID: "42", Name: "test", Size: 10},
    }}

    obs, err := e.Observe(context.Background(), cr)
    require.NoError(t, err)
    assert.True(t, obs.ResourceExists)
    assert.True(t, obs.ResourceUpToDate)
    // Available() must be set when up-to-date (lesson #2)
    assert.Equal(t, corev1.ConditionTrue, cr.GetCondition(xpv1.TypeReady).Status)
}

func TestObserve_Exists_Drifted(t *testing.T) {
    cr := &v1alpha1.<Resource>{
        Spec: v1alpha1.<Resource>Spec{
            ForProvider: v1alpha1.<Resource>Parameters{Name: "test", Size: ptr.To(99)}, // differs
        },
    }
    meta.SetExternalName(cr, "test")

    e := &external{client: &fakeClient{
        resource: &admin.<Resource>Info{ID: "42", Name: "test", Size: 10},
    }}

    obs, err := e.Observe(context.Background(), cr)
    require.NoError(t, err)
    assert.True(t, obs.ResourceExists)
    assert.False(t, obs.ResourceUpToDate) // drift detected
    // Available() must NOT be set when drifted
    assert.NotEqual(t, corev1.ConditionTrue, cr.GetCondition(xpv1.TypeReady).Status)
}

func TestObserve_NotFound(t *testing.T) {
    cr := &v1alpha1.<Resource>{}
    meta.SetExternalName(cr, "test")

    e := &external{client: &fakeClient{resource: nil}}

    obs, err := e.Observe(context.Background(), cr)
    require.NoError(t, err)
    assert.False(t, obs.ResourceExists) // -> Create will be called
}

func TestCreate_SetsExternalName(t *testing.T) {
    called := false
    e := &external{client: &fakeClient{...}}
    // configure fakeClient.Create to return id "backend-id"

    cr := &v1alpha1.<Resource>{...}
    _, err := e.Create(context.Background(), cr)
    require.NoError(t, err)
    assert.Equal(t, "backend-id", meta.GetExternalName(cr)) // must be stamped from backend
    assert.True(t, called)
}

func TestDelete_Idempotent(t *testing.T) {
    // fakeClient.Delete returns nil even when resource is gone
    e := &external{client: &fakeClient{}}
    _, err := e.Delete(context.Background(), &v1alpha1.<Resource>{})
    assert.NoError(t, err) // (ExternalDelete{}, nil) — not just nil
}
```

## Special cases

**madmin attach/detach ops**: madmin's `AttachPolicy`/`DetachPolicy` tries to decrypt a 200 response body. Return 204 (`http.StatusNoContent`) from mock servers for these ops — madmin treats 204 as "older minio, no body" and returns nil without attempting decryption.

**Encrypted admin endpoints** (madmin): the real client uses `EncryptData`/`DecryptData` internally. Test mocks just need to respond with the correct status code; the body can be empty for most ops.

## Running tests

```bash
go test -race ./internal/...
```

Or for a single package:
```bash
go test -race -v ./internal/controller/<resource>/...
```
