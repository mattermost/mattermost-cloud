# Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.

################################################################################
##                             VERSION PARAMS                                 ##
################################################################################

## Tool Versions
GOLANG_VERSION := $(shell cat go.mod | grep "^go " | cut -d " " -f 2)
ALPINE_VERSION = 3.19
TERRAFORM_VERSION=1.5.5
KOPS_VERSION=v1.27.3
HELM_VERSION=v3.13.3
KUBECTL_VERSION=v1.27.9
POSTGRES_VERSION=14.8
ARCH ?= amd64

## Docker Build Versions
DOCKER_BUILD_IMAGE := golang:$(GOLANG_VERSION)
DOCKER_BASE_IMAGE = alpine:$(ALPINE_VERSION)

################################################################################

GO ?= $(shell command -v go 2> /dev/null)
PACKAGES=$(shell go list ./... | grep -v internal/mocks)
MATTERMOST_CLOUD_IMAGE ?= mattermost/mattermost-cloud:test
MATTERMOST_CLOUD_REPO ?= mattermost/mattermost-cloud
MATTERMOST_CLOUD_E2E_IMAGE ?= mattermost/mattermost-cloud-e2e:test
MACHINE = $(shell uname -m)
GOFLAGS ?= $(GOFLAGS:)
BUILD_TIME := $(shell date -u +%Y%m%d.%H%M%S)
BUILD_HASH := $(shell git rev-parse HEAD)

################################################################################
TEST_FLAGS ?= -v

TEST_PSQL_NAME ?= cloud-test
TEST_PSQL_PORT ?= 5439

################################################################################

BUILD_HASH = $(shell git rev-parse HEAD)
LDFLAGS += -X "github.com/mattermost/mattermost-cloud/model.BuildHash=$(BUILD_HASH)"

# Binaries.
TOOLS_BIN_DIR := $(abspath bin)
GO_INSTALL = ./scripts/go_install.sh
ENSURE_GOLANGCI_LINT = ./scripts/ensure_golangci-lint.sh

MOCKGEN_VER := v1.4.3
MOCKGEN_BIN := mockgen
MOCKGEN := $(TOOLS_BIN_DIR)/$(MOCKGEN_BIN)-$(MOCKGEN_VER)

OUTDATED_VER := master
OUTDATED_BIN := go-mod-outdated
OUTDATED_GEN := $(TOOLS_BIN_DIR)/$(OUTDATED_BIN)

GOVERALLS_VER := master
GOVERALLS_BIN := goveralls
GOVERALLS_GEN := $(TOOLS_BIN_DIR)/$(GOVERALLS_BIN)

GOIMPORTS_VER := master
GOIMPORTS_BIN := goimports
GOIMPORTS := $(TOOLS_BIN_DIR)/$(GOIMPORTS_BIN)

GOLANGCILINT_VER := v1.53.3
GOLANGCILINT_BIN := golangci-lint
GOLANGCILINT := $(TOOLS_BIN_DIR)/$(GOLANGCILINT_BIN)

TRIVY_SEVERITY := CRITICAL
TRIVY_EXIT_CODE := 1
TRIVY_VULN_TYPE := os,library

export GO111MODULE=on

## Checks the code style, tests, builds and bundles.
all: check-style dist

## Runs govet and gofmt against all packages.
.PHONY: check-style
check-style: govet lint goformat goimports
	@echo Checking for style guide compliance

## Runs lint against all packages.
lint: $(GOLANGCILINT)
	@echo Running golangci-lint
	$(GOLANGCILINT) run

## Runs lint against all packages for changes only
lint-changes: $(GOLANGCILINT)
	@echo Running golangci-lint over changes only
	$(GOLANGCILINT) run -n

## Runs govet against all packages.
.PHONY: vet
govet:
	@echo Running govet
	$(GO) vet ./...
	@echo Govet success

## Checks if files are formatted with go fmt.
.PHONY: goformat
goformat:
	@echo Checking if code is formatted
	@for package in $(PACKAGES); do \
		echo "Checking "$$package; \
		files=$$(go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $$package); \
		if [ "$$files" ]; then \
			gofmt_output=$$(gofmt -d -s $$files 2>&1); \
			if [ "$$gofmt_output" ]; then \
				echo "$$gofmt_output"; \
				echo "gofmt failed"; \
				echo "To fix it, run:"; \
				echo "go fmt [FAILED_PACKAGE]"; \
				exit 1; \
			fi; \
		fi; \
	done
	@echo "gofmt success"; \

.PHONY: dev-start
dev-start:
	docker-compose up -d

