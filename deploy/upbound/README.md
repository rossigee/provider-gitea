# Provider Package Distribution

This repository publishes provider packages to `ghcr.io/rossigee/provider-gitea`. It does not publish an Upbound or Harbor package.

## Install

```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-gitea:v0.15.9
```

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-gitea
spec:
  package: ghcr.io/rossigee/provider-gitea:v0.15.9
```

## Configure

Create a v2 ProviderConfig and its Secret using the examples in the repository. The provider package contains the native controller binary and generated CRDs.

## Release

Publication is performed by the tag-only workflow after the release PR is merged. The workflow publishes the versioned xpkg and `latest` alias, verifies both Linux architectures and equal digests, and creates the GitHub Release.
