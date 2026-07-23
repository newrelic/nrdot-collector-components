// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Overrides []Overrides `yaml:"overrides"`
}

type Overrides struct {
	Path    string `yaml:"path"`
	RepoURL string `yaml:"repo_url"`
	Commit  string `yaml:"commit"`
}

func NewConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	for index, override := range cfg.Overrides {
		if override.Path == "" || override.RepoURL == "" || override.Commit == "" {
			return nil, fmt.Errorf("override[%d]: path, repo_url, and commit are all required", index)
		}
	}

	return &cfg, nil
}
