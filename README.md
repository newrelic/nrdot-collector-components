<a href="https://opensource.newrelic.com/oss-category/#community-project"><picture><source media="(prefers-color-scheme: dark)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/dark/Community_Project.png"><source media="(prefers-color-scheme: light)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Project.png"><img alt="New Relic Open Source community project banner." src="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Project.png"></picture></a>

# New Relic OpenTelemetry Collector Components

[![Build Status](https://img.shields.io/github/actions/workflow/status/newrelic/nrdot-collector-components/build-and-test.yml?branch=main&style=for-the-badge)](https://github.com/newrelic/nrdot-collector-components/actions/workflows/build-and-test.yml?query=branch%3Amain)
[![Go Report Card](https://goreportcard.com/badge/github.com/newrelic/nrdot-collector-components?style=for-the-badge)](https://goreportcard.com/report/github.com/newrelic/nrdot-collector-components)
[![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/newrelic/nrdot-collector-components?include_prereleases&style=for-the-badge)](https://github.com/newrelic/nrdot-collector-components/releases)

OpenTelemetry Collector components created and maintained by New Relic. This repository is a focused fork of [opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib), containing specialized receivers, processors, and infrastructure needed for New Relic's OpenTelemetry integrations.

**Featured Components:**
- **New Relic Oracle Receiver**: Comprehensive Oracle database monitoring with support for CDB/PDB, RAC, query performance monitoring, and 100+ metrics
- **Adaptive Telemetry Processor**: Intelligent metric filtering and adaptive sampling that reduces telemetry costs by up to 70%

This project follows semantic versioning and releases are aligned with upstream OpenTelemetry Collector releases. Components in this repository maintain Beta stability for traces, metrics, and logs unless otherwise noted.

## What's Included

This repository includes production-ready OpenTelemetry components and supporting infrastructure:

### Receivers
- **[New Relic Oracle Receiver](receiver/newrelicoraclereceiver/)**: Comprehensive Oracle database monitoring
  - Supports Oracle 11g, 12c, 19c, 21c, 23c (Standard and Enterprise Edition)
  - Full CDB/PDB and Oracle RAC support
  - Query performance monitoring with execution plans and wait events
  - 100+ metrics across connections, memory, disk I/O, tablespaces, and performance
- **Nop Receiver**: No-operation receiver for testing and development

### Processors
- **[Adaptive Telemetry Processor](processor/adaptivetelemetryprocessor/)**: Intelligent metric filtering and sampling
  - Reduces telemetry costs by 50-70% through dynamic threshold filtering
  - Process-based sampling with full path matching for security
  - Anomaly detection and multi-metric composite scoring
  - Stateful processing with persistent storage

### Exporters
- **Nop Exporter**: No-operation exporter for testing and development

### Infrastructure
- **Internal Utilities**: Common libraries and core internal packages for New Relic functionality
- **Testbed Framework**: Comprehensive end-to-end testing infrastructure for component validation
- **Build Tools**: Utilities for code generation and repository maintenance

For the complete list of included components, see [`versions.yaml`](versions.yaml).

## Components

### New Relic Oracle Receiver

**[Full Documentation](receiver/newrelicoraclereceiver/README.md)** | **Stability: Beta**

A comprehensive OpenTelemetry receiver for monitoring Oracle databases with extensive metric collection and query performance tracking.

**Key Features:**
- **Multi-Architecture Support**: Works with non-CDB, CDB/PDB (12c+), and Oracle RAC configurations
- **Query Performance Monitoring**: Tracks slow queries, execution plans, child cursors, wait events, and blocking queries
- **Comprehensive Metrics**: 100+ metrics including connections, memory (SGA/PGA), disk I/O, tablespaces, transactions, and system statistics
- **Container Database Support**: Full CDB and PDB monitoring with per-container metrics
- **RAC Monitoring**: ASM disk groups, cluster wait events, instance status, and service configuration
- **Flexible Configuration**: Configurable scrapers, collection intervals, and query monitoring thresholds

**Supported Oracle Versions:** 11g, 12c, 19c, 21c, 23c (Standard and Enterprise Edition)

**Quick Start:**
```yaml
receivers:
  newrelicoracledb:
    endpoint: "hostname:1521"
    username: "oracle_user"
    password: "${env:ORACLE_PASSWORD}"
    service: "ORCL"
    collection_interval: 60s
    enable_query_monitoring: true
```

**Example Metrics:**
- `newrelicoracledb.connection.active_sessions`
- `newrelicoracledb.memory.pga_in_use_bytes`
- `newrelicoracledb.disk.reads`
- `newrelicoracledb.slow_queries.avg_elapsed_time`
- `newrelicoracledb.tablespace.space_used_percentage`
- `newrelicoracledb.pdb.cpu_usage_per_second`

---

### Adaptive Telemetry Processor

**[Full Documentation](processor/adaptivetelemetryprocessor/README.md)** | **Stability: Alpha**

An intelligent metric filtering and adaptive sampling processor that dynamically filters low-value telemetry while preserving high-value data and anomalies.

**Key Features:**
- **Cost Reduction**: Reduces telemetry volume by 50-70% while maintaining observability during critical events
- **Dynamic Threshold Adjustment**: Automatically adapts thresholds based on historical baselines and workload patterns
- **Process-Based Filtering**: Monitors specific processes by full path (e.g., `/usr/sbin/nginx`, `/usr/bin/java`)
- **Anomaly Detection**: Detects sudden metric changes and ensures anomalous data is always captured
- **Multi-Metric Composite Scoring**: Combines multiple metrics with configurable weights for holistic health assessment
- **Stateful Processing**: Maintains persistent state across collector restarts with secure storage
- **Low Overhead**: Minimal performance impact (< 2% CPU, 10-50 MB memory)

**Use Cases:**
- Cost optimization by filtering "normal" metrics while preserving anomalies
- Noise reduction in idle or baseline systems
- Selective monitoring of specific applications or services
- Dynamic adaptation to changing workload patterns in cloud environments

**Quick Start:**
```yaml
processors:
  adaptivetelemetry:
    retention_minutes: 30
    include_process_list:
      - "/usr/sbin/nginx"
      - "/usr/bin/java"
    metric_thresholds:
      system.cpu.utilization: 0.05        # 5% threshold
      system.memory.utilization: 0.05
      process.cpu.utilization: 0.05
    enable_dynamic_thresholds: true
    enable_anomaly_detection: true
```

**Filtering Logic:**
- Metrics from monitored processes are always passed
- Metrics exceeding thresholds are passed
- Anomalous metrics (sudden changes) are passed
- Low-value metrics below thresholds are filtered

---

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

## Example Configurations

### Oracle Database Monitoring with Cost Optimization

This example combines the Oracle receiver with the Adaptive Telemetry Processor to monitor Oracle databases while reducing telemetry costs:

```yaml
receivers:
  # Oracle CDB Infrastructure Monitoring
  newrelicoracledb/cdb:
    endpoint: "oraclehost:1521"
    username: "c##monitor"
    password: "${env:ORACLE_PASSWORD}"
    service: "ORCL"
    collection_interval: 60s
    timeout: 45s
    enable_query_monitoring: false  # Focus on infrastructure

  # Oracle PDB Application Monitoring
  newrelicoracledb/pdbs:
    endpoint: "oraclehost:1521"
    username: "monitor"
    password: "${env:ORACLE_PASSWORD}"
    service: "CDB"
    collection_interval: 60s
    enable_query_monitoring: true
    enable_interval_based_averaging: true
    pdb_services: ["ALL"]

  # Host metrics for the Oracle database server
  hostmetrics:
    collection_interval: 60s
    scrapers:
      cpu:
      memory:
      disk:
      filesystem:
      network:
      process:

processors:
  # Filter low-value host metrics while preserving Oracle process metrics
  adaptivetelemetry:
    retention_minutes: 30
    include_process_list:
      - "/u01/app/oracle/product/19.0.0/dbhome_1/bin/oracle"
    metric_thresholds:
      system.cpu.utilization: 0.05
      system.memory.utilization: 0.05
      process.cpu.utilization: 0.05
    enable_dynamic_thresholds: true
    enable_anomaly_detection: true

  # Transform to reduce metadata size
  transform:
    metric_statements:
      - context: metric
        statements:
          - set(metric.description, "")
          - set(metric.unit, "")

  batch:
    timeout: 10s
    send_batch_size: 1024

exporters:
  otlphttp:
    endpoint: https://otlp.nr-data.net
    headers:
      api-key: ${env:NEW_RELIC_LICENSE_KEY}
    compression: gzip

service:
  pipelines:
    # Oracle metrics pipeline
    metrics/oracle:
      receivers: [newrelicoracledb/cdb, newrelicoracledb/pdbs]
      processors: [transform, batch]
      exporters: [otlphttp]

    # Host metrics pipeline with adaptive filtering
    metrics/host:
      receivers: [hostmetrics]
      processors: [adaptivetelemetry, batch]
      exporters: [otlphttp]
```

### Multi-Process Monitoring with Intelligent Filtering

Monitor multiple application processes while filtering out low-value system metrics:

```yaml
receivers:
  hostmetrics:
    collection_interval: 60s
    scrapers:
      processes:
      process:
        metrics:
          process.cpu.utilization:
            enabled: true
          process.memory.usage:
            enabled: true
      cpu:
      memory:
      disk:
      network:

processors:
  adaptivetelemetry:
    retention_minutes: 30
    # Monitor specific application processes
    include_process_list:
      - "/usr/sbin/nginx"
      - "/usr/bin/java"
      - "/usr/bin/postgres"
      - "/opt/app/bin/service"

    # Filter low-value system metrics
    metric_thresholds:
      system.cpu.utilization: 0.10      # 10% CPU threshold
      system.memory.utilization: 0.10
      process.cpu.utilization: 0.05
      process.memory.usage: 104857600   # 100 MB

    # Enable advanced features
    enable_dynamic_thresholds: true
    enable_anomaly_detection: true
    enable_multi_metric: true
    composite_threshold: 0.8

  batch:

exporters:
  otlphttp:
    endpoint: https://otlp.nr-data.net
    headers:
      api-key: ${env:NEW_RELIC_LICENSE_KEY}

service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [adaptivetelemetry, batch]
      exporters: [otlphttp]
```

### Production Oracle with Query Performance to Logs

Advanced configuration that converts high-cardinality query data to logs:

```yaml
receivers:
  newrelicoracledb/pdbs:
    endpoint: "oraclehost:1521"
    username: "monitor"
    password: "${env:ORACLE_PASSWORD}"
    service: "CDB"
    collection_interval: 45s
    enable_query_monitoring: true
    enable_interval_based_averaging: true
    query_monitoring_interval_seconds: 30
    pdb_services: ["ALL"]

processors:
  # Filter execution plans and query details for logs conversion
  filter/exec_plan_include:
    metrics:
      include:
        match_type: strict
        metric_names:
          - newrelicoracledb.execution_plan
          - newrelicoracledb.slow_queries.query_details

  # Exclude execution plans from metrics pipeline
  filter/exec_plan_exclude:
    metrics:
      exclude:
        match_type: strict
        metric_names:
          - newrelicoracledb.execution_plan
          - newrelicoracledb.slow_queries.query_details

  batch:

connectors:
  # Convert metrics to logs
  metricsaslogs:
    include_resource_attributes: true
    include_scope_info: true

exporters:
  otlphttp:
    endpoint: https://otlp.nr-data.net
    headers:
      api-key: ${env:NEW_RELIC_LICENSE_KEY}

service:
  pipelines:
    # Main metrics pipeline
    metrics:
      receivers: [newrelicoracledb/pdbs]
      processors: [filter/exec_plan_exclude, batch]
      exporters: [otlphttp]

    # Metrics-to-logs pipeline for execution plans
    metrics/to_logs:
      receivers: [newrelicoracledb/pdbs]
      processors: [filter/exec_plan_include]
      exporters: [metricsaslogs]

    # Logs pipeline
    logs:
      receivers: [metricsaslogs]
      exporters: [otlphttp]
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
