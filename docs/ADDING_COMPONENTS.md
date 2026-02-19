# Adding New Components to nrdot-collector-components

This guide provides step-by-step instructions for adding new components (receivers, exporters, processors, extensions, or connectors) to the nrdot-collector-components repository.

## Table of Contents
- [Quick Reference Checklist](#quick-reference-checklist)
- [Prerequisites](#prerequisites)
- [Step-by-Step Guide](#step-by-step-guide)
- [Required Files](#required-files)
- [Test Requirements](#test-requirements)
- [Files to Update](#files-to-update)
- [Common Issues](#common-issues)

## Quick Reference Checklist

Use this checklist to track your progress when adding a new component:

### Initial Setup
- [ ] Open an issue proposing the new component
- [ ] Find a sponsor (approver or maintainer)
- [ ] Create component directory structure
- [ ] Create `metadata.yaml`
- [ ] Create `doc.go` with generate pragma
- [ ] Create component implementation files

### Core Implementation
- [ ] Implement component interface (factory, config, component)
- [ ] Create `go.mod` file for the component
- [ ] Create boilerplate `Makefile`
- [ ] Write unit tests (target 80%+ coverage)
- [ ] Create `README.md` with usage examples

### Integration
- [ ] Add component to `versions.yaml`
- [ ] Add component to `cmd/nrdotcol/builder-config.yaml` (or `builder-config-cgo-enabled.yaml` if requires CGO)
- [ ] Add component to `cmd/oteltestbedcol/builder-config.yaml` (if testable)
- [ ] Run `make crosslink`
- [ ] Run `make generate`
- [ ] Run `make gencodeowners`
- [ ] Run `make generate-gh-issue-templates`

### Quality Checks
- [ ] Run `make checkdoc`
- [ ] Run `make checkmetadata`
- [ ] Run `make checkapi`
- [ ] Run `make goporto`
- [ ] Run `make gotidy`
- [ ] Run `make gennrdotcol` (and `make gennrdotcol-cgo` if component requires CGO)
- [ ] Run `make genoteltestbedcol`
- [ ] Run `make multimod-verify`
- [ ] Run `make gengithub`
- [ ] Run `make addlicense`
- [ ] Run `make gotest` (all tests pass)
- [ ] Run `make golint` (no lint errors)

### Changelog & PR
- [ ] Run `make chlog-new` to create changelog entry
- [ ] Fill in changelog details
- [ ] Run `make chlog-validate`
- [ ] Create pull request with proper title format
- [ ] Address review feedback

## Prerequisites

Before adding a component:

1. **Proposal**: Open an issue using the [new component template](https://github.com/newrelic/nrdot-collector-components/issues/new?assignees=&labels=Sponsor+Needed%2Cneeds+triage&projects=&template=new_component.yaml&title=New+component%3A+)

2. **Sponsor**: Find a sponsor (approver or maintainer) who will:
   - Review your code
   - Be a code owner for the component
   - Commit to maintaining the component

3. **Development Environment**: Ensure you have:
   - Go 1.24 or later installed
   - Git configured
   - GitHub access token (for `make update-codeowners`)

## Step-by-Step Guide

### 1. Create Component Structure

Create your component directory under the appropriate type folder:

```bash
# For a receiver named "myreceiver"
mkdir -p receiver/myreceiver

# For an exporter named "myexporter"
mkdir -p exporter/myexporter

# For a processor named "myprocessor"
mkdir -p processor/myprocessor

# For an extension named "myextension"
mkdir -p extension/myextension

# For a connector named "myconnector"
mkdir -p connector/myconnector
```

### 2. Create metadata.yaml

Create `metadata.yaml` with the minimum required fields:

```yaml
type: mycomponent  # Component name (e.g., apache, http, postgresql)

status:
  class: receiver  # One of: cmd, connector, exporter, extension, processor, receiver
  stability:
    development: [logs, metrics, traces]  # Or [extension] for extensions
  codeowners:
    active: [github-username]  # Your GitHub username and sponsor's username
  distributions: []  # Empty for development, add [nrdot] when ready for alpha

github_project: newrelic/nrdot-collector-components
```

### 3. Create doc.go

Create `doc.go` with a generate pragma:

```go
// Copyright 2025 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

// Package myreceiver implements a receiver for...
package myreceiver // import "github.com/newrelic/nrdot-collector-components/receiver/myreceiver"
```

### 4. Create go.mod

Create a `go.mod` file for your component:

```bash
cd receiver/myreceiver
go mod init github.com/newrelic/nrdot-collector-components/receiver/myreceiver
go mod tidy
```

### 5. Create Makefile

Create a boilerplate `Makefile` that references the top-level Makefile:

```makefile
include ../../Makefile.Common
```

### 6. Implement Component

Create the following files:

- **config.go**: Configuration structure
- **factory.go**: Factory implementation
- **[component].go**: Core component logic
- **[component]_test.go**: Unit tests

**Important**: All `.go` files must include the New Relic copyright header:
```go
// Copyright 2025 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
```

Example structure for a receiver:

```go
// config.go
// Copyright 2025 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package myreceiver

import "go.opentelemetry.io/collector/component"

type Config struct {
    // Your configuration fields
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
    // Validation logic
    return nil
}

// factory.go
// Copyright 2025 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package myreceiver

import (
    "context"
    "go.opentelemetry.io/collector/component"
    "go.opentelemetry.io/collector/consumer"
    "go.opentelemetry.io/collector/receiver"
)

func NewFactory() receiver.Factory {
    return receiver.NewFactory(
        component.MustNewType("myreceiver"),
        createDefaultConfig,
        receiver.WithTraces(createTracesReceiver, component.StabilityLevelDevelopment),
    )
}

func createDefaultConfig() component.Config {
    return &Config{}
}

func createTracesReceiver(
    _ context.Context,
    set receiver.Settings,
    cfg component.Config,
    consumer consumer.Traces,
) (receiver.Traces, error) {
    // Implementation
    return nil, nil
}
```

### 7. Write Tests

Create comprehensive tests with 80%+ coverage:

```go
// myreceiver_test.go
// Copyright 2025 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package myreceiver

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.opentelemetry.io/collector/consumer/consumertest"
    "go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewFactory(t *testing.T) {
    factory := NewFactory()
    require.NotNil(t, factory)
    assert.Equal(t, component.MustNewType("myreceiver"), factory.Type())
}

func TestCreateTracesReceiver(t *testing.T) {
    factory := NewFactory()
    cfg := factory.CreateDefaultConfig()

    receiver, err := factory.CreateTracesReceiver(
        context.Background(),
        receivertest.NewNopSettings(),
        cfg,
        consumertest.NewNop(),
    )

    require.NoError(t, err)
    require.NotNil(t, receiver)
}
```

### 8. Create README.md

Create a README with status badge and usage examples:

```markdown
# My Receiver

<!-- status autogenerated section -->
<!-- end autogenerated section -->

## Overview

[Brief description of what your component does]

## Configuration

```yaml
receivers:
  myreceiver:
    # Configuration options
```

## Example

[Full configuration example]
```

### 9. Generate Code and Documentation

Run the code generators:

```bash
# From repo root
make generate
make gencodeowners
make generate-gh-issue-templates
```

### 10. Update Repository Files

#### Add to versions.yaml

Edit `versions.yaml` and add your component to the `modules` list:

```yaml
module-sets:
  beta:
    version: v0.141.0
    modules:
      - github.com/newrelic/nrdot-collector-components
      # ... other modules
      - github.com/newrelic/nrdot-collector-components/receiver/myreceiver
```

#### Add to builder-config.yaml

Edit `cmd/nrdotcol/builder-config.yaml` (or `cmd/nrdotcol/builder-config-cgo-enabled.yaml` if your component requires CGO) and `cmd/oteltestbedcol/builder-config.yaml`:

**Note:** Components that require CGO (e.g., those using C libraries like Oracle database drivers) should be added to `builder-config-cgo-enabled.yaml` instead of the standard `builder-config.yaml`.

```yaml
receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.141.0
  - gomod: github.com/newrelic/nrdot-collector-components/receiver/myreceiver v0.141.0
```

### 11. Run Quality Checks

Run all quality checks from the repository root:

```bash
make checkdoc
make checkmetadata
make checkapi
make goporto
make crosslink
make gotidy
make gennrdotcol
# If component requires CGO, also run:
make gennrdotcol-cgo
make genoteltestbedcol
make multimod-verify
make gengithub
make addlicense
```

Fix any issues reported by these commands.

### 12. Run Tests and Linting

```bash
# Run all tests
make gotest

# Run linting
make golint

# Or run tests for your specific component
cd receiver/myreceiver
make test
make lint
```

### 13. Create Changelog Entry

```bash
make chlog-new
# Edit the generated .chloggen/<branch-name>.yaml file
make chlog-validate
```

Example changelog entry:

```yaml
change_type: enhancement  # One of: enhancement, bug_fix, breaking, deprecation, new_component

component: receiver/myreceiver

note: Add myreceiver for collecting data from My Service

issues: [123]  # GitHub issue numbers

subtext: |
  This receiver collects metrics and traces from My Service.
  Key features:
  - Feature 1
  - Feature 2
```

## Required Files

Every component must include:

### Mandatory Files
- `metadata.yaml` - Component metadata
- `doc.go` - Package documentation with generate pragma
- `go.mod` - Go module definition
- `Makefile` - Build configuration
- `README.md` - User documentation
- `config.go` - Configuration structure
- `factory.go` - Factory implementation
- `[component].go` - Core implementation
- `[component]_test.go` - Unit tests

### Generated Files (do not edit manually)
- `generated_component_test.go` - Auto-generated component tests
- `generated_package_test.go` - Auto-generated package tests
- `internal/metadata/generated_*.go` - Auto-generated metadata code

## Test Requirements

### Minimum Requirements
- **Code Coverage**: Target 80% or higher
- **Unit Tests**: Test all exported functions and types
- **Component Tests**: Test component lifecycle (Start, Shutdown)
- **Configuration Tests**: Test config validation and defaults
- **Integration Tests**: Test end-to-end functionality (if applicable)

### Required Test Patterns

#### Factory Tests
```go
func TestNewFactory(t *testing.T) {
    factory := NewFactory()
    require.NotNil(t, factory)
    assert.Equal(t, component.MustNewType("myreceiver"), factory.Type())
}

func TestCreateDefaultConfig(t *testing.T) {
    factory := NewFactory()
    cfg := factory.CreateDefaultConfig()
    require.NotNil(t, cfg)
    require.NoError(t, component.ValidateConfig(cfg))
}
```

#### Component Lifecycle Tests
```go
func TestComponentLifecycle(t *testing.T) {
    factory := NewFactory()
    cfg := factory.CreateDefaultConfig()

    receiver, err := factory.CreateTracesReceiver(
        context.Background(),
        receivertest.NewNopSettings(),
        cfg,
        consumertest.NewNop(),
    )
    require.NoError(t, err)

    // Test Start
    err = receiver.Start(context.Background(), componenttest.NewNopHost())
    require.NoError(t, err)

    // Test Shutdown
    err = receiver.Shutdown(context.Background())
    require.NoError(t, err)
}
```

#### Configuration Validation Tests
```go
func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name    string
        config  *Config
        wantErr bool
    }{
        {
            name:    "valid config",
            config:  &Config{/* valid fields */},
            wantErr: false,
        },
        {
            name:    "invalid config",
            config:  &Config{/* invalid fields */},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests in the repository
make gotest

# Run tests for specific component
cd receiver/myreceiver
make test

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Files to Update

When adding a new component, you must update these files:

| File | Purpose | Action |
|------|---------|--------|
| `versions.yaml` | Module version tracking | Add your component module |
| `cmd/nrdotcol/builder-config.yaml` | nrdotcol binary | Add component import |
| `cmd/oteltestbedcol/builder-config.yaml` | Test binary | Add component import (if testable) |
| `.github/CODEOWNERS` | Code ownership | Auto-generated by `make gencodeowners` |
| `.github/ISSUE_TEMPLATE/...` | Issue templates | Auto-generated by `make generate-gh-issue-templates` |
| `CHANGELOG.md` | User changelog | Auto-generated during release from `.chloggen/*.yaml` |

## Common Issues

### Issue: "module not found" errors

**Solution**: Run `make crosslink` to update intra-repository dependencies.

### Issue: Lint failures

**Solution**:
1. Run `make addlicense` to add license headers
2. Run `gofmt -w .` to format code
3. Run `make golint` to see remaining issues

### Issue: Generated files out of sync

**Solution**: Run `make generate` to regenerate all generated files.

### Issue: Tests fail with import errors

**Solution**:
1. Run `go mod tidy` in your component directory
2. Run `make crosslink` from repository root
3. Run `make gotidy` from repository root

### Issue: CODEOWNERS not updated

**Solution**:
1. Ensure you have a GitHub personal access token configured
2. Run `make update-codeowners` or manually update `.github/CODEOWNERS`
3. Run `make generate` to update README headers

### Issue: Component not included in binary

**Solution**: Verify the component is added to both:
- `versions.yaml`
- `cmd/nrdotcol/builder-config.yaml`
- Then run `make gennrdotcol`

## Pull Request Guidelines

### PR Title Format

```
[receiver/myreceiver] Add new receiver for My Service
```

Format: `[component-type/component-name] brief description`

### PR Description

Include:
- Link to the proposal issue (`Resolves #123`)
- Component overview
- Configuration example
- Test coverage percentage
- Checklist of completed items

### PR Review Process

1. **First PR**: Component structure and factory (usually trivial)
   - Mark stability as "In Development"
   - Focus on basic structure and interface implementation

2. **Second PR**: Full implementation
   - Complete functionality
   - Comprehensive tests
   - Full documentation

3. **Final PR**: Alpha promotion
   - Update `metadata.yaml` stability to `alpha`
   - Add `nrdot` to distributions list
   - Ensure 80%+ test coverage

## Additional Resources

- [OpenTelemetry Collector Contributing Guide](https://github.com/open-telemetry/opentelemetry-collector/blob/main/CONTRIBUTING.md)
- [Building a Trace Receiver Tutorial](https://opentelemetry.io/docs/collector/trace-receiver/)
- [Metadata Generator Documentation](https://github.com/open-telemetry/opentelemetry-collector/blob/main/cmd/mdatagen/README.md)
- [Component Stability Guidelines](https://github.com/open-telemetry/opentelemetry-collector#stability-levels)

## Getting Help

- Open an issue in the repository
- Ask in the #otel-collector-dev channel on CNCF Slack
- Tag maintainers in your PR for guidance
