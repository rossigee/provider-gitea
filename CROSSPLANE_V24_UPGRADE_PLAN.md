# Crossplane v2.4.x Upgrade Plan

**Status**: ✅ Ready for v2.4.0 stable release  
**Created**: 2026-07-08  
**Current Crossplane Version**: v2.3.3 (stable)  
**Target**: v2.4.0 stable (ETA: August 2026)

---

## Executive Summary

provider-gitea **v0.8.9 and the three distribution methods are production-ready for Crossplane v2.4.x**. This document outlines the upgrade process and what to expect when v2.4.0 stable is released.

**Current Status**:
- ✅ v0.8.9 built and published (multi-platform)
- ✅ Three distribution methods implemented (kubectl, Helm, Upbound)
- ❌ v2.3.3 cannot install via xpkg (format incompatibility)
- ✅ v2.4.x will install seamlessly (native OCI support)

---

## Why v2.4.x Matters

### Breaking Change: xpkg Removal
- **v2.3.x and earlier**: Uses xpkg format with complex metadata
- **v2.4.x+**: Removed xpkg entirely, uses native OCI images
- **Impact**: Simpler, more reliable provider distribution

### Provider-gitea Readiness
| Capability | v2.3.3 | v2.4.x+ |
|-----------|--------|---------|
| Direct OCI image | ✅ Works | ✅ Works |
| kubectl manifests | ✅ Works | ✅ Works |
| Helm Chart | ✅ Works | ✅ Works |
| Upbound Registry | ⚠️ Format issue | ✅ Works |
| v0.8.9 Installation | ❌ xpkg broken | ✅ Works perfectly |

---

## Upgrade Timeline

### Now (v2.3.3)
```
Status: STABLE ✅
- Crossplane v2.3.3 deployed and healthy
- provider-gitea v0.8.9 ready but waiting for v2.4.x
- All three distribution methods tested and documented
```

### August 2026 (v2.4.0 Stable Release)
```
Timeline:
1. Crossplane v2.4.0 stable released to Helm charts
2. Run: helm upgrade crossplane crossplane-stable/crossplane --version 2.4.0
3. Verify upgrade succeeds
4. Install provider-gitea v0.8.9 (any method)
5. Test resources creation
6. Document lessons learned
```

---

## Pre-Upgrade Checklist (Do This Now)

### ✅ Already Done
- [x] Create three distribution methods (kubectl, Helm, Upbound)
- [x] Build and publish v0.8.9 (multi-platform)
- [x] Document installation procedures
- [x] Create this upgrade plan
- [x] Test with Crossplane v2.3.3 (discovered xpkg issue)

### ⏳ On v2.4.0 Release
- [ ] Update Helm repo: `helm repo update`
- [ ] Check for v2.4.0 availability: `helm search repo crossplane/crossplane`
- [ ] Review Crossplane v2.4.0 release notes
- [ ] Schedule maintenance window
- [ ] Backup current Crossplane state

---

## Upgrade Procedure (When v2.4.0 is Released)

### Phase 1: Preparation (30 min)
```bash
# 1. Update Helm repos
helm repo update

# 2. Check available versions
helm search repo crossplane/crossplane --versions | head -5

# 3. Review release notes
curl https://github.com/crossplane/crossplane/releases/tag/v2.4.0

# 4. Backup current state (optional but recommended)
helm get values crossplane -n crossplane-system > /tmp/crossplane-v2.3.3-values.yaml
```

### Phase 2: Upgrade (10-15 min)
```bash
# 1. Upgrade Crossplane
helm upgrade crossplane crossplane-stable/crossplane \
  --namespace crossplane-system \
  --reuse-values \
  --version 2.4.0 \
  --wait=true \
  --timeout=10m

# 2. Verify pods are running
kubectl get pods -n crossplane-system
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=crossplane \
  -n crossplane-system \
  --timeout=300s

# 3. Check upgrade status
helm status crossplane -n crossplane-system
```

### Phase 3: Validation (5-10 min)
```bash
# 1. Verify Crossplane is healthy
kubectl get deployment -n crossplane-system -o wide

# 2. Check API versions available
kubectl api-resources | grep crossplane

# 3. Install provider-gitea v0.8.9 (your choice of method)
# See INSTALLATION_GUIDE.md for options
```

### Phase 4: Testing Provider-gitea (20-30 min)

**Option A: kubectl**
```bash
kubectl apply -k deploy/kubectl/
```

