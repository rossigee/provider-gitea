# Webhook

**API Version**: `webhook.gitea.m.crossplane.io/v2`

Manages webhooks for repositories and organizations.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.repository` | string | no* | Repository name (for repo webhooks) |
| `forProvider.owner` | string | no* | Repository owner |
| `forProvider.organization` | string | no* | Organization name (for org webhooks) |
| `forProvider.url` | string | yes | Webhook payload URL |
| `forProvider.type` | string | no | Webhook type (default: "gitea") |
| `forProvider.contentType` | string | no | Content type (json, form) |
| `forProvider.secret` | string | no | Webhook secret |
| `forProvider.active` | bool | no | Webhook is active (default: true) |
| `forProvider.events` | []string | no | Trigger events (default: ["push"]) |
| `forProvider.sslVerification` | bool | no | Verify SSL certificates (default: true) |

*Either `repository`+`owner` or `organization` must be specified.

## Example

```yaml
apiVersion: webhook.gitea.m.crossplane.io/v2
kind: Webhook
metadata:
  name: my-webhook
  namespace: production
spec:
  forProvider:
    repository: my-repo
    owner: myorg
    url: https://example.com/webhook
    events:
      - push
      - pull_request
    active: true
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Registers a new webhook
- **Update**: Updates webhook configuration
- **Delete**: Removes the webhook

## Status Fields

- `status.atProvider.id` — Webhook ID
- `status.atProvider.createdAt` — Creation timestamp
- `status.atProvider.updatedAt` — Last update timestamp
