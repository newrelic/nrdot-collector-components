// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/internal/tools/tools.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools // import "github.com/newrelic/nrdot-collector-components/internal/tools"

// This file pins tool dependencies as direct imports so go-licence-detector
// and go list correctly classify them as direct (not indirect) dependencies.
// The go.mod `tool` directive is the canonical declaration; these blank imports
// keep the packages visible as direct deps for tooling that doesn't yet
// understand the tool directive (e.g. go-licence-detector).

import (
	_ "github.com/Khan/genqlient"
	_ "github.com/client9/misspell/cmd/misspell"
	_ "github.com/daixiang0/gci"
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/google/addlicense"
	_ "github.com/jcchavezs/porto/cmd/porto"
	_ "github.com/jstemmer/go-junit-report/v2"
	_ "github.com/rhysd/actionlint/cmd/actionlint"
	_ "go.elastic.co/go-licence-detector"
	_ "go.opentelemetry.io/build-tools/checkapi"
	_ "go.opentelemetry.io/build-tools/checkfile"
	_ "go.opentelemetry.io/build-tools/chloggen"
	_ "go.opentelemetry.io/build-tools/crosslink"
	_ "go.opentelemetry.io/build-tools/githubgen"
	_ "go.opentelemetry.io/build-tools/issuegenerator"
	_ "go.opentelemetry.io/build-tools/multimod"
	_ "go.opentelemetry.io/collector/cmd/builder"
	_ "go.opentelemetry.io/collector/cmd/mdatagen"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize"
	_ "golang.org/x/vuln/cmd/govulncheck"
	_ "gotest.tools/gotestsum"
	_ "mvdan.cc/gofumpt"

	_ "github.com/newrelic/nrdot-collector-components/cmd/codecovgen"
	_ "github.com/newrelic/nrdot-collector-components/cmd/nrlicense"
)
