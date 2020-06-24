#!/usr/bin/env bash

# Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.

set -o errexit
set -o nounset
set -o pipefail
set -o verbose

KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

boilerDir="${KUBE_ROOT}/hack/boilerplate"
boiler="${boilerDir}/boilerplate.py"

files_need_boilerplate=()
while IFS=$'\n' read -r line; do
  files_need_boilerplate+=( "$line" )
done < <("${boiler}" "$@")

# Run boilerplate check
if [[ ${#files_need_boilerplate[@]} -gt 0 ]]; then
  for file in "${files_need_boilerplate[@]}"; do
    echo "Boilerplate header is wrong for: ${file}" >&2
  done

  exit 1
fi
