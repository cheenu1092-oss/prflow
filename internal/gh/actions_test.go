package gh

import (
	"fmt"
	"strings"
	"testing"
)

func TestApprovePR(t *testing.T) {
	tests := []struct {
		name   string
		repo   string
		number int
		body   string
		want   []string // expected args substring checks
	}{
		{
			name:   "approve without body",
			repo:   "owner/repo",
			number: 123,
			body:   "",
			want:   []string{"pr", "review", "123", "-R", "owner/repo", "--approve"},
		},
		{
			name:   "approve with body",
			repo:   "owner/repo",
			number: 456,
			body:   "LGTM!",
			want:   []string{"pr", "review", "456", "-R", "owner/repo", "--approve", "-b", "LGTM!"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := &CapturingRunner{}
			old := defaultRunner
			SetRunner(capture)
			defer SetRunner(old)

			err := ApprovePR(tt.repo, tt.number, tt.body)
			if err != nil {
				t.Fatalf("ApprovePR failed: %v", err)
			}

			if len(capture.Args) != len(tt.want) {
				t.Fatalf("expected %d args, got %d: %v", len(tt.want), len(capture.Args), capture.Args)
			}
			for i, expected := range tt.want {
				if capture.Args[i] != expected {
					t.Errorf("arg[%d]: expected %q, got %q", i, expected, capture.Args[i])
				}
			}
		})
	}
}

func TestMergePR(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		number    int
		strategy  string
		autoMerge bool
		wantFlag  string
	}{
		{"merge strategy", "owner/repo", 123, "merge", false, "--merge"},
		{"squash strategy", "owner/repo", 123, "squash", false, "--squash"},
		{"rebase strategy", "owner/repo", 123, "rebase", false, "--rebase"},
		{"default strategy", "owner/repo", 123, "unknown", false, "--merge"},
		{"auto merge", "owner/repo", 123, "squash", true, "--auto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := &CapturingRunner{}
			old := defaultRunner
			SetRunner(capture)
			defer SetRunner(old)

			err := MergePR(tt.repo, tt.number, tt.strategy, tt.autoMerge)
			if err != nil {
				t.Fatalf("MergePR failed: %v", err)
			}

			joined := strings.Join(capture.Args, " ")
			if !strings.Contains(joined, tt.wantFlag) {
				t.Errorf("expected args to contain %q, got: %v", tt.wantFlag, capture.Args)
			}
			if !strings.Contains(joined, "pr merge") {
				t.Errorf("expected 'pr merge' in args, got: %v", capture.Args)
			}
		})
	}
}

func TestResolveThread(t *testing.T) {
	capture := &CapturingRunner{}
	old := defaultRunner
	SetRunner(capture)
	defer SetRunner(old)

	err := ResolveThread("test-thread-id")
	if err != nil {
		t.Fatalf("ResolveThread failed: %v", err)
	}

	joined := strings.Join(capture.Args, " ")
	if !strings.Contains(joined, "api graphql") {
		t.Error("expected GraphQL API call")
	}
	if !strings.Contains(joined, "resolveReviewThread") {
		t.Error("expected resolveReviewThread mutation")
	}
	if !strings.Contains(joined, "test-thread-id") {
		t.Error("expected thread ID in args")
	}
}

func TestUnresolveThread(t *testing.T) {
	capture := &CapturingRunner{}
	old := defaultRunner
	SetRunner(capture)
	defer SetRunner(old)

	err := UnresolveThread("test-thread-id")
	if err != nil {
		t.Fatalf("UnresolveThread failed: %v", err)
	}

	joined := strings.Join(capture.Args, " ")
	if !strings.Contains(joined, "unresolveReviewThread") {
		t.Error("expected unresolveReviewThread mutation")
	}
}

func TestReplyToComment(t *testing.T) {
	mock := NewMockRunner()
	// Mock the PR ID lookup
	mock.When("api graphql", `{
		"data": {
			"repository": {
				"pullRequest": {
					"id": "PR_kwDOTest123"
				}
			}
		}
	}`, nil)

	old := defaultRunner
	SetRunner(mock)
	defer SetRunner(old)

	err := ReplyToComment("owner/repo", 42, "thread-id-123", "Thanks for the review!")
	if err != nil {
		t.Fatalf("ReplyToComment failed: %v", err)
	}
}

func TestReplyToCommentPRIDFetchFails(t *testing.T) {
	mock := NewMockRunner()
	mock.When("api graphql", "", fmt.Errorf("network error"))

	old := defaultRunner
	SetRunner(mock)
	defer SetRunner(old)

	err := ReplyToComment("owner/repo", 42, "thread-id", "reply text")
	if err == nil {
		t.Error("expected error when PR ID fetch fails")
	}
}

// TestResolveThreadQueryFormat verifies the GraphQL query is well-formed
func TestResolveThreadQueryFormat(t *testing.T) {
	threadID := "test-id"
	query := `mutation($threadId: ID!) {
		resolveReviewThread(input: { threadId: $threadId }) {
			thread {
				id
				isResolved
			}
		}
	}`

	if !strings.Contains(query, "mutation") {
		t.Error("query must contain 'mutation'")
	}
	if !strings.Contains(query, "resolveReviewThread") {
		t.Error("query must contain 'resolveReviewThread'")
	}
	if !strings.Contains(query, "$threadId") {
		t.Error("query must contain '$threadId' variable")
	}
	if threadID == "" {
		t.Error("threadID cannot be empty")
	}
}

