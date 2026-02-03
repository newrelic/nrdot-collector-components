package testbed

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

/*
	Sandbox: Dynamically extract component metadata to populate configs
*/

type ComponentMetadata struct {
	Type   string
	Status struct {
		Class string
	}
	Test struct {
		Config map[string]any
	}
}

func NewComponentMetadata(componentPathRelative string) *ComponentMetadata {
	repoRoot, err := getRepoRoot()
	if err != nil {
		panic(err)
	}

	fullPath := path.Join(repoRoot, componentPathRelative, "metadata.yaml")

	yamlBytes, err := os.ReadFile(fullPath)
	if err != nil {
		panic(err)
	}

	var metadata ComponentMetadata
	err = yaml.Unmarshal(yamlBytes, &metadata)
	if err != nil {
		panic(fmt.Errorf("error unmarshaling yaml: %w", err))
	}
	return &metadata
}

func (md *ComponentMetadata) GetFullComponentName() string {
	return fmt.Sprintf("%s%s", md.Type, md.Status.Class)
}

func (md *ComponentMetadata) GetTestConfigBody() (string, error) {
	data := map[string]any{
		md.GetFullComponentName(): md.Test.Config,
	}
	cfg, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshaling metadata condif: %w", err)
	}
	return string(cfg), nil
}

func getRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
