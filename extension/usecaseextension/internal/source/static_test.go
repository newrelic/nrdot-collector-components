// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticSource(t *testing.T) {
	ss := &StaticSource{Id: "use_case"}
	useCase, err := ss.Get(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, "use-case", useCase)
}
