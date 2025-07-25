# Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.

# Build the mattermost cloud
ARG DOCKER_BUILD_IMAGE=golang:1.24
ARG DOCKER_BASE_IMAGE=alpine:3.19

FROM --platform=${TARGETPLATFORM} ${DOCKER_BUILD_IMAGE} AS build
ARG TARGETARCH
WORKDIR /mattermost-cloud/
COPY . /mattermost-cloud/

RUN apt-get update -yq && apt-get install -yq unzip && \
  make get-terraform ARCH=${TARGETARCH} get-kops ARCH=${TARGETARCH} get-helm ARCH=${TARGETARCH} get-kubectl ARCH=${TARGETARCH} && \
  make build ARCH=${TARGETARCH}

# Final Image
FROM --platform=${TARGETPLATFORM} ${DOCKER_BASE_IMAGE}
ARG TARGETARCH

LABEL name="Mattermost Cloud" \
  maintainer="cloud-team@mattermost.com" \
  vendor="Mattermost" \
  distribution-scope="public" \
  architecture="x86_64, arm64" \
  url="https://mattermost.com" \
  io.k8s.description="Mattermost Cloud creates, configures and helps manage K8s Clusters and Mattermost installations on Kubernetes" \
  io.k8s.display-name="Mattermost Cloud"

ENV CLOUD=/mattermost-cloud/cloud \
  USER_UID=10001 \
  USER_NAME=cloud

RUN  apk update && apk add libc6-compat \
                           ca-certificates \
                           mandoc \
                           aws-cli \
                           && rm -rf /var/cache/apk/*
COPY --from=build /mattermost-cloud/build/terraform /usr/local/bin/
COPY --from=build /mattermost-cloud/build/kops /usr/local/bin/
COPY --from=build /mattermost-cloud/build/helm /usr/local/bin/
COPY --from=build /mattermost-cloud/build/kubectl /usr/local/bin/
COPY --from=build /mattermost-cloud/helm-charts /mattermost-cloud/helm-charts
COPY --from=build /mattermost-cloud/manifests /mattermost-cloud/manifests
COPY --from=build /mattermost-cloud/build/_output/${TARGETARCH}/bin/cloud /mattermost-cloud/cloud
COPY --from=build /mattermost-cloud/build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup
WORKDIR /mattermost-cloud/

USER ${USER_UID}

EXPOSE 8075
EXPOSE 8076

ENTRYPOINT ["/usr/local/bin/entrypoint"]
