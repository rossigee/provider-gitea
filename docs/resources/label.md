# Label

**API Version**: `label.gitea.m.crossplane.io/v2`

Manages issue and pull request labels.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Label name |
| `forProvider.repository` | string | yes | Repository name |
| `forProvider.owner` | string | yes | Repository owner |
| `forProvider.color` | string | yes | Label color (hex format) |
| `forProvider.description` | string | no | Label description |
| `forProvider.exclusive` | bool | no | Mutually exclusive label |

## Example

```yaml
apiVersion: label.gitea.m.crossplane.io/v2
kind: Label
metadata:
  name: bug-label
  namespace: production
spec:
  forProvider:
    name: bug
    repository: my-repo
    owner: myorg
    color: "d73a49"
    description: Bug reports
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates a new label
- **Update**: Updates label properties
- **Delete**: Removes the label

## Status Fields

- `status.atProvider.id` — Label ID
- `status.atProvider.url` — Label URL
