#!/bin/bash

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

files=(
    bin/nrdotpluscol_darwin_arm64
    bin/nrdotpluscol_darwin_amd64
    bin/nrdotpluscol_linux_arm64
    bin/nrdotpluscol_linux_amd64
    bin/nrdotpluscol_linux_ppc64le
    bin/nrdotpluscol_linux_riscv64
    bin/nrdotpluscol_linux_s390x
    bin/nrdotpluscol_windows_amd64.exe
    bin/nrdotpluscol_windows_arm64.exe
    # skip. See https://github.com/newrelic/nrdot-plus-collector-components/issues/10113
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
