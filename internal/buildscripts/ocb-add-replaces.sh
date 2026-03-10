#!/usr/bin/env bash
# Copyright The OpenTelemetry Authors
# Modifications copyright New Relic, Inc.
#
# Modifications can be found at the following URL:
# https://github.com/newrelic/nrdot-collector-components/commits/main/internal/buildscripts/ocb-add-replaces.sh?since=2025-11-26
#
# SPDX-License-Identifier: Apache-2.0

set -e

DIR="$1"

# Handle CGO-enabled config variant
if [[ "$DIR" == "nrdotcol-cgo" ]]; then
    CONFIG_IN="cmd/nrdotcol/builder-config-cgo-enabled.yaml"
    CONFIG_OUT="cmd/nrdotcol/builder-config-cgo-enabled-replaced.yaml"
else
    CONFIG_IN="cmd/$DIR/builder-config.yaml"
    CONFIG_OUT="cmd/$DIR/builder-config-replaced.yaml"
fi

cp "$CONFIG_IN" "$CONFIG_OUT"

local_mods=$(find . -type f -name "go.mod" -exec dirname {} \; | sort)
for mod_path in $local_mods; do
    mod=${mod_path#"."} # remove initial dot
    echo "  - github.com/newrelic/nrdot-collector-components$mod => ../..$mod" >> "$CONFIG_OUT"
done
echo "Wrote replace statements to $CONFIG_OUT"