## Checks if imports are formatted correctly.
.PHONY: goimports
goimports: $(GOIMPORTS)
	@echo Checking if imports are sorted
	@for package in $(PACKAGES); do \
		echo "Checking "$$package; \
		files=$$(go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $$package); \
		if [ "$$files" ]; then \
			goimports_output=$$($(GOIMPORTS) -d $$files 2>&1); \
			if [ "$$goimports_output" ]; then \
				echo "$$goimports_output"; \
				echo "goimports failed"; \
				echo "To fix it, run:"; \
				echo "goimports -w [FAILED_PACKAGE]"; \
				exit 1; \
			fi; \
		fi; \
	done
	@echo "goimports success"; \

## Checks for vulnerabilities
trivy: build-image
	@echo running trivy
	@trivy image --format table --exit-code $(TRIVY_EXIT_CODE) --ignore-unfixed --vuln-type $(TRIVY_VULN_TYPE) --severity $(TRIVY_SEVERITY) $(MATTERMOST_CLOUD_IMAGE)

## Builds and thats all :)
.PHONY: dist
dist:	build

.PHONY: build
build: ## Build the mattermost-cloud
	@echo Building Mattermost-Cloud for ARCH=$(ARCH)
	@if [ "$(ARCH)" = "amd64" ]; then \
		export GOARCH="amd64"; \
	elif [ "$(ARCH)" = "arm64" ]; then \
		export GOARCH="arm64"; \
	elif [ "$(ARCH)" = "arm" ]; then \
		export GOARCH="arm"; \
	else \
		echo "Unknown architecture $(ARCH)"; \
		exit 1; \
	fi; \
	GOOS=linux CGO_ENABLED=0 $(GO) build -ldflags '$(LDFLAGS)' -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o ./build/_output/$(ARCH)/bin/cloud ./cmd/cloud

.PHONY: package
package: build ## Package mattermost-cloud binaries
	@echo Packaging Mattermost-Cloud for ARCH=$(ARCH)
	@mkdir -p dist
	@tar cfz dist/mattermost-cloud-linux-$(ARCH).tar.gz --strip-components=5 ./build/_output/$(ARCH)/bin/cloud

build-image:  ## Build the docker image for mattermost-cloud
	@echo Building Mattermost-cloud Docker Image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
	--platform linux/amd64,linux/arm64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile -t $(MATTERMOST_CLOUD_IMAGE) \
	--no-cache \
	--push

build-image-with-tag:  ## Build the docker image for mattermost-cloud
	@echo Building Mattermost-cloud Docker Image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
	--platform linux/amd64,linux/arm64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile -t $(MATTERMOST_CLOUD_IMAGE) -t $(MATTERMOST_CLOUD_IMAGE)-$(BUILD_TIME) -t $(MATTERMOST_CLOUD_REPO):${TAG} \
	--no-cache \
	--push

.PHONY: push-image-pr
push-image-pr:
	@echo Push Image PR
	./scripts/push-image-pr.sh

.PHONY: push-image
push-image:
	@echo Push Image
	./scripts/push-image.sh

get-terraform: ## Download terraform only if it's not available. Used in the docker build
	@if [ ! -f build/terraform ]; then \
		curl -Lo build/terraform.zip "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_$(ARCH).zip" &&\
		echo "Downloaded file details:" && ls -l build/terraform.zip &&\
		cd build && unzip terraform.zip &&\
		chmod +x terraform && rm terraform.zip;\
	fi

get-kops: ## Download kops only if it's not available. Used in the docker build
	@if [ ! -f build/kops ]; then \
		curl -Lo build/kops https://github.com/kubernetes/kops/releases/download/${KOPS_VERSION}/kops-linux-$(ARCH) &&\
		chmod +x build/kops;\
	fi

get-helm: ## Download helm only if it's not available. Used in the docker build
	@if [ ! -f build/helm ]; then \
		curl -Lo build/helm.tar.gz https://get.helm.sh/helm-${HELM_VERSION}-linux-$(ARCH).tar.gz &&\
		cd build && tar -zxvf helm.tar.gz &&\
		cp linux-$(ARCH)/helm helm && chmod +x helm && rm helm.tar.gz && rm -rf linux-$(ARCH);\
	fi

get-kubectl: ## Download kubectl only if it's not available. Used in the docker build
	@if [ ! -f build/kubectl ]; then \
		curl -Lo build/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/$(ARCH)/kubectl &&\
		chmod +x build/kubectl;\
	fi

.PHONY: install
install: build
	go install ./...

