package main

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGoreleaserYAMLValid(t *testing.T) {
	data, err := os.ReadFile(".goreleaser.yaml")
	if err != nil {
		t.Fatalf("failed to read .goreleaser.yaml: %v", err)
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf(".goreleaser.yaml is not valid YAML: %v", err)
	}

	// Check required top-level keys
	for _, key := range []string{"version", "builds", "archives", "checksum"} {
		if _, ok := cfg[key]; !ok {
			t.Errorf(".goreleaser.yaml missing required key: %s", key)
		}
	}

	// Verify project name
	if name, ok := cfg["project_name"]; !ok || name != "prflow" {
		t.Errorf("expected project_name=prflow, got %v", cfg["project_name"])
	}
}
