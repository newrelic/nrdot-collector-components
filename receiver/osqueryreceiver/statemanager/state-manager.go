// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package statemanager

import (
	"go.uber.org/zap"
)

type IStateManager interface {
	Save(collectionName string, data any)
	Retrieve(collectionName string) any
	ComputeDiff([]byte, []byte) []byte
}

type StateManager struct {
	Logger *zap.Logger
}

func GetStateManager(managerType string, logger *zap.Logger, fileLocation string) IStateManager {
	switch managerType {
	case "inmemory":
		return NewInMemoryStateManager(logger)
	case "file":
		return NewFileStateManager(logger, fileLocation)
	default:
		return NewInMemoryStateManager(logger)
	}
}
