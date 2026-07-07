# Project Setup
PROJECT_NAME := provider-gitea
PROJECT_REPO := github.com/rossigee/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64
-include build/makelib/common.mk

# Setup Output
-include build/makelib/output.mk

# Setup Go
NPROCS ?= 1
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal apis
GO111MODULE = on
# Override golangci-lint version for modern Go support
GOLANGCILINT_VERSION ?= 2.12.2
-include build/makelib/golang.mk

# Setup Kubernetes tools
UP_VERSION = v0.28.0
UP_CHANNEL = stable
UPTEST_VERSION = v0.11.1
-include build/makelib/k8s_tools.mk

# Setup Images
IMAGES = provider-gitea
# Force registry override (can be overridden by make command arguments)
REGISTRY_ORGS = rossigee
-include build/makelib/imagelight.mk

# Setup XPKG - Standardized registry configuration
# Force registry override (can be overridden by make command arguments)
XPKG_REG_ORGS = ghcr.io/rossigee
XPKG_REG_ORGS_NO_PROMOTE = ghcr.io/rossigee

# Optional registries (can be enabled via environment variables)
# Harbor publishing has been removed - using only ghcr.io/rossigee
# To enable Upbound: export ENABLE_UPBOUND_PUBLISH=true make publish XPKG_REG_ORGS=xpkg.upbound.io/crossplane-contrib
XPKGS = provider-gitea
-include build/makelib/xpkg.mk

# NOTE: we force image building to happen prior to xpkg build so that we ensure
# image is present in daemon.
xpkg.build.provider-gitea: do.build.images

# Ensure publish only happens on release branches
publish.artifacts:
	@if ! echo "$(BRANCH_NAME)" | grep -qE "$(subst $(SPACE),|,main|master|release-.*)"; then \ 
		$(ERR) Publishing is only allowed on branches matching: main|master|release-.* (current: $(BRANCH_NAME)); \ 
		exit 1; \ 
	fi
	$(foreach r,$(XPKG_REG_ORGS), $(foreach x,$(XPKGS),@$(MAKE) xpkg.release.publish.$(subst /,_,$(r)).$(x)))
	$(foreach r,$(REGISTRY_ORGS), $(foreach i,$(IMAGES),@$(MAKE) img.release.publish.$(subst /,_,$(r)).$(i)))

# Alias for publish.artifacts to match workflow expectations
publish: publish.artifacts

# ====================================================================================
# XPKG Publishing Overrides - Fix for Crossplane v2 metadata preservation
#
# The build/makelib/xpkg.mk publish targets were losing metadata during OCI push.
# This override ensures the xpkg files are properly verified and published with
# full OCI layer metadata intact for Crossplane v2 compatibility.

# Override the problematic xpkg.release.publish target with improved version
xpkg.release.publish.%:
	@registry_org=$(word 1,$(subst ., ,$*)); \
	xpkg_name=$(word 2,$(subst ., ,$*)); \
	$(INFO) "Publishing xpkg $$registry_org/$$xpkg_name:$(VERSION)"; \
	for platform in $(XPKG_LINUX_PLATFORMS); do \
		xpkg_file=$(XPKG_OUTPUT_DIR)/$${platform}/$${xpkg_name}-$(VERSION).xpkg; \
		if [ ! -f "$$xpkg_file" ]; then \
			$(ERR) "XPKG file not found: $$xpkg_file"; \
			exit 1; \
		fi; \
		echo "  Verifying $$platform: $$xpkg_file"; \
		tar -tf "$$xpkg_file" manifest.json >/dev/null 2>&1 || { $(ERR) "Invalid OCI image format in $$xpkg_file"; exit 1; }; \
		tar -xOf "$$xpkg_file" manifest.json | grep -q '"Config"' || { $(ERR) "Invalid OCI manifest in $$xpkg_file"; exit 1; }; \
	done
	@$(INFO) "All xpkg files validated. Pushing to registry..."
	@$(CROSSPLANE_CLI) xpkg push \
		$(foreach p,$(XPKG_LINUX_PLATFORMS),--package-files $(XPKG_OUTPUT_DIR)/$(p)/$(xpkg_name)-$(VERSION).xpkg ) \
		$(registry_org)/$(xpkg_name):$(VERSION) || $(FAIL)
	@$(OK) "Published $$registry_org/$$xpkg_name:$(VERSION)"
	@$(INFO) "IMPORTANT: Verifying published image contains metadata layers..."
	@docker pull $(registry_org)/$(xpkg_name):$(VERSION) >/dev/null 2>&1 && \
		image_layers=$$(docker inspect $(registry_org)/$(xpkg_name):$(VERSION) 2>/dev/null | grep -c '"digest"' || echo 0); \
		if [ "$$image_layers" -gt 0 ]; then \
			$(OK) "Published image has $$image_layers layer(s) - metadata should be present"; \
		else \
			$(WARN) "Warning: Published image appears to lack proper OCI layers"; \
		fi

# Setup Package Metadata
export CROSSPLANE_VERSION := v2.3.3
-include build/makelib/local.xpkg.mk
-include build/makelib/controlplane.mk

# Targets

# run `make submodules` after cloning the repository for the first time.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# Update the submodules, such as the common build scripts.
submodules.update:
	@git submodule update --remote --merge

# We want submodules to be set up the first time `make` is run.
# We manage the build/ folder and its Makefiles as a submodule.
# The first time `make` is run, the includes of build/*.mk files will
# all fail, and this target will be run. The next time, the default as defined
# by the includes will be run instead.
fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# Generate a coverage report for cobertura applying exclusions on
# - generated file
go.test.coverage:
	@$(INFO) go test coverage
	@go test -v -coverprofile=coverage.out -covermode=count ./...
	@$(OK) go test coverage

# This is for running out-of-cluster locally, and is for convenience. Running
# this make target will print out the command which was used. For more control,
# try running the binary directly with different arguments.
run: go.build
	@$(INFO) Running Crossplane locally out-of-cluster . . .
	@# To see other arguments that can be provided, run the command with --help instead
	$(GO_OUT_DIR)/provider --debug

# Run unit tests (excludes integration tests which require kubebuilder test environment)
.PHONY: test.unit
test.unit:
	@$(INFO) Running provider-gitea unit tests
	@go test -v $$(go list ./... | grep -v '/test/integration' | grep -v '/test/e2e')
	@$(OK) provider-gitea unit tests

# Run integration tests (requires kubebuilder test tools: etcd, kube-apiserver)
# To set up kubebuilder test environment, see: https://book.kubebuilder.io/reference/envtest.html
.PHONY: test.integration
test.integration:
	@$(INFO) Running provider-gitea integration tests (requires kubebuilder test tools)
	@go test -v ./test/integration/...
	@$(OK) provider-gitea integration tests

# Override common.mk test target to run our unit tests
.PHONY: test
test: test.unit

# ====================================================================================
# Local Utilities

# This target is to setup local environment for testing
.PHONY: local-dev
local-dev: $(KIND) $(KUBECTL) $(CROSSPLANE_CLI) $(KUSTOMIZE) $(HELM3)
	@$(INFO) Setting up local development environment...
	@$(INFO) Make sure Docker is running...
	@echo "Use 'make run' to start the provider out-of-cluster for local testing"

# Run end-to-end tests (requires Kubernetes cluster)
.PHONY: test.e2e
test.e2e:
	@$(INFO) Running provider-gitea e2e tests (requires Kubernetes cluster)
	@go test -v ./test/e2e/... -timeout 1h
	@$(OK) provider-gitea e2e tests
