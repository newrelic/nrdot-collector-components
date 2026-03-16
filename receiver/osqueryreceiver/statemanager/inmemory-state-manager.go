// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package statemanager

import (
	"sync"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"go.uber.org/zap"
)

type inMemoryStateManager struct {
	StateManager
	state sync.Map // map[collectionName]any
}

func NewInMemoryStateManager(logger *zap.Logger) IStateManager {
	return &inMemoryStateManager{
		StateManager: StateManager{
			Logger: logger,
		},
	}
}

func (ism *inMemoryStateManager) Save(collectionName string, data any) {
	ism.state.Store(collectionName, data)
}

func (ism *inMemoryStateManager) Retrieve(collectionName string) any {
	value, ok := ism.state.Load(collectionName)
	if !ok {
		return nil
	}
	return value
}

func (ism *inMemoryStateManager) ComputeDiff(originalState, newState []byte) []byte {
	patch, err := jsonpatch.CreateMergePatch(originalState, newState)
	if err != nil {
		ism.Logger.Error("Failed to compute state diff", zap.Error(err))
		return nil
	}
	ism.Logger.Debug("Computed state diff", zap.ByteString("diff", patch))
	return patch
}
