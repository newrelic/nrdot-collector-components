// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package osqueryreceiver

import (
	"context"
	"time"

	"github.com/newrelic/nrdot-collector-components/receiver/osqueryreceiver/executor"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

type osqueryReceiver struct {
	host         component.Host
	cancel       context.CancelFunc
	logger       *zap.Logger
	nextConsumer consumer.Logs
	config       *Config
}

// QueryResponse contains the execution results
type QueryResponse struct {
	RawResults []executor.QueryResult // Raw results from osquery
	Structured any   // Structured results if schema was used, nil otherwise
	Error      error
}

func (o osqueryReceiver) Start(ctx context.Context, host component.Host) error {
	o.host = host
	ctx, o.cancel = context.WithCancel(ctx)

	interval, _ := time.ParseDuration(o.config.CollectionInterval)

	osQueryManager, err := NewOSQueryManager(o.config, o.logger)
	if err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				o.logger.Info("Starting collection")
				osQueryManager.collect(o.nextConsumer)
			case <-ctx.Done():
				o.logger.Info("Shutting down osquery receiver collection")
				return
			}
		}
	}()

	return nil
}

func (o osqueryReceiver) Shutdown(ctx context.Context) error {
	if o.cancel != nil {
		o.cancel()
	}
	return nil
}
