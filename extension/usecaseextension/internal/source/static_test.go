// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticSource(t *testing.T) {
	ss := &StaticSource{Value: "use-case"}
	useCase, err := ts.Get(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, "use-case", useCase)
}
