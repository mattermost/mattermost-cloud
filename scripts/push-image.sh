#!/bin/bash
set -e

if [ -n "${TAG}" ]
  then
    echo "Pushing ${TAG} for release ..."
else
  echo "Pushing latest for ${GITHUB_REF_NAME} ..."
  export TAG="latest"
fi

make build-image-with-tag
