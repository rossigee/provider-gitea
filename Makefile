# Project Setup
PROJECT_NAME := provider-gitea
PROJECT_REPO := github.com/crossplane-contrib/$(PROJECT_NAME)

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
-include build/makelib/golang.mk

# Setup Kubernetes tools
UP_VERSION = v0.28.0
UP_CHANNEL = stable
UPTEST_VERSION = v0.11.1
-include build/makelib/k8s_tools.mk

# Setup Images
IMAGES = provider-gitea
-include build/makelib/imagelight.mk

# Setup XPKG - Standardized registry configuration
# Primary registry: GitHub Container Registry under rossigee
XPKG_REG_ORGS ?= ghcr.io/rossigee
XPKG_REG_ORGS_NO_PROMOTE ?= ghcr.io/rossigee

# Optional registries (can be enabled via environment variables)
# To enable Harbor: export ENABLE_HARBOR_PUBLISH=true make publish XPKG_REG_ORGS=harbor.golder.lan/library
# To enable Upbound: export ENABLE_UPBOUND_PUBLISH=true make publish XPKG_REG_ORGS=xpkg.upbound.io/crossplane-contrib
XPKGS = provider-gitea
-include build/makelib/xpkg.mk

# NOTE: we force image building to happen prior to xpkg build so that we ensure
# image is present in daemon.
xpkg.build.provider-gitea: do.build.images

# Setup Package Metadata
export CROSSPLANE_VERSION := v1.20.0
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

.PHONY: test
test:
	@$(INFO) Running provider-gitea tests
	@go test -v ./...
	@$(OK) provider-gitea tests

# ====================================================================================
# Local Utilities

# This target is to setup local environment for testing
.PHONY: local-dev
local-dev: $(KIND) $(KUBECTL) $(CROSSPLANE_CLI) $(KUSTOMIZE) $(HELM3)
	@$(INFO) Setting up local development environment...
	@$(INFO) Make sure Docker is running...
	@echo "Use 'make run' to start the provider out-of-cluster for local testing"

.PHONY: e2e
e2e:
	@$(INFO) Running e2e tests...
	@go test -v ./test/e2e/... -timeout 1h