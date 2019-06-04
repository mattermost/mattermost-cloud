GO ?= $(shell command -v go 2> /dev/null)

# GOOS/GOARCH of the build host, used to determine whether we're cross-compiling or not
BUILDER_GOOS_GOARCH="$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)"

BUNDLE_NAME ?= mattermost-cloud.tar.gz

export GO111MODULE=on

## Checks the code style, tests, builds and bundles.
all: check-style dist

## Runs govet and gofmt against all packages.
.PHONY: check-style
check-style: lint govet
	@echo Checking for style guide compliance

## Runs lint against all packages.
.PHONY: lint
lint:
	@echo Running lint
	env GO111MODULE=off $(GO) get -u golang.org/x/lint/golint
	golint -set_exit_status ./...
	@echo lint success

# Runs govet against all packages.
.PHONY: vet
govet:
	@echo Running govet
	$(GO) vet ./...
	@echo Govet success

## Builds and bundles.
.PHONY: dist
dist:	build bundle

## build.
.PHONY: build
build:
	rm -rf build/
	mkdir -p build/
	env GOOS=linux GOARCH=amd64 $(GO) install -i -pkgdir build $(GOFLAGS) -ldflags '$(LDFLAGS)' ./...
	env GOOS=darwin GOARCH=amd64 $(GO) install -i -pkgdir build $(GOFLAGS) -ldflags '$(LDFLAGS)' ./...
	env GOOS=windows GOARCH=amd64 $(GO) install -i -pkgdir build $(GOFLAGS) -ldflags '$(LDFLAGS)' ./...

## Generates a tar bundle of the plugin for install.
.PHONY: bundle
bundle:
	rm -rf dist/
	mkdir -p dist/
	mkdir -p dist/mattermost-cloud/bin
	cp -R operator-manifests dist/mattermost-cloud/

	@# ----- PLATFORM SPECIFIC -----
	@# Make osx package
	@# Copy binary
ifeq ($(BUILDER_GOOS_GOARCH),"darwin_amd64")
	cp $(GOPATH)/bin/cloud dist/mattermost-cloud/bin # from native bin dir, not cross-compiled
else
	cp $(GOPATH)/bin/darwin_amd64/cloud dist/mattermost-cloud/bin # from cross-compiled bin dir
endif
	@# Package
	tar -C dist -czf dist/mattermost-cloud-osx-amd64.tar.gz mattermost-cloud
	@# Cleanup
	rm -f dist/mattermost-cloud/bin/cloud

	@# Make windows package
	@# Copy binary
ifeq ($(BUILDER_GOOS_GOARCH),"windows_amd64")
	cp $(GOPATH)/bin/cloud.exe dist/mattermost-cloud/bin # from native bin dir, not cross-compiled
else
	cp $(GOPATH)/bin/windows_amd64/cloud.exe dist/mattermost-cloud/bin # from cross-compiled bin dir
endif
	@# Package
	cd dist && zip -9 -r -q -l mattermost-cloud-windows-amd64.zip mattermost-cloud && cd ..
	@# Cleanup
	rm -f dist/mattermost-cloud/bin/cloud.exe

	@# Make linux package
	@# Copy binary
ifeq ($(BUILDER_GOOS_GOARCH),"linux_amd64")
	cp $(GOPATH)/bin/cloud dist/mattermost-cloud/bin # from native bin dir, not cross-compiled
else
	cp $(GOPATH)/bin/linux_amd64/cloud dist/mattermost-cloud/bin # from cross-compiled bin dir
endif
	@# Package
	tar -C dist -czf dist/mattermost-cloud-linux-amd64.tar.gz mattermost-cloud
	@# Don't clean up native package so dev machines will have an unzipped package available
	@#rm -f dist/mattermost-cloud/bin/cloud

	rm -rf dist/mattermost-cloud
