// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/internal/coreinternal/occonventions/opencensus.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package occonventions // import "github.com/newrelic/nrdot-collector-components/internal/coreinternal/occonventions"

// OTLP attributes to map certain OpenCensus proto fields. These fields don't have
// corresponding fields in OTLP, nor are defined in OTLP semantic conventions.
const (
	AttributeProcessStartTime        = "opencensus.starttime"
	AttributeExporterVersion         = "opencensus.exporterversion"
	AttributeResourceType            = "opencensus.resourcetype"
	AttributeSameProcessAsParentSpan = "opencensus.same_process_as_parent_span"
)
