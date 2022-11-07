#!/bin/bash

# Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.

set -e
set -u

if [[ -z "${CIRCLE_TAG:-}" ]]; then
  echo "Pushing lastest for $CIRCLE_BRANCH..."
  TAG=latest
else
  echo "Pushing release $CIRCLE_TAG..."
  TAG="$CIRCLE_TAG"
fi

echo $DOCKER_PASS | docker login --username $DOCKER_USER --password-stdin

docker tag mattermost/mattermost-cloud:test mattermost/mattermost-cloud:$TAG
docker tag mattermost/mattermost-cloud-e2e:test mattermost/mattermost-cloud-e2e:$TAG

docker push mattermost/mattermost-cloud:$TAG
docker push mattermost/mattermost-cloud-e2e:$TAG
