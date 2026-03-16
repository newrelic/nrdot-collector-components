#!/bin/bash
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
# Update all go.mod files
echo "Updating all go.mod files..."
find . -name "go.mod" -type f -exec bash -c 'sed_inplace() { if [[ "$OSTYPE" == "darwin"* ]]; then sed -i "" "$@"; else sed -i "$@"; fi; }; sed_inplace -E "s/^go [0-9]+\.[0-9]+.*/go '"$VERSION"'/g" "$1"' bash {} \;

echo ""
echo "✓ Successfully bumped golang version to $VERSION"
echo ""