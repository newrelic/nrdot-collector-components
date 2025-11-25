// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"go.uber.org/zap"

	"github.com/newrelic/nrdot-plus-collector-components/cmd/opampsupervisor/supervisor/config"
)

func NewLogger(cfg config.Logs) (*zap.Logger, error) {
	zapCfg := zap.NewProductionConfig()

	zapCfg.Level = zap.NewAtomicLevelAt(cfg.Level)
	zapCfg.OutputPaths = cfg.OutputPaths

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}
	return logger, nil
}
