#!/bin/bash -ex
# Copyright The OpenTelemetry Authors
# Modifications copyright New Relic, Inc.
#
# Modifications can be found at the following URL:
# https://github.com/newrelic/nrdot-collector-components/commits/main/.github/workflows/scripts/prepare-release-notes.sh?since=2025-11-26
#
# SPDX-License-Identifier: Apache-2.0

touch release-notes.md
echo "## End User Changelog" >> release-notes.md

awk '/<!-- next version -->/,/<!-- previous-version -->/' CHANGELOG.md > tmp-chlog.md # select changelog of latest version only
sed '1,3d' tmp-chlog.md >> release-notes.md # delete first 3 lines of file

echo "" >> release-notes.md
echo "## API Changelog" >> release-notes.md

awk '/<!-- next version -->/,/<!-- previous-version -->/' CHANGELOG-API.md > tmp-chlog-api.md # select changelog of latest version only
sed '1,3d' tmp-chlog-api.md >> release-notes.md # delete first 3 lines of file
