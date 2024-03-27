# Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.

GOLANGCILINT_VER := v1.41.1
GOLANGCILINT_BIN := golangci-lint

## Docker Build Versions
DOCKER_BUILD_IMAGE = golang:1.19.2
DOCKER_BASE_IMAGE = alpine:3.16.2
MATTERMOST_TESTWICK_IMAGE ?= mattermost/testwick:test


LDFLAGS := -ldflags "-s -w -X main.BuildVersion=${TAG} -X main.BuildTime=$(shell date +%s)"\

.PHONY: build
build: # Builds the binary for testwick
	@echo Build binary for local
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o build/_output/bin/main ./
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o build/_output/bin/testwick ./

.PHONY: build-image
build-image:  ## Build the docker image for testwick
	@echo Building testwick Docker Image
	docker buildx build \
	--platform linux/amd64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile -t $(MATTERMOST_TESTWICK_IMAGE) \
	--no-cache \
	--push

.PHONY: clean
clean: # Cleaning go and bin directories
	@echo Cleaning directories
	go clean
	rm -rf build/
	rm -f cover.out

.PHONY: setup
get-golangci-lint: # Installs lint for Linux only
	@echo Install golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s ${GOLANGCILINT_VER}
	cp ./bin/${GOLANGCILINT_BIN} ${GOPATH}/bin/
	@echo Installing golangci-lint success

.PHONY: lint # Runs lint against golang codebase
lint: $(GOLANGCILINT_GEN)
	@echo Running golangci lint
	${GOLANGCILINT_BIN} run -E gofmt --timeout 5m
	@echo lint success

.PHONY: unittest
unittest: # Runs the unit tests
	go test ./... -v -covermode=count -coverprofile=coverage.out
