// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestType(t *testing.T) {
	assert.Equal(t, "newrelicsqlserver", Type.String())
}

func TestScopeName(t *testing.T) {
	assert.Equal(t, "github.com/newrelic/nrdot-collector-components/receiver/newrelicsqlserverreceiver", ScopeName)
}
