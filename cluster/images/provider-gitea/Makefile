# Image Makefile for provider-gitea

# Build arguments
TARGETOS ?= linux
TARGETARCH ?= amd64

# The registry and image name (will be set by parent makefile)
IMAGE ?= provider-gitea:latest

# Default target
img.build:
	@echo Building $(IMAGE)
	@docker build \
		--build-arg TARGETOS=$(TARGETOS) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		-t $(IMAGE) \
		-f Dockerfile ../../../

img.publish:
	@echo Publishing $(IMAGE)
	@docker push $(IMAGE)

img.promote:
	@echo Promoting $(FROM_IMAGE) to $(TO_IMAGE)
	@docker tag $(FROM_IMAGE) $(TO_IMAGE)
	@docker push $(TO_IMAGE)

.PHONY: img.build img.publish img.promote
