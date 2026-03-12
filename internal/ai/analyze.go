package ai

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cheenu1092-oss/prflow/internal/deps"
	"github.com/cheenu1092-oss/prflow/internal/gh"
)

// Available returns whether AI features can be used
func Available() bool {
	return deps.HasClaudeCode()
}

// claudePath returns the path to the claude CLI
func claudePath() string {
	s := deps.CheckAll()
	for _, d := range s {
		if d.Name == "Claude Code CLI" && d.Installed {
			return d.Path
		}
	}
	return "claude"
}

// PRAnalysis is the AI's assessment of a PR
type PRAnalysis struct {
	Summary        string   `json:"summary"`         // 1-2 sentence summary of what the PR does
	ActionNeeded   string   `json:"action_needed"`   // what the user should do next
	ReviewSummary  string   `json:"review_summary"`  // summary of review feedback
	RiskLevel      string   `json:"risk_level"`      // low/medium/high
	SuggestedFixes []string `json:"suggested_fixes"`  // concrete suggestions
	BlockedBy      string   `json:"blocked_by"`      // who/what is blocking
}

// ThreadAnalysis is the AI's assessment of a review thread
type ThreadAnalysis struct {
	Intent     string `json:"intent"`      // what the reviewer is asking
	Complexity string `json:"complexity"`  // trivial/moderate/significant
	Suggestion string `json:"suggestion"`  // suggested approach to address
	DraftReply string `json:"draft_reply"` // draft response to post
}

// AnalyzePR uses Claude Code to analyze a PR's diff, comments, and review threads
func AnalyzePR(repo string, number int, repoPath string) (*PRAnalysis, error) {
	if !Available() {
		return nil, fmt.Errorf("Claude Code not available")
	}

	// Get PR diff
	diff, _ := gh.GetPRDiff(repo, number)
	if diff == "" {
		diff = "(diff unavailable)"
	}

	// Truncate diff if too large
	if len(diff) > 8000 {
		diff = diff[:8000] + "\n...(truncated)"
	}

	// Get review threads for context
	threads, _ := gh.GetReviewThreads(repo, number)
	threadSummary := ""
	for _, t := range threads {
		if t.IsResolved {
			continue
		}
		if len(t.Comments) > 0 {
			last := t.Comments[len(t.Comments)-1]
			threadSummary += fmt.Sprintf("- %s:%d — @%s: %s\n", t.Path, t.Line, last.Author, truncate(last.Body, 200))
		}
	}

	prompt := fmt.Sprintf(`Analyze this GitHub PR and respond with JSON only.

PR: %s#%d

Diff (truncated):
%s

Open review threads:
%s

Respond with this exact JSON structure:
{
  "summary": "1-2 sentence summary of what this PR changes",
  "action_needed": "specific next action for the PR author",
  "review_summary": "summary of reviewer feedback",
  "risk_level": "low|medium|high",
  "suggested_fixes": ["list", "of", "concrete", "suggestions"],
  "blocked_by": "who or what is blocking this PR, or empty"
}`, repo, number, diff, threadSummary)

	result, err := runClaude(prompt, repoPath)
	if err != nil {
		return nil, err
	}

	var analysis PRAnalysis
	if err := json.Unmarshal([]byte(extractJSON(result)), &analysis); err != nil {
		// If JSON parsing fails, create a basic analysis from the raw text
		analysis = PRAnalysis{
			Summary:      truncate(result, 200),
			ActionNeeded: "Review AI analysis output manually",
		}
	}

	return &analysis, nil
}

// AnalyzeThread uses Claude Code to analyze a specific review thread and suggest a fix
func AnalyzeThread(repo string, number int, thread gh.ReviewThread, repoPath string) (*ThreadAnalysis, error) {
	if !Available() {
		return nil, fmt.Errorf("Claude Code not available")
	}

	// Get the diff for context
	diff, _ := gh.GetPRDiff(repo, number)
	if len(diff) > 6000 {
		diff = diff[:6000] + "\n...(truncated)"
	}

	// Build comment chain
	var commentChain string
	for _, c := range thread.Comments {
		commentChain += fmt.Sprintf("@%s: %s\n---\n", c.Author, c.Body)
	}

	prompt := fmt.Sprintf(`Analyze this GitHub PR review thread and respond with JSON only.

PR: %s#%d
File: %s:%d

Comment thread:
%s

Relevant diff context:
%s

Respond with this exact JSON structure:
{
  "intent": "what the reviewer is asking for in plain English",
  "complexity": "trivial|moderate|significant",
  "suggestion": "specific approach to address the feedback",
  "draft_reply": "draft reply to post (professional, concise)"
}`, repo, number, thread.Path, thread.Line, commentChain, diff)

	result, err := runClaude(prompt, repoPath)
	if err != nil {
		return nil, err
	}

	var analysis ThreadAnalysis
	if err := json.Unmarshal([]byte(extractJSON(result)), &analysis); err != nil {
		analysis = ThreadAnalysis{
			Intent:     truncate(result, 200),
			Complexity: "unknown",
		}
	}

	return &analysis, nil
}

// GenerateFix uses Claude Code to generate a code fix for a review thread
func GenerateFix(repo string, number int, thread gh.ReviewThread, repoPath string) (string, error) {
	if !Available() {
		return "", fmt.Errorf("Claude Code not available")
	}

	if repoPath == "" {
		return "", fmt.Errorf("repo not cloned locally — press [c] to checkout first")
	}

	// Build comment context
	var commentChain string
	for _, c := range thread.Comments {
		commentChain += fmt.Sprintf("@%s: %s\n", c.Author, c.Body)
	}

	prompt := fmt.Sprintf(`Fix the code issue described in this review comment.

File: %s (line %d)
Review comments:
%s

Instructions:
1. Read the file at %s
2. Apply the fix suggested by the reviewer
3. Show me the diff of your changes only

Do NOT explain. Just output the unified diff of changes.`, thread.Path, thread.Line, commentChain, thread.Path)

	return runClaude(prompt, repoPath)
}

// runClaude executes a prompt through Claude Code CLI
func runClaude(prompt string, workDir string) (string, error) {
	path := claudePath()

	args := []string{
		"--print",     // non-interactive, just print response
		"--output-format", "text",
		prompt,
	}

	cmd := exec.Command(path, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("claude failed: %w\nOutput: %s", err, string(out))
	}

	return strings.TrimSpace(string(out)), nil
}

// extractJSON extracts the first JSON object from a string that may contain other text
func extractJSON(s string) string {
	// Find first { and last }
	start := strings.Index(s, "{")
	if start == -1 {
		return s
	}
	// Find matching closing brace
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return s[start:]
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if max < 4 {
		if len(s) > max {
			return s[:max]
		}
		return s
	}
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
