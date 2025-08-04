# Enterprise GitOps Example

This example demonstrates how to use the Organization Settings and Git Hooks resources together to implement enterprise-grade GitOps workflows with Gitea.

## Scenario

A software company wants to implement strict GitOps practices across their organization with:

1. **Organization-level governance**: Consistent security and collaboration policies
2. **Repository-level automation**: Automated validation and deployment triggers
3. **Compliance requirements**: Signed commits, controlled access, audit trails

## Resources

### 1. Organization Settings (`org-security-policy.yaml`)

Establishes organization-wide security and collaboration policies:
- Requires signed commits for compliance
- Restricts repository creation permissions
- Enables security features like dependency graphs
- Controls Git hooks usage

### 2. Pre-receive Hook (`validation-hook.yaml`)

Implements automated validation before code is accepted:
- Commit message format validation
- Branch protection rules
- Security scanning triggers
- Code quality checks

### 3. Post-receive Hook (`deployment-hook.yaml`)

Triggers automated workflows after code is accepted:
- CI/CD pipeline initiation
- Notification systems
- Compliance logging
- Artifact generation

## Deployment Order

1. First, apply the organization settings to establish the security baseline
2. Then apply the Git hooks to specific repositories that need automation

```bash
kubectl apply -f org-security-policy.yaml
kubectl apply -f validation-hook.yaml
kubectl apply -f deployment-hook.yaml
```

## Benefits

- **Consistent Security**: Organization-wide policies ensure uniform security standards
- **Automated Compliance**: Git hooks enforce rules without manual intervention
- **Scalable Governance**: Policies apply to all repositories in the organization
- **Audit Trail**: All changes are logged and traceable through Crossplane
- **GitOps Native**: Everything is managed as code through Kubernetes manifests
