# Upbound Registry Integration

This document contains instructions for enabling Upbound registry integration once the provider is registered.

## Current Status

- ✅ **GHCR**: All packages and images are published to GitHub Container Registry
- ⏳ **Upbound**: Commented out in workflows, ready to enable

## When Ready to Enable Upbound Registry

### 1. Registry Setup
1. Register the provider at [Upbound Marketplace](https://marketplace.upbound.io/)
2. Get your `XPKG_ACCESS_ID` and `XPKG_TOKEN` credentials
3. Add them as GitHub repository secrets:
   - `XPKG_ACCESS_ID`: Your Upbound access ID
   - `XPKG_TOKEN`: Your Upbound access token

### 2. Uncomment Workflow Steps

In `.github/workflows/ci.yml`, uncomment:
```yaml
- name: Upload Package to Registry
  if: startsWith(github.ref, 'refs/tags/')
  env:
    XPKG_ACCESS_ID: ${{ secrets.XPKG_ACCESS_ID }}
    XPKG_TOKEN: ${{ secrets.XPKG_TOKEN }}
  run: |
    make xpkg.push PACKAGE_REGISTRY=xpkg.upbound.io
```

In `.github/workflows/release.yml`, uncomment:
```yaml
- name: Upload Package to Upbound Registry
  if: "!contains(steps.version.outputs.version, '-')"
  env:
    XPKG_ACCESS_ID: ${{ secrets.XPKG_ACCESS_ID }}
    XPKG_TOKEN: ${{ secrets.XPKG_TOKEN }}
  run: |
    echo "Uploading package to Upbound registry..."
    make xpkg.push PACKAGE_REGISTRY=xpkg.upbound.io VERSION=${{ steps.version.outputs.version }}
```

### 3. Update Documentation

Update installation instructions to include Upbound registry:

**In README.md:**
```bash
# Install from Upbound (recommended)
kubectl crossplane install provider xpkg.upbound.io/crossplane-contrib/provider-gitea:latest

# Or install from GHCR
kubectl crossplane install provider ghcr.io/crossplane-contrib/provider-gitea:latest
```

**In examples/install.yaml:**
```yaml
spec:
  package: xpkg.upbound.io/crossplane-contrib/provider-gitea:latest
```

### 4. Test the Integration

1. Create a test release to verify Upbound upload works
2. Check that the package appears in the Upbound marketplace
3. Test installation from both registries

## Dual Registry Strategy

Once enabled, the provider will be available from both registries:

- **Upbound Registry**: `xpkg.upbound.io/crossplane-contrib/provider-gitea`
  - Primary distribution channel
  - Better discoverability in Crossplane ecosystem
  - Integrated with Crossplane documentation

- **GitHub Container Registry**: `ghcr.io/crossplane-contrib/provider-gitea`
  - Backup distribution channel
  - Direct from source repository
  - No external dependencies

This provides redundancy and flexibility for users.
