# Controller Implementation Guide

## Status
Implementation on **hold pending v2.3.2 API clarification**. The Crossplane v2.3.2 upgrade removed the `*_managed.go` and `*_managedlist.go` files that previously contained the `Managed` interface implementations.

## Architecture Overview

### Current State (Post-v2.3.2 Upgrade)
- ✅ 22 resource type definitions with `ManagedResourceSpec`/`ManagedResourceStatus` embedded
- ✅ Gitea client library with 60+ methods for all operations
- ✅ v2 API groups properly configured
- ❌ Controllers: 0/22 implemented
- ❓ Managed interface implementations: Need clarification on v2.3.2 patterns

### Known Changes in v2.3.2
1. Consolidated resource interface implementations
2. Changed how managed resource interfaces are exposed
3. Likely impacts on type assertions and method signatures

## Implementation Requirements

Each controller requires:

### 1. **Connector** - Establish external client connection
```go
type connector struct {
    kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
    // Get provider config
    // Create Gitea client
    // Return external client
}
```

### 2. **ExternalClient** - Perform CRUD operations
```go
type externalClient struct {
    client clients.Client
}

// Observe: Check if external resource exists and is up-to-date
func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error)

// Create: Create new external resource
func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error)

// Update: Update external resource if needed
func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error)

// Delete: Delete external resource
func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error)

// Disconnect: Clean up external connections
func (e *externalClient) Disconnect(ctx context.Context) error
```

### 3. **Controller Setup** - Register with reconciler
```go
func Setup(mgr ctrl.Manager, o xpv1.Options) error {
    // Create reconciler with:
    // - Connector
    // - Initializers (API + ExternalName)
    // - Logger
    // - Event recorder
    // - Connection publishers
    
    // Register controller with manager
}
```

## Resource Implementation Priority

### Phase 1: Core Resources (Foundation)
1. **Repository** - Starter controller
2. **Organization** - Foundation for teams
3. **Team** - Used by collaborators
4. **DeployKey** - Simple & independent
5. **User** - Simple & independent
6. **Label** - Lightweight reference
7. **Webhook** - CI/CD integration

### Phase 2: Security Resources
8. **AccessToken** - Authentication
9. **UserKey** - Authentication
10. **RepositoryKey** - Authentication
11. **RepositorySecret** - Secrets
12. **OrganizationSecret** - Secrets
13. **BranchProtection** - Policy
14. **OrganizationMember** - Access

### Phase 3: CI/CD & Admin
15-22. **Action, Runner, AdminUser, OrganizationSettings, GitHook, RepositoryCollaborator, Issue, PullRequest, Release**

## Key Implementation Patterns

### External Name Tracking
```go
// Get external identifier
externalID := meta.GetExternalName(cr)

// Set external identifier after creation
meta.SetExternalName(cr, "owner/repo")
```

### Status Updates
```go
// Update observed state
cr.Status.AtProvider = v2.RepositoryObservation{
    ID:       &repo.ID,
    FullName: &repo.FullName,
    // ... other fields
}

// Set conditions
condition.SetStatus(cr, corev1.ConditionTrue)
```

### Error Handling
```go
// Resource doesn't exist - return ResourceExists: false
if strings.Contains(err.Error(), "404") {
    return managed.ExternalObservation{ResourceExists: false}, nil
}

// Resource exists - return ResourceExists: true, ResourceUpToDate: true/false
return managed.ExternalObservation{
    ResourceExists: true,
    ResourceUpToDate: isUpToDate,
}, nil
```

### Client Methods

All client operations available in `internal/clients/gitea.go`:

**Repository**:
- `GetRepository(owner, name)`
- `CreateRepository() / CreateOrganizationRepository(org)`
- `UpdateRepository(owner, name, req)`
- `DeleteRepository(owner, name)`

**Organization**:
- `GetOrganization(name)`
- `CreateOrganization(req)`
- `UpdateOrganization(name, req)`
- `DeleteOrganization(name)`

**Team**:
- `GetTeam(teamID)`
- `CreateTeam(org, req)`
- `UpdateTeam(teamID, req)`
- `DeleteTeam(teamID)`

*And 50+ more methods for all resource types...*

## Testing Infrastructure

Available in `internal/controller/testing/`:
- `TestSuite` - Orchestration framework
- `TestFixtures` - Pre-built test data
- `MockClient` - Gitea client mock
- Helper utilities for assertions

## Next Steps

1. **Clarify v2.3.2 API Patterns** - Understand new interface implementation approach
2. **Implement Repository Controller** - Foundational proof-of-concept
3. **Create Controller Template** - Standardized pattern for all 22 resources
4. **Implement Phase 1** - 7 core controllers
5. **Iterate Phases** - Complete remaining 15 controllers

## Known Issues

- [ ] v2.3.2 Managed interface implementation pattern
- [ ] Type assertion compatibility with new API
- [ ] ExternalClient method signatures in v2.3.2
- [ ] Generated type safety for resource types

## Estimated Effort

- Repository controller: 2-3 hours (discovery + implementation)
- Each Phase 1 controller: 1-2 hours
- Each Phase 2 controller: 1.5-2 hours
- Each Phase 3 controller: 1-2 hours
- **Total: ~50-70 hours for complete 22-resource implementation**
