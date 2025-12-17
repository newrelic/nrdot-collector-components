// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

// Package adaptivetelemetryprocessor implements a processor that intelligently
// filters metrics based on configurable thresholds, dynamic thresholds,
// multi-metric evaluation, and anomaly detection.
package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"
