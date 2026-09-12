# Release

**API Version**: `release.gitea.m.crossplane.io/v2`

Manages repository releases and version tags.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.repository` | string | yes | Repository name |
| `forProvider.owner` | string | yes | Repository owner |
| `forProvider.tagName` | string | yes | Git tag name |
| `forProvider.name` | string | no | Release name |
| `forProvider.body` | string | no | Release notes |
| `forProvider.draft` | bool | no | Draft release |
| `forProvider.prerelease` | bool | no | Pre-release version |
| `forProvider.targetCommitish` | string | no | Target branch or commit |

## Example

```yaml
apiVersion: release.gitea.m.crossplane.io/v2
kind: Release
metadata:
  name: v1.0.0
  namespace: production
spec:
  forProvider:
    repository: my-repo
    owner: myorg
    tagName: v1.0.0
    name: Release v1.0.0
    body: |
      ## What's New
      - Initial release
      - Bug fixes
    prerelease: false
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates a new release
- **Update**: Updates release notes or metadata
- **Delete**: Deletes the release

## Status Fields

- `status.atProvider.id` — Release ID
- `status.atProvider.url` — Release URL
- `status.atProvider.assetsUrl` — Assets URL
- `status.atProvider.tarballUrl` — Tarball URL
- `status.atProvider.zipballUrl` — Zipball URL
- `status.atProvider.createdAt` — Creation timestamp
- `status.atProvider.publishedAt` — Publish timestamp
