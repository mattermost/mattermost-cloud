#!/bin/bash

# Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.

set -euox

echo $DOCKERHUB_TOKEN | docker login --username $DOCKERHUB_USERNAME --password-stdin

docker tag mattermost/mattermost-cloud:test mattermost/mattermost-cloud:$TAG
docker tag mattermost/mattermost-cloud-e2e:test mattermost/mattermost-cloud-e2e:$TAG

docker push mattermost/mattermost-cloud:$TAG
docker push mattermost/mattermost-cloud-e2e:$TAG
