// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"sync"
	"time"

	"github.com/newrelic/nrdot-collector-components/receiver/osqueryreceiver/collection"
	"github.com/newrelic/nrdot-collector-components/receiver/osqueryreceiver/statemanager"
	"go.uber.org/zap"
)

type CollectionExecutor struct {
	logger       *zap.Logger
	Collections  []collection.ICollection
	StateManager statemanager.IStateManager
}

func NewCollectionExecutor(logger *zap.Logger, collections []collection.ICollection, tmpDir string) *CollectionExecutor {
	return &CollectionExecutor{
		logger:      logger,
		Collections: collections,
		// Using in-memory state manager for simplicity; will be replaced with persistent one later
		// StateManager: statemanager.GetStateManager("inmemory", logger, ""),
		StateManager: statemanager.GetStateManager("file", logger, tmpDir),
	}
}

func (e *CollectionExecutor) ExecuteAll() map[string]QueryExecution {
	results := make(map[string]QueryExecution)
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		stateMu sync.Mutex
	)
	for _, coll := range e.Collections {
		wg.Add(1)
		go func(coll collection.ICollection) {
			defer wg.Done()

			collectionName := coll.GetName()
			e.logger.Info("Executing collection", zap.String("collection", collectionName))

			query := coll.GetQuery()
			data, err := e.Run(query)
			execResult := QueryExecution{
				Query:      query,
				ExecutedAt: time.Now(),
			}

			if err != nil {
				e.logger.Error("Failed to execute query", zap.String("query", query), zap.Error(err))
				execResult.Error = err
				mu.Lock()
				results[collectionName] = execResult
				mu.Unlock()
				return
			}

			transformed := coll.Unmarshal(data)
			execResult.TransformInto = transformed
			execResult.State = transformed

			previousState := e.getCollectionState(collectionName)
			changedRows, hasChange := computeChanges(previousState, transformed)

			e.logger.Info("Collection execution completed", zap.String("collection", collectionName))

			if !hasChange {
				e.logger.Debug("No state change detected", zap.String("collection", collectionName))
				stateMu.Lock()
				e.updateCollectionState(collectionName, transformed)
				stateMu.Unlock()
				return
			}

			e.logStateChange(collectionName, previousState, transformed, changedRows)
			stateMu.Lock()
			e.updateCollectionState(collectionName, transformed)
			stateMu.Unlock()

			if changedRows == nil {
				return
			}

			execResult.TransformInto = changedRows
			execResult.ResultCount = countRecords(changedRows)

			mu.Lock()
			results[collectionName] = execResult
			mu.Unlock()
		}(coll)
	}
	wg.Wait()
	return results
}

func (e *CollectionExecutor) getCollectionState(collectionName string) any {
	return e.StateManager.Retrieve(collectionName)
}

func (e *CollectionExecutor) updateCollectionState(collectionName string, latest any) {
	e.StateManager.Save(collectionName, latest)
}

func (e *CollectionExecutor) logStateChange(collectionName string, previous, current, changed any) {
	payload := map[string]any{
		"collection":  collectionName,
		"changedRows": changed,
	}
	if previous != nil {
		payload["previousState"] = previous
	}
	if current != nil {
		payload["currentState"] = current
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		e.logger.Debug("State change detected", zap.String("collection", collectionName), zap.Any("changedRows", changed))
		return
	}

	e.logger.Debug("State change detected", zap.ByteString("state_change", encoded))
}

func computeChanges(previous, current any) (any, bool) {
	if current == nil {
		if previous != nil {
			return nil, true
		}
		return nil, false
	}

	currentValue := reflect.ValueOf(current)
	if currentValue.Kind() == reflect.Slice {
		changeSet := reflect.MakeSlice(currentValue.Type(), 0, currentValue.Len())
		previousKeys := make(map[string]struct{})

		if previous != nil {
			previousValue := reflect.ValueOf(previous)
			if previousValue.Kind() == reflect.Slice {
				for i := 0; i < previousValue.Len(); i++ {
					key := comparableValue(previousValue.Index(i).Interface())
					previousKeys[key] = struct{}{}
				}
			}
		}

		for i := 0; i < currentValue.Len(); i++ {
			elem := currentValue.Index(i)
			key := comparableValue(elem.Interface())
			if _, exists := previousKeys[key]; !exists {
				changeSet = reflect.Append(changeSet, elem)
			}
		}

		if changeSet.Len() == 0 {
			return nil, false
		}

		return changeSet.Interface(), true
	}

	if previous != nil && reflect.DeepEqual(previous, current) {
		return nil, false
	}

	return current, true
}

func comparableValue(value any) string {
	if value == nil {
		return ""
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}

	return string(encoded)
}

func countRecords(data any) int {
	if data == nil {
		return 0
	}

	value := reflect.ValueOf(data)
	if value.Kind() == reflect.Slice {
		return value.Len()
	}

	return 1
}

func (e *CollectionExecutor) Run(query string) (any, error) {
	e.logger.Debug("Executing osquery query", zap.String("query", query))
	cmd := exec.Command("osqueryi", "--json", query)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	e.logger.Debug("Osquery query executed successfully", zap.String("query", query))
	var outputData any
	if err := json.Unmarshal(output, &outputData); err != nil {
		e.logger.Error("Failed to unmarshal osquery output", zap.Error(err))
		return nil, err
	}
	e.logger.Debug("Unmarshalled osquery output", zap.Any("output", outputData))
	outputDataSlice, ok := outputData.([]any)
	if !ok {
		e.logger.Error("Failed to convert osquery output to slice", zap.String("query", query))
		return nil, nil
	}
	if len(outputDataSlice) == 0 {
		e.logger.Warn("No results returned from osquery", zap.String("query", query))
		return nil, nil
	}

	// Convert []any to []map[string]any for collections to unmarshal
	resultMaps := make([]map[string]any, 0, len(outputDataSlice))
	for _, item := range outputDataSlice {
		if itemMap, ok := item.(map[string]any); ok {
			resultMaps = append(resultMaps, itemMap)
		}
	}

	// If only one result, return as single map for single-row collections (like system_info)
	// Otherwise return as slice for multi-row collections (like package_info)
	if len(resultMaps) == 1 {
		return resultMaps[0], nil
	}
	return resultMaps, nil
}
