#!/bin/bash

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

files=(
    bin/nrdotcol_darwin_arm64
    bin/nrdotcol_darwin_amd64
    bin/nrdotcol_linux_arm64
    bin/nrdotcol_linux_amd64
    bin/nrdotcol_linux_ppc64le
    bin/nrdotcol_linux_riscv64
    bin/nrdotcol_linux_s390x
    bin/nrdotcol_windows_amd64.exe
    bin/nrdotcol_windows_arm64.exe
    # skip. See https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/10113
    # dist/otel-contrib-collector-*amd64.msi

);
for f in "${files[@]}"
do
    if [[ ! -f $f ]]
    then
        echo "$f does not exist."
        echo "passed=false" >> $GITHUB_OUTPUT
        exit 1
    fi
done
echo "passed=true" >> $GITHUB_OUTPUT
