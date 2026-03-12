package deps

import (
	"strings"
	"testing"
)

func TestCheckAll(t *testing.T) {
	statuses := CheckAll()
	if len(statuses) != 3 {
		t.Errorf("expected 3 deps, got %d", len(statuses))
	}

	// Verify names
	names := make(map[string]bool)
	for _, s := range statuses {
		names[s.Name] = true
	}
	if !names["GitHub CLI (gh)"] {
		t.Error("missing gh dep")
	}
	if !names["Git"] {
		t.Error("missing git dep")
	}
	if !names["Claude Code CLI"] {
		t.Error("missing claude dep")
	}
}

func TestCheckGH(t *testing.T) {
	s := checkGH()
	if s.Name != "GitHub CLI (gh)" {
		t.Errorf("unexpected name: %s", s.Name)
	}
	if !s.Required {
		t.Error("gh should be required")
	}
	// gh should be installed in test env (CI or dev)
	if s.Installed && s.Path == "" {
		t.Error("installed but no path")
	}
}

func TestCheckGit(t *testing.T) {
	s := checkGit()
	if s.Name != "Git" {
		t.Errorf("unexpected name: %s", s.Name)
	}
	if !s.Required {
		t.Error("git should be required")
	}
	// Git should always be installed
	if !s.Installed {
		t.Error("git should be installed")
	}
	if !strings.Contains(s.Version, "git version") {
		t.Errorf("unexpected git version: %s", s.Version)
	}
}

func TestCheckClaudeCode(t *testing.T) {
	s := checkClaudeCode()
	if s.Name != "Claude Code CLI" {
		t.Errorf("unexpected name: %s", s.Name)
	}
	if s.Required {
		t.Error("claude should NOT be required")
	}
	// Claude may or may not be installed, just verify no panic
}

func TestCheckRequired(t *testing.T) {
	// Should not error if gh and git are installed
	err := CheckRequired()
	// On most dev machines this passes; on bare CI it might fail
	_ = err
}

func TestPrintStatus(t *testing.T) {
	output := PrintStatus()
	if !strings.Contains(output, "PRFlow Dependencies") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "Git") {
		t.Error("missing Git in output")
	}
}

func TestHasGH(t *testing.T) {
	// Just verify it doesn't panic
	_ = HasGH()
}

func TestHasClaudeCode(t *testing.T) {
	// Just verify it doesn't panic
	_ = HasClaudeCode()
}
