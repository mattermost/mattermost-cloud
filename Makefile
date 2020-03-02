################################################################################
##                             VERSION PARAMS                                 ##
################################################################################

## Docker Build Versions
DOCKER_BUILD_IMAGE = golang:1.13
DOCKER_BASE_IMAGE = alpine:3.10

## Tool Versions
TERRAFORM_VERSION=0.11.14
KOPS_VERSION=1.15.0
HELM_VERSION=v2.14.2
KUBECTL_VERSION=v1.14.0

################################################################################

GO ?= $(shell command -v go 2> /dev/null)
MATTERMOST_CLOUD_IMAGE ?= mattermost/mattermost-cloud:test
MACHINE = $(shell uname -m)
GOFLAGS ?= $(GOFLAGS:)
BUILD_TIME := $(shell date -u +%Y%m%d.%H%M%S)
BUILD_HASH := $(shell git rev-parse HEAD)

################################################################################

AWS_SDK_URL := github.com/aws/aws-sdk-go
AWS_SDK_VERSION := $(shell find go.mod -type f -exec cat {} + | grep ${AWS_SDK_URL} | awk '{print $$NF}')
AWS_SDK_PATH := $(GOPATH)/pkg/mod/${AWS_SDK_URL}\@${AWS_SDK_VERSION}

LOGRUS_URL := github.com/sirupsen/logrus
LOGRUS_VERSION := $(shell find go.mod -type f -exec cat {} + | grep ${LOGRUS_URL} | awk '{print $$NF}')
LOGRUS_PATH := $(GOPATH)/pkg/mod/${LOGRUS_URL}\@${LOGRUS_VERSION}

export GO111MODULE=on

.PHONY: paths
paths:
	@echo ${LOGRUS_VERSION}
	@echo ${LOGRUS_PATH}

## Checks the code style, tests, builds and bundles.
all: check-style dist

## Runs govet and gofmt against all packages.
.PHONY: check-style
check-style: govet lint
	@echo Checking for style guide compliance

## Runs lint against all packages.
.PHONY: lint
lint:
	@echo Running lint
	env GO111MODULE=off $(GO) get -u golang.org/x/lint/golint
	golint -set_exit_status ./...
	@echo lint success

## Runs govet against all packages.
.PHONY: vet
govet:
	@echo Running govet
	$(GO) vet ./...
	@echo Govet success

## Builds and thats all :)
.PHONY: dist
dist:	build

.PHONY: build
build: ## Build the mattermost-cloud
	@echo Building Mattermost-Cloud
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/cloud  ./cmd/cloud

build-image:  ## Build the docker image for mattermost-cloud
	@echo Building Mattermost-cloud Docker Image
	docker build \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile -t $(MATTERMOST_CLOUD_IMAGE) \
	--no-cache

get-terraform: ## Download terraform only if it's not available. Used in the docker build
	@if [ ! -f build/terraform ]; then \
		curl -Lo build/terraform.zip https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && cd build && unzip terraform.zip &&\
		chmod +x terraform && rm terraform.zip;\
	fi

get-kops: ## Download kops only if it's not available. Used in the docker build
	@if [ ! -f build/kops ]; then \
		curl -Lo build/kops https://github.com/kubernetes/kops/releases/download/${KOPS_VERSION}/kops-linux-amd64 &&\
		chmod +x build/kops;\
	fi

get-helm: ## Download helm only if it's not available. Used in the docker build
	@if [ ! -f build/helm ]; then \
		curl -Lo build/helm.tar.gz https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz &&\
		cd build && tar -zxvf helm.tar.gz &&\
		cp linux-amd64/helm . && chmod +x helm && rm helm.tar.gz && rm -rf linux-amd64;\
	fi

get-kubectl: ## Download kubectl only if it's not available. Used in the docker build
	@if [ ! -f build/kubectl ]; then \
		curl -Lo build/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl &&\
		chmod +x build/kubectl;\
	fi

.PHONY: install
install: build
	go install ./...

# Generate mocks from the interfaces.
.PHONY: mocks
mocks:
	@if [ ! -f $(GOPATH)/pkg/mod ]; then \
		$(GO) mod download;\
	fi
	$(GOPATH)/bin/mockery -dir $(AWS_SDK_PATH)/service/ec2/ec2iface -all -output ./internal/mocks/aws-sdk/
	$(GOPATH)/bin/mockery -dir $(AWS_SDK_PATH)/service/rds/rdsiface -all -output ./internal/mocks/aws-sdk/
	$(GOPATH)/bin/mockery -dir $(AWS_SDK_PATH)/service/s3/s3iface -all -output ./internal/mocks/aws-sdk/
	$(GOPATH)/bin/mockery -dir $(AWS_SDK_PATH)/service/acm/acmiface -all -output ./internal/mocks/aws-sdk/
	$(GOPATH)/bin/mockery -dir $(AWS_SDK_PATH)/service/iam/iamiface -all -output ./internal/mocks/aws-sdk/
	$(GOPATH)/bin/mockery -dir $(AWS_SDK_PATH)/service/route53/route53iface -all -output ./internal/mocks/aws-sdk/
	$(GOPATH)/bin/mockery -dir $(AWS_SDK_PATH)/service/secretsmanager/secretsmanageriface -all -output ./internal/mocks/aws-sdk/
	$(GOPATH)/bin/mockery -dir $(LOGRUS_PATH)/ -all -output ./internal/mocks/logger/
	$(GOPATH)/bin/mockery -dir ./internal/tools/aws -all -output ./internal/mocks/aws-tools/