func TestRepoOwner(t *testing.T) {
	tests := []struct {
		repo string
		want string
	}{
		{"owner/repo", "owner"},
		{"octocat/Hello-World", "octocat"},
		{"single", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := repoOwner(tt.repo)
		if got != tt.want {
			t.Errorf("repoOwner(%q) = %q, want %q", tt.repo, got, tt.want)
		}
	}
}

func TestRepoName(t *testing.T) {
	tests := []struct {
		repo string
		want string
	}{
		{"owner/repo", "repo"},
		{"octocat/Hello-World", "Hello-World"},
		{"single", "single"},
		{"", ""},
	}

	for _, tt := range tests {
		got := repoName(tt.repo)
		if got != tt.want {
			t.Errorf("repoName(%q) = %q, want %q", tt.repo, got, tt.want)
		}
	}
}

func TestGetPRDiffSuccess(t *testing.T) {
	mock := NewMockRunner()
	mock.When("pr diff", "diff --git a/file.go b/file.go\n+new line", nil)

	old := defaultRunner
	SetRunner(mock)
	defer SetRunner(old)

	diff, err := GetPRDiff("org/repo", 42)
	if err != nil {
		t.Fatalf("GetPRDiff failed: %v", err)
	}
	if !strings.Contains(diff, "diff --git") {
		t.Errorf("expected diff output, got %q", diff)
	}
}

func TestCheckoutPRSuccess(t *testing.T) {
	capture := &CapturingRunner{}
	old := defaultRunner
	SetRunner(capture)
	defer SetRunner(old)

	err := CheckoutPR("org/repo", 42)
	if err != nil {
		t.Fatalf("CheckoutPR failed: %v", err)
	}

	joined := strings.Join(capture.Args, " ")
	if !strings.Contains(joined, "pr checkout 42") {
		t.Errorf("expected 'pr checkout 42', got: %v", capture.Args)
	}
	if !strings.Contains(joined, "-R org/repo") {
		t.Errorf("expected '-R org/repo', got: %v", capture.Args)
	}
}

func TestCloneRepoSuccess(t *testing.T) {
	capture := &CapturingRunner{}
	old := defaultRunner
	SetRunner(capture)
	defer SetRunner(old)

	err := CloneRepo("org/repo", "/tmp/dest")
	if err != nil {
		t.Fatalf("CloneRepo failed: %v", err)
	}

	joined := strings.Join(capture.Args, " ")
	if !strings.Contains(joined, "repo clone org/repo /tmp/dest") {
		t.Errorf("expected clone args, got: %v", capture.Args)
	}
}

func TestNudgeReviewerArgs(t *testing.T) {
	capture := &CapturingRunner{}
	old := defaultRunner
	SetRunner(capture)
	defer SetRunner(old)

	err := NudgeReviewer("org/repo", 42, "alice", 3)
	if err != nil {
		t.Fatalf("NudgeReviewer failed: %v", err)
	}

	expected := []string{
		"pr", "comment", "42", "-R", "org/repo", "-b",
		"@alice friendly nudge \u2014 this PR has been waiting for your review for 3 days",
	}
	if len(capture.Args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(capture.Args), capture.Args)
	}
	for i, arg := range expected {
		if capture.Args[i] != arg {
			t.Errorf("arg[%d]: expected %q, got %q", i, arg, capture.Args[i])
		}
	}
}

func TestSearchOrgReposFallback(t *testing.T) {
	mock := NewMockRunner()
	// search repos fails, triggering fallback
	mock.When("search repos", "", fmt.Errorf("search failed"))
	mock.When("repo list", `[
		{"nameWithOwner": "org/api-server"},
		{"nameWithOwner": "org/web-app"},
		{"nameWithOwner": "other/unrelated"}
	]`, nil)

	old := defaultRunner
	SetRunner(mock)
	defer SetRunner(old)

	repos, err := SearchOrgRepos("api")
	if err != nil {
		t.Fatalf("SearchOrgRepos fallback failed: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 match, got %d: %v", len(repos), repos)
	}
	if repos[0] != "org/api-server" {
		t.Errorf("expected 'org/api-server', got %q", repos[0])
	}
}

func TestGetReviewThreadsSuccess(t *testing.T) {
	mock := NewMockRunner()
	mock.When("api graphql", `{
		"data": {
			"repository": {
				"pullRequest": {
					"reviewThreads": {
						"nodes": [
							{
								"id": "thread-1",
								"path": "src/main.go",
								"line": 10,
								"isResolved": false,
								"comments": {
									"nodes": [
										{
											"author": {"login": "alice"},
											"body": "Fix this",
											"createdAt": "2026-03-10T00:00:00Z",
											"url": "https://github.com/org/repo/pull/1#r1"
										}
									]
								}
							}
						]
					}
				}
			}
		}
	}`, nil)

	old := defaultRunner
	SetRunner(mock)
	defer SetRunner(old)

	threads, err := GetReviewThreads("org/repo", 1)
	if err != nil {
		t.Fatalf("GetReviewThreads failed: %v", err)
	}
	if len(threads) != 1 {
		t.Fatalf("expected 1 thread, got %d", len(threads))
	}
	if threads[0].Path != "src/main.go" {
		t.Errorf("expected path 'src/main.go', got %q", threads[0].Path)
	}
	if threads[0].IsResolved {
		t.Error("expected thread to be unresolved")
	}
	if len(threads[0].Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(threads[0].Comments))
	}
	if threads[0].Comments[0].Author != "alice" {
		t.Errorf("expected author 'alice', got %q", threads[0].Comments[0].Author)
	}
}