# Generate mocks from the interfaces.
.PHONY: mocks
mocks: $(MOCKGEN)
	go generate --mod=mod ./internal/mocks/...

.PHONY: code-gen
code-gen:
	@echo Installing provisioner-code-gen...
	go install ./cmd/provisioner-code-gen
	@echo Generating code
	go generate ./model

.PHONY: check-modules
check-modules: $(OUTDATED_GEN) ## Check outdated modules
	@echo Checking outdated modules
	$(GO) list -mod=mod -u -m -json all | $(OUTDATED_GEN) -update -direct

.PHONY: goverall
goverall: $(GOVERALLS_GEN) ## Runs goveralls
	$(GOVERALLS_GEN) -coverprofile=coverage.out -service=circle-ci -repotoken ${COVERALLS_REPO_TOKEN} || true

ifndef CLOUD_DATABASE
CLOUD_DATABASE ?= postgres://$(TEST_PSQL_NAME):$(TEST_PSQL_NAME)@localhost:$(TEST_PSQL_PORT)/$(TEST_PSQL_NAME)?sslmode=disable

unittest: unittest-create-db

.PHONY: unittest-create-db
unittest-create-db: unittest-destroy-db ## Start a postgresql database for unit tests, cleaning up any previous instance
	@echo Start a docker postgesql database
	@docker run --detach --rm --name $(TEST_PSQL_NAME) -p $(TEST_PSQL_PORT):5432 -e POSTGRES_USER=$(TEST_PSQL_NAME) -e POSTGRES_PASSWORD=$(TEST_PSQL_NAME) -e POSTGRES_DB=$(TEST_PSQL_NAME) -d postgres:$(POSTGRES_VERSION)-alpine

.PHONY: unittest-destroy-db
unittest-destroy-db: ## Destroy the postgresql database for unit tests
	@echo Destroy the docker postgesql database
	@docker stop $(TEST_PSQL_NAME) || true
endif

.PHONY: unittest
unittest:
	CLOUD_DATABASE=$(CLOUD_DATABASE) $(GO) test -failfast ./... ${TEST_FLAGS} -covermode=count -coverprofile=coverage.out

.PHONY: verify-mocks
verify-mocks: mocks
	@if !(git diff --quiet HEAD); then \
		git status \
		git diff \
		echo "generated files are out of date, run make mocks"; exit 1; \
	fi

.PHONY: build-image-e2e-pr
build-image-e2e-pr:
	@echo Building e2e image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
    --platform linux/amd64,linux/arm64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile.e2e -t $(MATTERMOST_CLOUD_E2E_IMAGE) \
	--no-cache \
    --push

.PHONY: build-image-e2e
build-image-e2e:
	@echo Building e2e image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
    --platform linux/amd64,linux/arm64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile.e2e -t $(MATTERMOST_CLOUD_E2E_IMAGE) -t  $(MATTERMOST_CLOUD_E2E_IMAGE)-$(BUILD_TIME) \
	--no-cache \
    --push

.PHONY: e2e-db-migration
e2e-db-migration:
	@echo Warning!
	@echo This may require adjusting some environment variables like DESTINATION_DB.
	@echo For full configuration check TestConfig struct in ./e2e/tests/dbmigration/suite.go
	@echo
	@echo Starting DB migration e2e test.
	go test ./e2e/tests/dbmigration -tags=e2e -v -timeout 30m

.PHONY: e2e-cluster
e2e-cluster:
	@echo Starting cluster e2e test.
	go test ./e2e/tests/cluster -tags=e2e -v -timeout 90m

.PHONY: e2e-installation
e2e-installation:
	@echo Starting installation e2e test
	go test ./e2e/tests/installation -tags=e2e -v -timeout 90m


.PHONY: e2e
e2e: e2e-cluster e2e-installation


## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(MOCKGEN): ## Build mockgen.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/golang/mock/mockgen $(MOCKGEN_BIN) $(MOCKGEN_VER)

$(OUTDATED_GEN): ## Build go-mod-outdated.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/psampaz/go-mod-outdated $(OUTDATED_BIN) $(OUTDATED_VER)

$(GOVERALLS_GEN): ## Build goveralls.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/mattn/goveralls $(GOVERALLS_BIN) $(GOVERALLS_VER)

$(GOIMPORTS): ## Build goimports.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) golang.org/x/tools/cmd/goimports $(GOIMPORTS_BIN) $(GOIMPORTS_VER)

$(GOLANGCILINT): ## Build golangci-lint
	BINDIR=$(TOOLS_BIN_DIR) TAG=$(GOLANGCILINT_VER) $(ENSURE_GOLANGCI_LINT)
