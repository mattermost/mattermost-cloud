#!/bin/bash

# Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.

set -euox

echo $DOCKERHUB_TOKEN | docker login --username $DOCKERHUB_USERNAME --password-stdin

if [ "$TAG" = "" ]; then
    echo "TAG was not provided"
    exit 1
fi

echo "Tagging images with SHA $TAG"

docker tag mattermost/mattermost-cloud:test mattermost/mattermost-cloud:$TAG
docker tag mattermost/mattermost-cloud-e2e:test mattermost/mattermost-cloud-e2e:$TAG

docker push mattermost/mattermost-cloud:$TAG
docker push mattermost/mattermost-cloud-e2e:$TAG

if [ "$REF_NAME" = "master" ] || [ "$REF_NAME" = "main" ]; then
    echo "Tagging images with 'latest' tag"

    docker tag mattermost/mattermost-cloud:test mattermost/mattermost-cloud:latest
    docker tag mattermost/mattermost-cloud-e2e:test mattermost/mattermost-cloud-e2e:latest

    docker push mattermost/mattermost-cloud:latest
    docker push mattermost/mattermost-cloud-e2e:latest
fi
