#!/bin/bash
# Copyright New Relic
# SPDX-License-Identifier: New-Relic

# Traces through component_licenses.yaml to generate top-level and per-component LICENSE files.

REPO_DIR="$( cd "$(dirname "$( dirname "${BASH_SOURCE[0]}" )")" &> /dev/null && pwd )"

LICENSING="$(cat $REPO_DIR/internal/assets/license/LICENSING.tmpl)\n\n"

APACHE="$(cat $REPO_DIR/internal/assets/license/LICENSE_APACHE_component.tmpl)"
for component in $(yq '.apache[]' $REPO_DIR/component_licenses.yaml); do
    echo "generating license for $component"
    # Remove any existing license files
    find "$REPO_DIR/$component" -maxdepth 1 -type f -iname "LICENSE_*" -delete
    component_path="$REPO_DIR/$component"
    echo "$APACHE" > "$component_path/LICENSE_APACHE_$(basename $component)"
done

NEW_RELIC_SOFTWARE_LICENSE="$(cat $REPO_DIR/internal/assets/license/LICENSE_NEW_RELIC_component.tmpl)"
for component in $(yq '.new_relic[]' $REPO_DIR/component_licenses.yaml); do
    echo "generating license for $component"
    find "$REPO_DIR/$component" -maxdepth 1 -type f -iname "LICENSE_*" -delete
    component_path="$REPO_DIR/$component"
    echo "$NEW_RELIC_SOFTWARE_LICENSE" > "$component_path/LICENSE_NEW_RELIC_$(basename $component)"
    LICENSING+="New Relic Software License - $component"
done

echo "generating top-level LICENSING file"
echo -e "$LICENSING" > "$REPO_DIR/LICENSING"