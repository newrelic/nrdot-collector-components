#!/bin/bash
# Copyright New Relic, Inc. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

set -e

VERSION=''

while getopts v: flag
do
    case "${flag}" in
        v) VERSION=${OPTARG};;
        *) exit 1;;
    esac
done

if [ -z "$VERSION" ]; then
    echo "Error: VERSION is required"
    echo "Usage: $0 -v <version>"
    exit 1
fi

echo "Bumping Go version to $VERSION..."

# Determine the OS and set the sed function accordingly
if [[ "$OSTYPE" == "darwin"* ]]; then
  # macOS
  sed_inplace() {
    sed -i '' "$@"
  }
else
  # Linux
  sed_inplace() {
    sed -i "$@"
  }
fi

# Find all go.mod files
echo "Finding all go.mod files..."
GO_MOD_FILES=$(find . -name "go.mod" -type f)

# Update all go.mod files
echo "Updating all go.mod files..."
while IFS= read -r file; do
  sed_inplace -E "s/^go [0-9]+\.[0-9]+.*/go $VERSION/g" "$file"
done <<< "$GO_MOD_FILES"

echo ""
echo "✓ Successfully bumped golang version to $VERSION"
echo ""
