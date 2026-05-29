// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package usecaseextension // import "github.com/newrelic/nrdot-collector-components/extension/usecaseextension"

import (
	"net/http"

	"github.com/newrelic/nrdot-collector-components/extension/usecaseextension/internal/source"
)

type useCaseRoundTripper struct {
	base   http.RoundTripper
	source source.Source
}

func (rt *useCaseRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())

	useCase, err := rt.source.Get(req.Context())
	if err != nil {
		return nil, err
	}

	if useCase != "" {
		ua := req2.Header.Get("User-Agent")
		if ua != "" {
			ua = ua + " " + useCase
		} else {
			ua = useCase
		}
		req2.Header.Set("User-Agent", ua)
	}

	return rt.base.RoundTrip(req2)
}
