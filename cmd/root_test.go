package cmd

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cheenu1092-oss/prflow/internal/config"
	"github.com/cheenu1092-oss/prflow/internal/gh"
)

func TestPrintUsage(t *testing.T) {
	printUsage()
}

func TestPrintUsageContainsWatch(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printUsage()

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "watch") {
		t.Error("expected 'watch' in usage output")
	}
}

func TestRunConfig(t *testing.T) {
	err := runConfig()
	if err != nil {
		t.Errorf("runConfig should not error: %v", err)
	}
}

func TestRunSyncNoConfig(t *testing.T) {
	config.SetPathOverride("/nonexistent/config.yaml")
	defer config.SetPathOverride("")
	err := runSync()
	if err == nil {
		t.Error("runSync should error without config")
	}
}

func TestClassifyPR(t *testing.T) {
	tests := []struct {
		name     string
		pr       *gh.PR
		username string
		want     string
	}{
		{"changes requested", &gh.PR{Author: gh.Author{Login: "me"}, ReviewDecision: "CHANGES_REQUESTED"}, "me", "do_now"},
		{"approved", &gh.PR{Author: gh.Author{Login: "me"}, ReviewDecision: "APPROVED"}, "me", "do_now"},
		{"conflicting", &gh.PR{Author: gh.Author{Login: "me"}, Mergeable: "CONFLICTING"}, "me", "do_now"},
		{"waiting", &gh.PR{Author: gh.Author{Login: "me"}}, "me", "waiting"},
		{"review", &gh.PR{Author: gh.Author{Login: "other"}}, "me", "review"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyPR(tt.pr, tt.username)
			if got != tt.want {
				t.Errorf("classifyPR() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseOpenArgs(t *testing.T) {
	tests := []struct {
		input   string
		repo    string
		number  int
		wantErr bool
	}{
		{input: "org/repo#42", repo: "org/repo", number: 42},
		{input: "#42", repo: "", number: 42},
		{input: "org/repo", repo: "org/repo", number: 0},
		{input: "", repo: "", number: 0},
		{input: "#", wantErr: true},
		{input: "#abc", wantErr: true},
		{input: "#0", wantErr: true},
		{input: "#-5", wantErr: true},
		{input: "badarg", wantErr: true},
	}
	for _, tt := range tests {
		t.Run("input="+tt.input, func(t *testing.T) {
			got, err := parseOpenArgs(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseOpenArgs(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOpenArgs(%q) unexpected error: %v", tt.input, err)
			}
			if got.Repo != tt.repo {
				t.Errorf("Repo = %q, want %q", got.Repo, tt.repo)
			}
			if got.Number != tt.number {
				t.Errorf("Number = %d, want %d", got.Number, tt.number)
			}
		})
	}
}

func TestParseRepoFromURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/org/repo.git", "org/repo"},
		{"https://github.com/org/repo", "org/repo"},
		{"git@github.com:org/repo.git", "org/repo"},
		{"git@github.com:org/repo", "org/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseRepoFromURL(tt.input)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionStringDefault(t *testing.T) {
	oldV, oldC, oldD := Version, Commit, Date
	defer func() { Version, Commit, Date = oldV, oldC, oldD }()

	Version, Commit, Date = "", "", ""
	got := VersionString()
	if got != "prflow v0.1.0" {
		t.Errorf("expected 'prflow v0.1.0', got %q", got)
	}
}

func TestVersionStringFull(t *testing.T) {
	oldV, oldC, oldD := Version, Commit, Date
	defer func() { Version, Commit, Date = oldV, oldC, oldD }()

	Version, Commit, Date = "1.2.3", "abc1234", "2026-03-30"
	got := VersionString()
	expected := "prflow v1.2.3 (commit abc1234, built 2026-03-30)"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestHasFlag(t *testing.T) {
	if !hasFlag([]string{"--json"}, "--json") {
		t.Error("expected true for --json")
	}
	if hasFlag([]string{"--verbose"}, "--json") {
		t.Error("expected false for missing --json")
	}
	if hasFlag(nil, "--json") {
		t.Error("expected false for nil args")
	}
}
