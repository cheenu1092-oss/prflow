package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/nagarjun226/prflow/internal/cache"
	"github.com/nagarjun226/prflow/internal/config"
	"github.com/nagarjun226/prflow/internal/gh"
)

func TestPrintUsage(t *testing.T) {
	printUsage()
}

func TestPrintUsageContainsCommands(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printUsage()

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	output := string(out)

	commands := []string{"watch", "setup", "sync", "ls", "config", "open", "doctor", "version"}
	for _, cmd := range commands {
		if !strings.Contains(output, cmd) {
			t.Errorf("expected %q in usage output", cmd)
		}
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
		{"case insensitive", &gh.PR{Author: gh.Author{Login: "ME"}, ReviewDecision: "APPROVED"}, "me", "do_now"},
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
		input   string
		want    string
		wantErr bool
	}{
		{"https://github.com/org/repo.git", "org/repo", false},
		{"https://github.com/org/repo", "org/repo", false},
		{"git@github.com:org/repo.git", "org/repo", false},
		{"git@github.com:org/repo", "org/repo", false},
		{"notaurl", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseRepoFromURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.input)
				}
				return
			}
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

func TestVersionStringCommitOnly(t *testing.T) {
	oldV, oldC, oldD := Version, Commit, Date
	defer func() { Version, Commit, Date = oldV, oldC, oldD }()

	Version, Commit, Date = "2.0.0", "deadbeef", ""
	got := VersionString()
	expected := "prflow v2.0.0 (commit deadbeef)"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestVersionStringDateOnly(t *testing.T) {
	oldV, oldC, oldD := Version, Commit, Date
	defer func() { Version, Commit, Date = oldV, oldC, oldD }()

	Version, Commit, Date = "2.0.0", "", "2026-04-01"
	got := VersionString()
	expected := "prflow v2.0.0 (built 2026-04-01)"
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
	if !hasFlag([]string{"--verbose", "--json", "--debug"}, "--json") {
		t.Error("expected true for --json in middle")
	}
}

func TestToJSONPRs(t *testing.T) {
	input := []cache.CachedPR{
		{PR: gh.PR{Number: 1, Title: "First PR", ReviewDecision: "APPROVED", Mergeable: "MERGEABLE", UpdatedAt: "2026-03-10T00:00:00Z"}, Repo: "org/repo1"},
		{PR: gh.PR{Number: 2, Title: "Second PR", ReviewDecision: "CHANGES_REQUESTED", UpdatedAt: "2026-03-11T00:00:00Z"}, Repo: "org/repo2"},
	}

	result := toJSONPRs(input)
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	if result[0].Repo != "org/repo1" {
		t.Errorf("expected repo 'org/repo1', got %q", result[0].Repo)
	}
	if result[0].Number != 1 {
		t.Errorf("expected number 1, got %d", result[0].Number)
	}
	if result[1].ReviewDecision != "CHANGES_REQUESTED" {
		t.Errorf("expected CHANGES_REQUESTED, got %q", result[1].ReviewDecision)
	}
}

func TestToJSONPRsEmpty(t *testing.T) {
	result := toJSONPRs(nil)
	if len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}

func TestRunListToNoConfig(t *testing.T) {
	config.SetPathOverride("/nonexistent/config.yaml")
	defer config.SetPathOverride("")

	var buf bytes.Buffer
	err := runListTo(&buf, false)
	if err == nil {
		t.Error("expected error when no config")
	}
}

func TestRunListToJSONNoConfig(t *testing.T) {
	config.SetPathOverride("/nonexistent/config.yaml")
	defer config.SetPathOverride("")

	var buf bytes.Buffer
	err := runListTo(&buf, true)
	if err == nil {
		t.Error("expected error when no config")
	}
}

func TestRunOpenNoArg(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Override repoFromRemote to return a fake repo
	oldRemote := repoFromRemote
	repoFromRemote = func() (string, error) {
		return "org/test-repo", nil
	}
	defer func() { repoFromRemote = oldRemote }()

	os.Args = []string{"prflow", "open"}
	err := runOpen()
	// Should succeed (opens browser, which may fail in CI but that's ok)
	_ = err
}

func TestRunOpenWithRepoHash(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"prflow", "open", "org/repo#42"}
	err := runOpen()
	// Will try to open browser - we just verify it doesn't panic
	_ = err
}

func TestRunOpenWithRepoOnly(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"prflow", "open", "org/repo"}
	err := runOpen()
	_ = err
}

func TestRunOpenInvalidArg(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"prflow", "open", "badarg"}
	err := runOpen()
	if err == nil {
		t.Error("expected error for invalid arg")
	}
}
