package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import "time"

// trackedEntity represents a resource entity tracked by the adaptive filter.
type trackedEntity struct {
	Identity      string             `json:"identity"`
	FirstSeen     time.Time          `json:"first_seen"`
	LastExceeded  time.Time          `json:"last_exceeded"` // Used for threshold-based retention (static/dynamic/multi-metric)
	CurrentValues map[string]float64 `json:"current_values"`
	MaxValues     map[string]float64 `json:"max_values"`
	Attributes    map[string]string  `json:"attributes,omitempty"`

	// Anomaly detection fields - uses separate retention tracking
	MetricHistory       map[string][]float64 `json:"metric_history,omitempty"`
	LastAnomalyDetected time.Time            `json:"last_anomaly_detected,omitempty"` // Used for anomaly-based retention (independent)
}
