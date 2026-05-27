// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package source // import "github.com/newrelic/nrdot-collector-components/extension/usecaseextension/internal/source"

import "context"

type Source interface {
	Get(context.Context) (string, error)
}
