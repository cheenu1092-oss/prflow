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
	Version = "1.2.3"
	Commit = "abc1234"
	Date = "2026-03-30"
	got := VersionString()
	expected := "prflow v1.2.3 (commit abc1234, built 2026-03-30)"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestVersionStringPartial(t *testing.T) {
	oldV, oldC, oldD := Version, Commit, Date
	defer func() { Version, Commit, Date = oldV, oldC, oldD }()
	Version = "0.2.0"
	Commit = "def5678"
	Date = ""
	got := VersionString()
	if !strings.Contains(got, "commit def5678") {
		t.Errorf("expected commit info in %q", got)
	}
	if strings.Contains(got, "built") {
		t.Errorf("should not contain 'built' when date is empty: %q", got)
	}
}
