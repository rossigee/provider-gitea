#!/bin/bash

# Script to remove large files from git history
echo "🧹 Cleaning git history of large files..."

# Create a backup branch first
git branch backup-before-cleanup

# Remove large files from git history
echo "Removing large binary files..."
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch \
    provider \
    bin/provider \
    _output/bin/linux_amd64/provider \
    *.xpkg \
    *.tar.gz \
    provider-gitea-v*.xpkg \
    provider-gitea-v*-xpkg.tar.gz \
    package/provider-gitea-*.xpkg \
    _build/package/provider-gitea-*.xpkg' \
  --prune-empty --tag-name-filter cat -- --all

# Remove cache and build directories
echo "Removing cache and build directories..."
git filter-branch --force --index-filter \
  'git rm -r --cached --ignore-unmatch \
    .cache \
    _build \
    _output' \
  --prune-empty --tag-name-filter cat -- --all

# Clean up filter-branch refs
echo "Cleaning up temporary references..."
rm -rf .git/refs/original/
git reflog expire --expire=now --all
git gc --prune=now --aggressive

echo "✅ Git history cleanup complete!"
echo "📊 Repository size before/after:"
echo "Original size: 99M"
du -sh .git