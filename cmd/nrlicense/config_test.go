// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewConfigFromFile(t *testing.T) {
	testCases := []struct {
		name        string
		config      *Config
		expectedErr string
	}{
		{
			name:        "empty",
			config:      &Config{Overrides: []Overrides{}},
			expectedErr: "Error: Config must have at least one override",
		},
		{
			name: "valid config",
			config: &Config{
				Overrides: []Overrides{{
					Path:    "some/forked/dir",
					RepoURL: "https://github.com/open-telemetry/opentelemetry-collector-contrib",
					Commit:  "abc1def2ghi3jkl4mno5pqr6stu7vwx8yz901234",
				}},
			},
		},
		{
			name: "missing path",
			config: &Config{
				Overrides: []Overrides{{
					RepoURL: "https://github.com/open-telemetry/opentelemetry-collector-contrib",
					Commit:  "abc1def2ghi3jkl4mno5pqr6stu7vwx8yz901234",
				}},
			},
			expectedErr: "Error: path, repo_url, and commit are all required",
		},
		{
			name: "missing repo_url",
			config: &Config{
				Overrides: []Overrides{{
					Path:   "some/forked/dir",
					Commit: "abc1def2ghi3jkl4mno5pqr6stu7vwx8yz901234",
				}},
			},
			expectedErr: "Error: path, repo_url, and commit are all required",
		},
		{
			name: "missing commit",
			config: &Config{
				Overrides: []Overrides{{
					Path:    "some/forked/dir",
					RepoURL: "https://github.com/open-telemetry/opentelemetry-collector-contrib",
				}},
			},
			expectedErr: "Error: path, repo_url, and commit are all required",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			data, err := yaml.Marshal(testCase.config)
			if err != nil {
				t.Fatalf("marshaling config: %v", err)
			}

			tmpFile, err := os.CreateTemp(t.TempDir(), "*.yaml")
			if err != nil {
				t.Fatalf("creating temp file: %v", err)
			}
			if _, err := tmpFile.Write(data); err != nil {
				t.Fatalf("writing temp file: %v", err)
			}
			tmpFile.Close()

			config, err := NewConfigFromFile(tmpFile.Name())

			if testCase.expectedErr != "" {
				require.EqualError(t, err, testCase.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.config, config)
		})
	}
}
