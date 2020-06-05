#!/bin/bash

set -e
set -u

if [[ -z "${CIRCLE_TAG:-}" ]]; then
  echo "Pushing lastest for $CIRCLE_BRANCH..."
  TAG=latest
else
  echo "Pushing release $CIRCLE_TAG..."
  TAG="$CIRCLE_TAG"
fi

echo $DOCKER_PASSWORD | docker login --username $DOCKER_USERNAME --password-stdin

docker tag mattermost/mattermost-cloud:test mattermost/mattermost-cloud:$TAG

docker push mattermost/mattermost-cloud:$TAG

