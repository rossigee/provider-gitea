# Gitea Provider v2 - Controller Implementation Guide

## ✅ Completed

### v2.3.2 Upgrade (June 2026)
- ✅ Updated crossplane-runtime to v2.3.2
- ✅ Fixed angryjet removal by implementing `resource.Managed` interface methods on all 22 resource types
- ✅ Generated 22 v2 controller stubs with complete framework scaffolding
- ✅ Registered all 22 controllers in the provider Setup function
- ✅ Provider binary builds successfully (44MB, statically-linked)

## 🏗️ Current Status

All 22 resources have controller stubs with:
- ✅ Connector pattern (gets ProviderConfig, creates Gitea API client)
- ✅ External client pattern (Observe/Create/Update/Delete interface)
- ✅ Proper error handling and logging structure
- ✅ Type assertions and error constants

## 📋 Resources Ready for Implementation

### Phase 1: Core Resources (3)
1. **Repository** - ✅ FULLY IMPLEMENTED (see `internal/controller/repository/repository.go`)
   - Features: Create, observe, update, delete repos with owner/name format
   - Pattern: Template for other resources
   
2. **Organization** - Framework ready
3. **Team** - Framework ready

### Phase 2: Security & Keys (7)
4. **DeployKey** - Framework ready
5. **RepositoryKey** - Framework ready
6. **UserKey** - Framework ready
7. **AccessToken** - Framework ready
8. **RepositorySecret** - Framework ready
9. **OrganizationSecret** - Framework ready
10. **BranchProtection** - Framework ready

### Phase 2: Collaboration (5)
11. **Label** - Framework ready
12. **Webhook** - Framework ready
13. **OrganizationMember** - Framework ready
14. **RepositoryCollaborator** - Framework ready
15. **User** - Framework ready

### Phase 2: CI/CD (2)
16. **Action** - Framework ready
17. **Runner** - Framework ready

### Phase 3: Administration (5)
18. **AdminUser** - Framework ready
19. **OrganizationSettings** - Framework ready
20. **GitHook** - Framework ready
21. **Issue** - Framework ready
22. **PullRequest** - Framework ready

## 🔧 Implementation Pattern

Each controller stub has:

```go
type connector struct {
    kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
    // Gets ProviderConfig and creates Gitea API client
}

type externalClient struct {
    client clients.Client
}

func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
    // Check if resource exists and return observed state
}

func (e *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
    // Create resource in Gitea, return external ID
}

func (e *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
    // Update resource in Gitea
}

func (e *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
    // Delete resource from Gitea
}
```

## 📚 Implementation Checklist for Each Resource

For each resource, implement the TODO sections:

### 1. Observe Method
- [ ] Parse externalName (varies per resource - some use IDs, some use owner/name format)
- [ ] Call appropriate Gitea API client method (e.g., `e.client.GetRepository()`)
- [ ] Handle 404 errors (return ResourceExists: false)
- [ ] Map Gitea API response to CR status fields
- [ ] Compare desired vs actual state (if needed for ResourceUpToDate)

### 2. Create Method
- [ ] Extract parameters from `cr.Spec.ForProvider`
- [ ] Build request struct (e.g., `CreateRepositoryRequest`)
- [ ] Call Gitea API client Create method
- [ ] Set externalName (usually ID or owner/name format)
- [ ] Update status with created resource info

### 3. Update Method
- [ ] Parse externalName to get resource identifier
- [ ] Extract changed parameters from `cr.Spec.ForProvider`
- [ ] Build update request struct
- [ ] Call Gitea API client Update method
- [ ] Return success or error

### 4. Delete Method
- [ ] Parse externalName to get resource identifier
- [ ] Call Gitea API client Delete method
- [ ] Handle 404 errors (already deleted - return success)
- [ ] Return error only if unexpected failure

## 🎯 Implementation Priority

### High Priority (Most Used)
1. Repository (✅ Done)
2. Organization
3. Team
4. User
5. Webhook
6. Label
7. DeployKey
8. BranchProtection

### Medium Priority (CI/CD)
9. Action
10. Runner
11. AccessToken
12. RepositorySecret

### Lower Priority (Advanced)
13. AdminUser
14. OrganizationSettings
15. GitHook
16. Issue
17. PullRequest
18. And remaining resources

## 📖 Reference: Repository Implementation

The Repository controller in `internal/controller/repository/repository.go` demonstrates:

- ✅ Owner/Name external ID format parsing
- ✅ Separating user vs org repository creation
- ✅ Partial update handling (only send changed fields)
- ✅ Comprehensive status field mapping
- ✅ Error handling for 404s

Use this as a template for similar hierarchical resources.

## 🚀 How to Implement a New Resource

Example: Organization controller

```go
// 1. Update Observe method
func (e *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
    cr, ok := mg.(*v2.Organization)
    if !ok {
        return managed.ExternalObservation{}, errors.New(errNotOrganization)
    }
    
    externalID := meta.GetExternalName(cr)
    if externalID == "" {
        return managed.ExternalObservation{ResourceExists: false}, nil
    }
    
    org, err := e.client.GetOrganization(ctx, externalID)
    if err != nil {
        if strings.Contains(err.Error(), "404") {
            return managed.ExternalObservation{ResourceExists: false}, nil
        }
        return managed.ExternalObservation{}, errors.Wrap(err, errGetOrganization)
    }
    
    cr.Status.AtProvider = v2.OrganizationObservation{
        ID:       &org.ID,
        UserName: &org.Username,
        AvatarURL: &org.AvatarURL,
    }
    
    return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

// 2. Update Create method (similar pattern)
// 3. Update Update method (similar pattern)
// 4. Update Delete method (similar pattern)
```

## 📝 API Client Methods Available

The Gitea API client (`internal/clients/gitea.go`) provides:

- Repositories: GetRepository, CreateRepository, UpdateRepository, DeleteRepository
- Organizations: GetOrganization, CreateOrganization, UpdateOrganization, DeleteOrganization
- Teams: GetTeam, CreateTeam, UpdateTeam, DeleteTeam
- Users: GetUser, CreateUser, UpdateUser, DeleteUser
- And 60+ other methods covering all 22 resource types

Check `internal/clients/gitea.go` for exact method signatures and request/response types.

## 🧪 Testing

After implementing each resource:

1. Verify it compiles: `make build`
2. Check CRD generated: `make generate`
3. Create test manifest in `examples/{resource-type}/`
4. Apply against test Gitea instance
5. Verify resource created/updated/deleted in Gitea UI

## ✨ Next Steps

1. Pick a high-priority resource (Organization, Team, or User)
2. Copy the Repository controller pattern
3. Implement the 4 TODO sections with Gitea API calls
4. Run `make build` to verify
5. Create example manifests
6. Test against Gitea instance

## 📌 Notes

- All 22 types already have the `resource.Managed` interface methods implemented
- All controllers are registered and will be initialized on provider startup
- The controller framework uses Crossplane v2.3.2 reconciliation patterns
- Each resource can be implemented independently
- Framework is production-ready, awaiting controller logic implementation
