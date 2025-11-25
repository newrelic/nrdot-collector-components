#!/bin/bash
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# Test to verify that private New Relic components cannot be fetched without authentication
# This test ensures OCB cannot reference private components without proper credentials

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
TEST_DIR=$(mktemp -d)
EXIT_CODE=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Authentication Requirement Test"
echo "=========================================="
echo ""
echo "This test verifies that private New Relic components"
echo "require authentication to be fetched by OCB."
echo ""

# Cleanup function
cleanup() {
    if [ -n "$TEST_DIR" ] && [ -d "$TEST_DIR" ]; then
        echo ""
        echo "Cleaning up test directory: $TEST_DIR"
        rm -rf "$TEST_DIR"
    fi
}
trap cleanup EXIT

# List of private components to test
PRIVATE_COMPONENTS=(
    "github.com/newrelic/nrdot-plus-collector-components/extension/pprofextension@v0.139.0"
    "github.com/newrelic/nrdot-plus-collector-components/receiver/carbonreceiver@v0.139.0"
    "github.com/newrelic/nrdot-plus-collector-components/exporter/prometheusexporter@v0.139.0"
    "github.com/newrelic/nrdot-plus-collector-components/processor/attributesprocessor@v0.139.0"
)

# Test each component
echo -e "${BLUE}Testing private components (should fail without authentication):${NC}"
echo ""

for component in "${PRIVATE_COMPONENTS[@]}"; do
    echo -e "${YELLOW}Testing: ${component}${NC}"

    # Create a fresh test module
    cd "$TEST_DIR"
    rm -rf test-module 2>/dev/null || true
    mkdir -p test-module
    cd test-module

    # Initialize a test module
    go mod init test-auth-check >/dev/null 2>&1

    # Try to fetch the component with direct proxy (bypass cache)
    # Unset authentication-related env vars to ensure clean test
    if (
        unset GOPRIVATE
        unset GH_TOKEN
        unset GITHUB_TOKEN
        export GOPROXY=direct
        export GIT_TERMINAL_PROMPT=0
        go get "$component" 2>&1
    ) | grep -q "could not read Username\|Authentication failed\|terminal prompts disabled\|repository not found"; then
        echo -e "  ${GREEN}✓ PASS: Authentication required (as expected)${NC}"
    else
        echo -e "  ${RED}✗ FAIL: Component was accessible without authentication${NC}"
        EXIT_CODE=1
    fi
    echo ""
done

# Test with OCB builder config
echo "=========================================="
echo -e "${BLUE}Testing OCB Builder Configuration${NC}"
echo "=========================================="
echo ""

# Check if OCB builder exists
if [ ! -f "$REPO_ROOT/.tools/builder" ]; then
    echo -e "${YELLOW}⚠ WARNING: OCB builder not found at $REPO_ROOT/.tools/builder${NC}"
    echo "Skipping OCB test. Run 'make' to build the OCB tool."
else
    # Create a minimal builder config with a private component
    cat > "$TEST_DIR/test-builder-config.yaml" <<EOF
dist:
  module: github.com/test/auth-test
  name: auth-test-collector
  description: Test collector for authentication verification
  version: v0.0.1-test
  output_path: ./auth-test

extensions:
  - gomod: github.com/newrelic/nrdot-plus-collector-components/extension/pprofextension v0.139.0

exporters:
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.140.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.140.0

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.140.0

providers:
  - gomod: go.opentelemetry.io/collector/confmap/provider/envprovider v1.40.0
EOF

    cd "$TEST_DIR"
    echo -e "${YELLOW}Testing OCB build with private component...${NC}"

    if "$REPO_ROOT/.tools/builder" --skip-compilation --config test-builder-config.yaml 2>&1 | \
       grep -q "could not read Username\|Authentication failed\|terminal prompts disabled\|invalid version.*git ls-remote"; then
        echo -e "${GREEN}✓ PASS: OCB correctly requires authentication${NC}"
    else
        echo -e "${RED}✗ FAIL: OCB did not require authentication${NC}"
        EXIT_CODE=1
    fi
fi

echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo ""

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED${NC}"
    echo ""
    echo "Private New Relic components correctly require authentication."
    echo "OCB cannot fetch these components without proper credentials."
    echo ""
    echo "To authenticate, use one of the following methods:"
    echo "  1. SSH: git config --global url.\"git@github.com:\".insteadOf \"https://github.com/\""
    echo "  2. GOPRIVATE: export GOPRIVATE=\"github.com/newrelic/*\""
    echo "  3. GitHub CLI: gh auth login"
else
    echo -e "${RED}✗ SOME TESTS FAILED${NC}"
    echo ""
    echo "WARNING: Private components may be accessible without authentication."
    echo "This could indicate a security issue."
fi

echo ""
exit $EXIT_CODE