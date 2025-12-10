#!/usr/bin/env bash

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

#
# verifies if the collector components are using the main core collector version
# as a dependency.
#

source ./internal/buildscripts/modules

set -eu -o pipefail

mod_files=$(find . -type f -name "go.mod")

# Check if GNU sed is installed
GNU_SED_INSTALLED=false
if sed --version 2>/dev/null | grep -q "GNU sed"; then
   GNU_SED_INSTALLED=true
fi

# Return the collector main core version
get_collector_version() {
   collector_module="$1"
   main_mod_file="$2"

   if grep -q "$collector_module" "$main_mod_file"; then
      grep "$collector_module" "$main_mod_file" | (read -r mod version rest;
         echo "$version")
   else
      echo "Error: failed to retrieve the \"$collector_module\" version from \"$main_mod_file\"."
      exit 1
   fi
}

# Compare the collector main core version against all the collector component
# modules to verify that they are using this version as its dependency
check_collector_versions_correct() {
   collector_module="$1"
   collector_mod_version="$2"
   echo "Checking $collector_module is used with $collector_mod_version"

   # Loop through all the module files, checking the collector version
   if [ "${GNU_SED_INSTALLED}" = false ]; then
      sed -i '' "s|$collector_module [^ ]*|$collector_module $collector_mod_version|g" $mod_files
   else
      sed -i'' "s|$collector_module [^ ]*|$collector_module $collector_mod_version|g" $mod_files
   fi
}

MAIN_MOD_FILE="./cmd/nrdotcol/go.mod"


BETA_MODULE="go.opentelemetry.io/collector"
# Note space at end of string. This is so it filters for the exact string
# only and does not return string which contains this string as a substring.
BETA_MOD_VERSION=$(get_collector_version "$BETA_MODULE " "$MAIN_MOD_FILE")
check_collector_versions_correct "$BETA_MODULE" "$BETA_MOD_VERSION"
for mod in "${beta_modules[@]}"; do
   check_collector_versions_correct "$mod" "$BETA_MOD_VERSION"
done

# Check stable modules, none currently exist, uncomment when pdata is 1.0.0
STABLE_MODULE="go.opentelemetry.io/collector/pdata"
STABLE_MOD_VERSION=$(get_collector_version "$STABLE_MODULE" "$MAIN_MOD_FILE")
check_collector_versions_correct "$STABLE_MODULE" "$STABLE_MOD_VERSION"
for mod in "${stable_modules[@]}"; do
   check_collector_versions_correct "$mod" "$STABLE_MOD_VERSION"
done

# Get the latest patch version of contrib for the same minor as collector beta
get_contrib_version() {
   beta_version="$1"
   minor_version=$(echo "$beta_version" | cut -d. -f1,2)

   contrib_version=$(go list -m -versions github.com/open-telemetry/opentelemetry-collector-contrib/testbed 2>/dev/null | tr ' ' '\n' | grep "^$minor_version\." | sort -V | tail -1)

   if [ -z "$contrib_version" ]; then
      echo "$beta_version"
   else
      echo "$contrib_version"
   fi
}

# Update contrib dependencies (both direct and indirect) to match the target version
check_contrib_versions_correct() {
   contrib_version="$1"
   contrib_prefix="github.com/open-telemetry/opentelemetry-collector-contrib"

   echo "Checking contrib dependencies are using $contrib_version"

   for mod_file in $mod_files; do
      # Update all contrib dependencies (both direct and indirect)
      grep "^	$contrib_prefix/" "$mod_file" | while IFS= read -r line; do
         module=$(echo "$line" | awk '{print $1}')
         current_version=$(echo "$line" | awk '{print $2}')
         if [ "$current_version" != "$contrib_version" ]; then
            echo "Updating $module from $current_version to $contrib_version in $mod_file"
            if [ "${GNU_SED_INSTALLED}" = false ]; then
               sed -i '' "s|$module $current_version|$module $contrib_version|g" "$mod_file"
            else
               sed -i'' "s|$module $current_version|$module $contrib_version|g" "$mod_file"
            fi
         fi
      done || true
   done
}

CONTRIB_VERSION=$(get_contrib_version "$BETA_MOD_VERSION")
check_contrib_versions_correct "$CONTRIB_VERSION"

git diff --exit-code
