#!/bin/bash
set -e
set -u

: ${GITHUB_REF_TYPE:?}
: ${GITHUB_REF_NAME:?}

if [ "${GITHUB_REF_TYPE:-}" = "branch" ]; then
  echo "Pushing latest for $GITHUB_REF_NAME..."
  export TAG=latest
else
  echo "Pushing release $GITHUB_REF_NAME..."
  export TAG="$GITHUB_REF_NAME"
fi
make build-image-with-tag
