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
