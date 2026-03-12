package ai

import (
	"testing"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"pure json",
			`{"key": "value"}`,
			`{"key": "value"}`,
		},
		{
			"json with prefix",
			`Here is the analysis:\n{"key": "value"}`,
			`{"key": "value"}`,
		},
		{
			"json with suffix",
			`{"key": "value"}\nHope that helps!`,
			`{"key": "value"}`,
		},
		{
			"nested json",
			`{"outer": {"inner": "value"}, "list": [1, 2]}`,
			`{"outer": {"inner": "value"}, "list": [1, 2]}`,
		},
		{
			"no json",
			`plain text with no braces`,
			`plain text with no braces`,
		},
		{
			"json in markdown",
			"```json\n{\"key\": \"value\"}\n```",
			`{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSON(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
	}{
		{"short", 10},
		{"this is a long string", 10},
		{"", 5},
		{"ab", 1},
		{"hello\nworld", 15},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.max)
		if len(result) > tt.max {
			t.Errorf("truncate(%q, %d) len=%d exceeds max", tt.input, tt.max, len(result))
		}
	}
}

func TestAvailable(t *testing.T) {
	// Just verify it doesn't panic
	_ = Available()
}

func TestClaudePath(t *testing.T) {
	path := claudePath()
	// Should return "claude" as fallback even if not installed
	if path == "" {
		t.Error("expected non-empty path")
	}
}
