# Package Registry

Provider-gitea publishes its xpkg package to `ghcr.io/rossigee/provider-gitea`. The repository does not publish to Upbound, Harbor, or Docker Hub.

Install the released package with:

```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-gitea:v0.15.9
```

Publication is performed by the tag-only release workflow after the release PR is merged.