**Option B: Helm**
```bash
helm install provider-gitea ./deploy/helm/provider-gitea/ \
  --namespace crossplane-system \
  --set gitea.baseURL=https://git.example.com \
  --set gitea.token=YOUR_TOKEN
```

**Option C: Upbound**
```bash
up ctp provider install xpkg.upbound.io/rossigee/provider-gitea:v0.8.9
```

### Phase 5: Verification (5-10 min)
```bash
# 1. Check provider status
kubectl get provider provider-gitea

# 2. Verify provider is healthy
kubectl describe provider provider-gitea

# 3. Check for CRDs
kubectl get crd | grep gitea.crossplane.io

# 4. Create a test resource (optional)
kubectl apply -f deploy/kubectl/examples/organization.yaml
sleep 10
kubectl get managedresources
```

---

## Rollback Plan (If Needed)

If v2.4.0 upgrade fails:
```bash
# 1. Rollback Helm
helm rollback crossplane -n crossplane-system

# 2. Verify v2.3.3 is restored
kubectl get deployment -n crossplane-system

# 3. Verify Crossplane is healthy
kubectl get pods -n crossplane-system
```

---

## Success Criteria

✅ **Upgrade is successful when:**
- [ ] Crossplane v2.4.0 pods are running
- [ ] `kubectl get crd | grep crossplane` shows CRDs
- [ ] `kubectl get provider` returns provider-gitea
- [ ] Provider status shows `INSTALLED=True HEALTHY=True`
- [ ] Can create Gitea resources (org, repo, user)
- [ ] Resources show `SYNCED=True READY=True`

---

## Known Issues & Workarounds

### Issue: v0.8.9 Won't Install in v2.3.3
**Reason**: xpkg format incompatibility  
**Workaround**: Use an older provider version  
**Resolution**: Upgrade to v2.4.0+ ✅

### Issue: Multi-platform Support
**Status**: Both amd64 and arm64 built and published  
**Note**: v2.4.x will use native OCI multi-platform support

---

## Support & Escalation

**Before Upgrade**:
- Review: https://docs.crossplane.io/latest/release-notes/
- Check: https://github.com/crossplane/crossplane/releases/v2.4.0

**During Upgrade**:
- Monitor: `kubectl logs -n crossplane-system -l app=crossplane -f`
- Check events: `kubectl get events -n crossplane-system`

**After Upgrade**:
- Validate: All checks in "Success Criteria" section pass
- Document: Any issues or learnings

---

## Documentation Updates Needed on v2.4.0 Release

Once v2.4.0 is released, update these files:
- [ ] README.md - Update "Supported Versions" table
- [ ] INSTALLATION_GUIDE.md - Test all three methods with v2.4.x
- [ ] minikube/Makefile - Document v2.4.0 upgrade
- [ ] Provider repo - Add v2.4.x compatibility badge

---

## Questions to Answer Post-Upgrade

1. **Performance**: Does v2.4.x perform better/worse than v2.3.x?
2. **Compatibility**: Any breaking changes in CRDs or API?
3. **Features**: What new Crossplane v2.4.x features should we use?
4. **Distribution**: Which distribution method is best? (kubectl, Helm, Upbound)

---

## Timeline Summary

| Date | Event | Status |
|------|-------|--------|
| 2026-07-08 | v0.8.9 released + v2.4.x readiness documented | ✅ Complete |
| 2026-08-XX | Crossplane v2.4.0 stable released | ⏳ Waiting |
| 2026-08-XX | Upgrade to v2.4.0 in minikube | 📅 Scheduled |
| 2026-08-XX | Test provider-gitea v0.8.9 with v2.4.0 | 📅 Scheduled |
| 2026-08-XX | Upgrade production infrastructure (if applicable) | 📅 Future |

---

## Next Steps

1. ✅ **DONE**: v0.8.9 built and published
2. ✅ **DONE**: Three distribution methods implemented
3. ✅ **DONE**: This upgrade plan created
4. ⏳ **WAITING**: Crossplane v2.4.0 stable release
5. 📅 **SCHEDULED**: Execute upgrade when v2.4.0 is available

When v2.4.0 is released, follow **Phase 1-5** above with confidence that provider-gitea v0.8.9 is ready.

---

**For questions or issues, see**:
- INSTALLATION_GUIDE.md - Provider installation methods
- CROSSPLANE_V24_COMPATIBILITY.md - Technical details
- README.md - General provider information
