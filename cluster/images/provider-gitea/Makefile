# Image Makefile for provider-gitea

# Build arguments
PLATFORM ?= linux_amd64
TARGETOS ?= $(word 1, $(subst _, ,$(PLATFORM)))
TARGETARCH ?= $(word 2, $(subst _, ,$(PLATFORM)))
IMAGE_PLATFORM ?= $(subst _,/,$(PLATFORM))

# The registry and image name (will be set by parent makefile)
IMAGE ?= provider-gitea:latest

# Default target
img.build:
	@echo Building $(IMAGE)
	@docker buildx build \
		--platform $(IMAGE_PLATFORM) \
		--load \
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
