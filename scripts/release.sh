#!/bin/bash
set -euo pipefail

if (( $# != 1 )); then
    printf 'Usage: %s vMAJOR.MINOR.PATCH\n' "$0" >&2
    exit 1
fi

version="$1"

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    printf 'Invalid version: %s\n' "$version" >&2
    exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

git fetch origin master

status="$(git status --porcelain)"
if [[ -n "$status" ]]; then
    printf 'Worktree must be clean\n' >&2
    exit 1
fi

head_commit="$(git rev-parse HEAD)"
origin_master_commit="$(git rev-parse origin/master)"
if [[ "$head_commit" != "$origin_master_commit" ]]; then
    printf 'HEAD must match origin/master\n' >&2
    exit 1
fi

if [[ -f VERSION ]]; then
    version_file_contents="$(<VERSION)"
    version_file_contents="${version_file_contents#"${version_file_contents%%[![:space:]]*}"}"
    version_file_contents="${version_file_contents%"${version_file_contents##*[![:space:]]}"}"
    if [[ "$version_file_contents" != "$version" ]]; then
        printf 'VERSION does not match %s\n' "$version" >&2
        exit 1
    fi
fi

local_tag="$(git tag --list "$version")"
if [[ -n "$local_tag" ]]; then
    printf 'Local tag already exists: %s\n' "$version" >&2
    exit 1
fi

remote_tag="$(git ls-remote --refs --tags origin "refs/tags/$version")"
if [[ -n "$remote_tag" ]]; then
    printf 'Remote tag already exists: %s\n' "$version" >&2
    exit 1
fi

git tag -a "$version" -m "Release $version" HEAD
git push origin "refs/tags/$version"
