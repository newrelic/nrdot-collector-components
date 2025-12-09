#!/bin/bash
# Copyright New Relic
# SPDX-License-Identifier: New-Relic

# Checks for the presence of LICENSE files in all components.s
REPO_DIR="$( cd "$(dirname "$( dirname "${BASH_SOURCE[0]}" )")" &> /dev/null && pwd )"
NEW_RELIC_SOFTWARE_LICENSE="$(cat $REPO_DIR/internal/assets/license/LICENSE_NEW_RELIC_component.tmpl)"

MOD_TYPE_DIRS=("receiver" "exporter" "connector" "extension" "processor" "pkg")
for MOD_TYPE in "${MOD_TYPE_DIRS[@]}"; do
    component_dirs=$(find "$REPO_DIR/$MOD_TYPE" -mindepth 1 -maxdepth 1 -type d)
    for component in $component_dirs; do
        license=$(basename $(find "$component" -maxdepth 1 -type f -iname "LICENSE*"))
        component_path="${component#$REPO_DIR/}"

        if [ -z "$license" ]; then
            echo "❌ No license file found in $component_path"
            exit 1
        fi

        if [[ "$license" == "LICENSE_NEW_RELIC"* ]]; then
            licensing_file_entry=$(cat $REPO_DIR/LICENSING | grep "$component_path")
            if [ ! -n "$licensing_file_entry" ]; then
                echo "❌ Proprietary component $component_path not listed in LICENSING file."
                exit 1
            fi
            if [[ ! "$(cat "$component/$license")" == "$NEW_RELIC_SOFTWARE_LICENSE" ]]; then
                echo "❌ LICENSE file in $component_path does not contain the correct New Relic Software License text."
                exit 1
            fi
        fi
    done
done

# Validate that all components listed in LICENSING exist in file tree.
listed_component_paths=$(cat $REPO_DIR/LICENSING | grep '^[[:space:]]*-' | sed 's/^[[:space:]]*- //')
for component_path in $listed_component_paths; do
    if [ ! -d "$REPO_DIR/$component_path" ]; then
        echo "❌ Component $component_path mentioned in LICENSING file does not exist in the repository."
        exit 1
    fi
done

echo "✅ Licenses validated!"
