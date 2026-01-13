// Copyright New Relic, Inc. All rights reserved.
// New Relic Software License

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"sync"
	"time"

	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

// processorImp is the main implementation of the adaptive telemetry processor.
// This file now only contains the core structure definition. All implementation
// details have been moved to specialized files.
type processorImp struct {
	logger       *zap.Logger
	config       *Config
	nextConsumer consumer.Metrics

	trackedEntities    map[string]*trackedEntity
	mu                 sync.RWMutex // protects trackedEntities & dynamicCustomThresholds
	storage            EntityStateStorage
	lastPersistenceOp  time.Time
	persistenceEnabled bool

	lastThresholdUpdate      time.Time // Separate: tracks when dynamic thresholds were last updated
	lastInfoLogTime          time.Time // Tracks when we last logged at INFO level
	dynamicThresholdsEnabled bool
	multiMetricEnabled       bool

	// Dynamic thresholds for metrics (including cpu/memory if configured)
	dynamicCustomThresholds map[string]float64
	// Note: Anomaly detection uses LastAnomalyDetected in trackedEntity (separate timestamp)
}
