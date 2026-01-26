<a href="https://opensource.newrelic.com/oss-category/#community-project"><picture><source media="(prefers-color-scheme: dark)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/dark/Community_Project.png"><source media="(prefers-color-scheme: light)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Project.png"><img alt="New Relic Open Source community project banner." src="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Project.png"></picture></a>

# New Relic OpenTelemetry Collector Components

[![Build Status](https://img.shields.io/github/actions/workflow/status/newrelic/nrdot-collector-components/build-and-test.yml?branch=main&style=for-the-badge)](https://github.com/newrelic/nrdot-collector-components/actions/workflows/build-and-test.yml?query=branch%3Amain)
[![Go Report Card](https://goreportcard.com/badge/github.com/newrelic/nrdot-collector-components?style=for-the-badge)](https://goreportcard.com/report/github.com/newrelic/nrdot-collector-components)
[![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/newrelic/nrdot-collector-components?include_prereleases&style=for-the-badge)](https://github.com/newrelic/nrdot-collector-components/releases)

OpenTelemetry Collector components created and maintained by New Relic. This repository is a focused fork of [opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib), containing only the core components and infrastructure needed for New Relic's OpenTelemetry integrations.

This project follows semantic versioning and releases are aligned with upstream OpenTelemetry Collector releases. Components in this repository maintain Beta stability for traces, metrics, and logs unless otherwise noted.

## What's Included

This fork includes:

- **Core Components**: Essential receivers and exporters (nop receiver, nop exporter)
- **Internal Utilities**: Common libraries and core internal packages for New Relic functionality
- **Testbed Framework**: Comprehensive end-to-end testing infrastructure for component validation
- **Build Tools**: Utilities for code generation and repository maintenance

For the complete list of included components, see [`versions.yaml`](versions.yaml).

## Installation

### Prerequisites

- Go 1.22 or later
- Make

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/newrelic/nrdot-collector-components.git
cd nrdot-collector-components

# Install build tools
make install-tools

# Build all components
make build

# Run tests
make gotest
```

## Getting Started

### Pre-built Distributions

If you're looking for pre-built OpenTelemetry Collector distributions that use these components, see the [nrdot-collector-releases](https://github.com/newrelic/nrdot-collector-releases) repository.

### Building a Custom Collector

Use the [OpenTelemetry Collector Builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder) to create a custom collector with components from this repository:

```bash
# Install the builder
go install go.opentelemetry.io/collector/cmd/builder@latest

# Build using the provided configuration
cd cmd/otelcontribcol
builder --config builder-config.yaml
```

### Testing Components

This repository includes a comprehensive testbed for end-to-end component testing:

```bash
# Run all testbed tests
make e2e-test

# Run specific test suite
TESTS_DIR=correctnesstests/metrics make e2e-test

# Run testbed tests directly
cd testbed
make run-tests
```

For guidance on writing testbed tests for new components, see [`testbed/examples/processor_example_test.go`](testbed/examples/processor_example_test.go).

## Usage

### Adding Components

Components are tracked in [`versions.yaml`](versions.yaml). To add a new component:

1. Create your component following the [contribution guidelines](CONTRIBUTING.md)
2. Add the component to `versions.yaml` in the appropriate module set
3. Update [`cmd/otelcontribcol/builder-config.yaml`](cmd/otelcontribcol/builder-config.yaml)
4. Run `make crosslink` to update intra-repository dependencies
5. Run `make gotidy` to update Go module dependencies
6. Run `make addlicense` to generate license headers and third-party notices

### Component Stability

Each component has stability levels for each signal type (traces, metrics, logs):

- **Beta**: Ready for production use with backwards compatibility guarantees
- **Alpha**: Ready for testing, breaking changes may occur
- **Development**: Experimental, not recommended for production

See component README files for specific stability information.

## Building

### Build Commands

```bash
# Build all components
make build

# Build for specific OS/architecture
make GOOS=linux GOARCH=amd64 build

# Generate code
make generate

# Lint code
make lint

# Run all checks (lint + test)
make checks
```

### Cross-compilation

```bash
# Test cross-compilation for all supported platforms
make cross-compile
```

## Testing

### Unit Tests

```bash
# Run all unit tests
make gotest

# Run tests for specific component
cd exporter/nopexporter
make test

# Run tests with coverage
make gotest-with-cover
```

### Integration Tests

```bash
# Run correctness tests
cd testbed
make run-correctness-tests

# Run stability tests
make run-stability-tests
```

### End-to-End Tests

```bash
# Run full e2e test suite
make e2e-test

# Run specific test category
TESTS_DIR=correctnesstests/traces make e2e-test
```

## Support

New Relic hosts and moderates an online forum where you can interact with New Relic employees as well as other customers to get help and share best practices. Like all official New Relic open source projects, there's a related Community topic in the New Relic Explorers Hub.

- [New Relic Documentation](https://docs.newrelic.com/)
- [New Relic Community Forum](https://forum.newrelic.com/)
- [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/)

## Contribute

We encourage your contributions to improve New Relic OpenTelemetry Collector Components! Keep in mind that when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.

If you have any questions, or to execute our corporate CLA (which is required if your contribution is on behalf of a company), drop us an email at opensource@newrelic.com.

**A note about vulnerabilities**

As noted in our [security policy](../../security/policy), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [our bug bounty program](https://docs.newrelic.com/docs/security/security-privacy/information-security/report-security-vulnerabilities/).

If you would like to contribute to this project, review [these guidelines](./CONTRIBUTING.md).

To all contributors, we thank you! Without your contribution, this project would not be what it is today.

## License

New Relic OpenTelemetry Collector Components is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License.

This project uses source code from the [OpenTelemetry Collector Contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib) project, which is also licensed under Apache 2.0.
