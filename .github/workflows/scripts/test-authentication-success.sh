#!/bin/bash
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# Test to verify that WITH proper authentication, components CAN be successfully
# referenced by nrdot-collector-releases and other authorized systems.

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
echo "Authentication Success Test"
echo "=========================================="
echo ""
echo "This test verifies that WITH proper authentication,"
echo "components CAN be successfully referenced from"
echo "nrdot-collector-releases and other authorized systems."
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

# Check if SSH authentication is working
echo -e "${BLUE}Checking GitHub SSH Authentication...${NC}"
if ssh -T git@github.com 2>&1 | grep -q "successfully authenticated"; then
    echo -e "${GREEN}✓ PASS: GitHub SSH authentication is working${NC}"
else
    echo -e "${RED}✗ FAIL: GitHub SSH authentication not configured${NC}"
    echo ""
    echo "Please configure SSH authentication:"
    echo "  1. Generate SSH key: ssh-keygen -t ed25519 -C \"your_email@example.com\""
    echo "  2. Add to GitHub: cat ~/.ssh/id_ed25519.pub | pbcopy"
    echo "  3. Add to GitHub Settings > SSH Keys"
    EXIT_CODE=1
fi
echo ""

# Test 1: Verify nrdot-collector-releases repository is accessible
echo "=========================================="
echo -e "${BLUE}Test 1: nrdot-collector-releases Access${NC}"
echo "=========================================="
echo ""

if git ls-remote git@github.com:newrelic/nrdot-collector-releases.git >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS: nrdot-collector-releases repository is accessible${NC}"
else
    echo -e "${RED}✗ FAIL: Cannot access nrdot-collector-releases repository${NC}"
    EXIT_CODE=1
fi
echo ""

# Test 2: Verify nrdot-plus-collector-components repository is accessible
echo "=========================================="
echo -e "${BLUE}Test 2: nrdot-plus-collector-components Access${NC}"
echo "=========================================="
echo ""

if git ls-remote git@github.com:newrelic/nrdot-plus-collector-components.git >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS: nrdot-plus-collector-components repository is accessible${NC}"
else
    echo -e "${RED}✗ FAIL: Cannot access nrdot-plus-collector-components repository${NC}"
    EXIT_CODE=1
fi
echo ""

# Test 3: Verify components from opentelemetry-collector-contrib can be fetched
# (This is what nrdot-collector-releases currently uses)
echo "=========================================="
echo -e "${BLUE}Test 3: Fetch OTel Collector Contrib Components${NC}"
echo "=========================================="
echo ""

cd "$TEST_DIR"
mkdir -p test-otel && cd test-otel
go mod init test-otel >/dev/null 2>&1

COMPONENT="github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver@v0.135.0"
echo -e "${YELLOW}Fetching: ${COMPONENT}${NC}"

if go get "$COMPONENT" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS: Successfully fetched OTel Contrib component${NC}"
else
    echo -e "${RED}✗ FAIL: Could not fetch OTel Contrib component${NC}"
    EXIT_CODE=1
fi
echo ""

# Test 4: Verify that WITH proper git configuration, private repos can be accessed
echo "=========================================="
echo -e "${BLUE}Test 4: Git SSH Configuration for Private Repos${NC}"
echo "=========================================="
echo ""

# Check if git is configured to use SSH instead of HTTPS
if git config --get url."git@github.com:".insteadOf >/dev/null 2>&1; then
    INSTEADOF=$(git config --get url."git@github.com:".insteadOf)
    echo -e "${GREEN}✓ PASS: Git configured to use SSH: ${INSTEADOF} → git@github.com:${NC}"
else
    echo -e "${YELLOW}⚠ INFO: Git not configured to use SSH for HTTPS URLs${NC}"
    echo "  To enable, run:"
    echo "  git config --global url.\"git@github.com:\".insteadOf \"https://github.com/\""
fi
echo ""

# Test 5: Verify GOPRIVATE is set correctly (optional but recommended)
echo "=========================================="
echo -e "${BLUE}Test 5: GOPRIVATE Configuration${NC}"
echo "=========================================="
echo ""

if [ -n "$GOPRIVATE" ] && echo "$GOPRIVATE" | grep -q "github.com/newrelic"; then
    echo -e "${GREEN}✓ PASS: GOPRIVATE is configured: $GOPRIVATE${NC}"
elif [ -n "$GOPRIVATE" ]; then
    echo -e "${YELLOW}⚠ INFO: GOPRIVATE is set but doesn't include github.com/newrelic: $GOPRIVATE${NC}"
    echo "  Consider adding: export GOPRIVATE=\"github.com/newrelic/*\""
else
    echo -e "${YELLOW}⚠ INFO: GOPRIVATE not set${NC}"
    echo "  To set: export GOPRIVATE=\"github.com/newrelic/*\""
fi
echo ""

# Test 6: Test building a collector from nrdot-collector-releases manifest
echo "=========================================="
echo -e "${BLUE}Test 6: Build Collector from nrdot-collector-releases Manifest${NC}"
echo "=========================================="
echo ""

# Check if OCB builder exists
if [ ! -f "$REPO_ROOT/.tools/builder" ]; then
    echo -e "${YELLOW}⚠ INFO: OCB builder not found at $REPO_ROOT/.tools/builder${NC}"
    echo "Skipping OCB test. Run 'make' to build the OCB tool."
else
    # Clone nrdot-collector-releases and try to build a distribution
    cd "$TEST_DIR"
    echo -e "${YELLOW}Cloning nrdot-collector-releases...${NC}"

    if git clone --depth 1 git@github.com:newrelic/nrdot-collector-releases.git releases-test >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Successfully cloned nrdot-collector-releases${NC}"

        cd releases-test/distributions/nrdot-collector
        echo -e "${YELLOW}Testing OCB build with nrdot-collector manifest...${NC}"

        if "$REPO_ROOT/.tools/builder" --skip-compilation --config manifest.yaml 2>&1 | grep -q "Sources created\|Generating source codes"; then
            echo -e "${GREEN}✓ PASS: OCB successfully generated sources from nrdot-collector-releases manifest${NC}"
        else
            echo -e "${RED}✗ FAIL: OCB could not generate sources${NC}"
            EXIT_CODE=1
        fi
    else
        echo -e "${RED}✗ FAIL: Could not clone nrdot-collector-releases${NC}"
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
    echo "Authentication is properly configured and:"
    echo "  • nrdot-collector-releases repository IS accessible"
    echo "  • nrdot-plus-collector-components repository IS accessible"
    echo "  • Components CAN be fetched with proper authentication"
    echo "  • OCB CAN build collectors using authorized components"
    echo ""
    echo "The authentication mechanism is working correctly for authorized access."
else
    echo -e "${RED}✗ SOME TESTS FAILED${NC}"
    echo ""
    echo "Please review the failures above and ensure:"
    echo "  1. SSH authentication is configured for GitHub"
    echo "  2. You have access to New Relic private repositories"
    echo "  3. Git is configured to use SSH: git config --global url.\"git@github.com:\".insteadOf \"https://github.com/\""
    echo "  4. GOPRIVATE is set: export GOPRIVATE=\"github.com/newrelic/*\""
fi

echo ""
exit $EXIT_CODE