#!/bin/bash

# Build and push script for provider-gitea
set -e

# Default values
VERSION=${VERSION:-"latest"}
REGISTRY=${REGISTRY:-"ghcr.io/crossplane-contrib"}
IMAGE_NAME=${IMAGE_NAME:-"provider-gitea"}

echo "Building provider-gitea version: $VERSION"

# Build the provider
make build

# Build the Docker image
make docker-build

# Build the package
make xpkg.build

# Tag and push the image
docker tag crossplane/provider-gitea:latest $REGISTRY/$IMAGE_NAME:$VERSION
docker push $REGISTRY/$IMAGE_NAME:$VERSION

if [ "$VERSION" != "latest" ]; then
    docker tag $REGISTRY/$IMAGE_NAME:$VERSION $REGISTRY/$IMAGE_NAME:latest
    docker push $REGISTRY/$IMAGE_NAME:latest
fi

echo "Successfully built and pushed provider-gitea:$VERSION"