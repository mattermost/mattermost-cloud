#!/bin/bash

set -e
set -u

export TAG="${CIRCLE_SHA1:0:7}"

echo $DOCKER_PASSWORD | docker login --username $DOCKER_USERNAME --password-stdin

docker tag mattermost/mattermost-cloud:test mattermost/mattermost-cloud:$TAG

docker push mattermost/mattermost-cloud:$TAG
