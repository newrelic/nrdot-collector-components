// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package osqueryreceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/newrelic/nrdot-collector-components/receiver/osqueryreceiver/cache"
	"github.com/newrelic/nrdot-collector-components/receiver/osqueryreceiver/collection"
	"github.com/newrelic/nrdot-collector-components/receiver/osqueryreceiver/executor"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type OSQueryManager struct {
	// configurations
	extensionsSocket string
	logger           *zap.Logger

	// Executor to run queries for all the collections
	executor *executor.CollectionExecutor

	// cache to store previous results if needed
	cache cache.CacheManager
}

func (m *OSQueryManager) RegisterCollections(config *Config) error {
	// Build query tasks from predefined collections
	for _, collectionName := range config.Collections {
		collection, err := collection.GetCollection(collectionName)
		if err != nil {
			m.logger.Warn("Failed to get collection", zap.String("collection", collectionName), zap.Error(err))
			continue
		}
		if collection == nil {
			m.logger.Warn("Skipping unknown collection", zap.String("collection", collectionName))
			continue
		}
		m.executor.Collections = append(m.executor.Collections, collection)
	}

	// Build query tasks from custom queries
	for i, query := range config.CustomQueries {
		collection := collection.GetCustomCollection(fmt.Sprintf("custom_%d", i), query)
		m.executor.Collections = append(m.executor.Collections, collection)
	}

	return nil
}

func NewOSQueryManager(config *Config, logger *zap.Logger) (*OSQueryManager, error) {
	manager := &OSQueryManager{
		extensionsSocket: config.ExtensionsSocket,
		logger:           logger,
		cache:            cache.NewCacheManager(logger),
		executor:         executor.NewCollectionExecutor(logger, []collection.ICollection{}, config.TmpDir),
	}

	if err := manager.RegisterCollections(config); err != nil {
		logger.Error("Failed to register collections", zap.Error(err))
		return nil, err
	}

	return manager, nil
}

func (m *OSQueryManager) collect(nextConsumer consumer.Logs) error {
	results := m.executor.ExecuteAll()
	m.sendToConsumer(context.Background(), results, nextConsumer)
	return nil
}

// sendToConsumer converts query execution results to OTel logs and sends to consumer
func (m *OSQueryManager) sendToConsumer(ctx context.Context, results map[string]executor.QueryExecution, nextConsumer consumer.Logs) error {
	for collectionName, execution := range results {
		// Skip if there was an error executing the query
		if execution.Error != nil {
			m.logger.Error("Skipping collection due to execution error",
				zap.String("collection", collectionName),
				zap.Error(execution.Error))
			continue
		}

		// Check if TransformInto is a slice - if so, send each item as a separate log record
		var items []any
		val := reflect.ValueOf(execution.TransformInto)
		if val.Kind() == reflect.Slice {
			// Convert any slice type to []any
			items = make([]any, val.Len())
			for i := 0; i < val.Len(); i++ {
				items[i] = val.Index(i).Interface()
			}
		} else {
			// Single item - wrap it in a slice
			items = []any{execution.TransformInto}
		}

		logs := plog.NewLogs()
		resourceLogs := logs.ResourceLogs().AppendEmpty()
		scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
		scopeLogs.Scope().SetName("osqueryreceiver")

		// Create a separate log record for each item
		for _, item := range items {
			logRecord := scopeLogs.LogRecords().AppendEmpty()
			logRecord.SetTimestamp(pcommon.NewTimestampFromTime(execution.ExecutedAt))
			logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(execution.ExecutedAt))

			// Set query and collection attributes
			logRecord.Attributes().PutStr("query", execution.Query)
			logRecord.Attributes().PutStr("collection", collectionName)

			// Set body as JSON from the individual item
			bodyBytes, err := json.Marshal(item)
			if err != nil {
				m.logger.Error("Failed to marshal execution result item",
					zap.String("collection", collectionName),
					zap.Error(err))
				continue
			}
			logRecord.Body().SetStr(string(bodyBytes))
		}

		err := nextConsumer.ConsumeLogs(ctx, logs)
		if err != nil {
			return err
		}
	}
	return nil
}
