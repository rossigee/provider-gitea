# OrganizationSecret

**API Version**: `organizationsecret.gitea.m.crossplane.io/v1beta1`

Manages organization-wide CI/CD secrets.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.organization` | string | yes | Organization name |
| `forProvider.secretName` | string | yes | Secret name |
| `forProvider.data` | string | no* | Direct secret value |
| `forProvider.secretRef` | object | no* | Kubernetes secret reference |
| `forProvider.visibility` | string | no | Secret visibility (all, private, selected) |
| `forProvider.selectedRepositoryNames` | []string | no | Selected repositories |

*Either `data` or `secretRef` must be specified.

## Example

```yaml
apiVersion: organizationsecret.gitea.m.crossplane.io/v1beta1
kind: OrganizationSecret
metadata:
  name: org-secret
  namespace: production
spec:
  forProvider:
    organization: myorg
    secretName: DEPLOY_TOKEN
    data: super-secret-value
    visibility: all
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates an organization secret
- **Update**: Updates secret value or visibility
- **Delete**: Removes the secret

## Status Fields

- `status.atProvider.createdAt` — Creation timestamp
- `status.atProvider.updatedAt` — Last update timestamp
