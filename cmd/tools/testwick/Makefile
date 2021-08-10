# Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.

GOLANGCILINT_VER := v1.41.1
GOLANGCILINT_BIN := golangci-lint

LDFLAGS := -ldflags "-s -w -X main.BuildVersion=${TAG} -X main.BuildTime=$(shell date +%s)"\

.PHONY: build
build: # Builds the binary for testwick
	@echo Build binary for local
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o build/_output/bin/testwick ./

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
