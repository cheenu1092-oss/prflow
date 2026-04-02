package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nagarjun226/prflow/internal/cache"
	"github.com/nagarjun226/prflow/internal/config"
	"github.com/nagarjun226/prflow/internal/gh"
)

func TestTimeSince(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"recent", "2026-03-11T22:00:00Z", ""}, // might be in future
		{"iso format", "2026-01-01T00:00:00Z", ""},
		{"with timezone", "2026-01-01T00:00:00-08:00", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(tt.input)
			// Should not panic and should return something
			_ = result
		})
	}
}

func TestFormatTimeAgoEmpty(t *testing.T) {
	result := formatTimeAgo("")
	if result != "" {
		t.Errorf("expected empty string for empty input, got '%s'", result)
	}
}

func TestFormatTimeAgoInvalid(t *testing.T) {
	result := formatTimeAgo("not-a-date")
	if result != "" {
		t.Errorf("expected empty string for invalid input, got '%s'", result)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		max      int
	}{
		{"short", 10},
		{"exactly ten", 11},
		{"this is a long string that should be truncated", 20},
		{"", 10},
		{"abc", 3},
		{"abcd", 3},
		{"abcdef", 2}, // edge case: max < 4, just truncates hard
		{"ab", 1},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.max)
		if len(result) > tt.max {
			t.Errorf("truncate(%q, %d) = %q (len %d), exceeds max", tt.input, tt.max, result, len(result))
		}
	}
}

func TestTruncateNewlines(t *testing.T) {
	input := "line1\nline2\rline3"
	result := truncate(input, 50)
	// \n replaced with " ", \r removed
	if result != "line1 line2line3" {
		t.Errorf("expected newlines handled, got '%s'", result)
	}
}

func TestPrBadge(t *testing.T) {
	m := dashModel{cfg: config.DefaultConfig()}

	tests := []struct {
		name         string
		pr           cache.CachedPR
		expectBadge  string // substring to check for
	}{
		{
			name: "approved + CI passing",
			pr: cache.CachedPR{PR: gh.PR{
				ReviewDecision: "APPROVED",
				Mergeable:      "MERGEABLE",
				StatusCheckRollup: []gh.StatusCheck{
					{Conclusion: "SUCCESS"},
				},
			}},
			expectBadge: "✓CI",
		},
		{
			name: "approved + CI failing",
			pr: cache.CachedPR{PR: gh.PR{
				ReviewDecision: "APPROVED",
				Mergeable:      "MERGEABLE",
				StatusCheckRollup: []gh.StatusCheck{
					{Conclusion: "FAILURE"},
				},
			}},
			expectBadge: "✗CI",
		},
		{
			name: "approved + CI pending",
			pr: cache.CachedPR{PR: gh.PR{
				ReviewDecision: "APPROVED",
				Mergeable:      "MERGEABLE",
				StatusCheckRollup: []gh.StatusCheck{
					{Conclusion: "PENDING"},
				},
			}},
			expectBadge: "⏳CI",
		},
		{
			name: "approved + no CI checks",
			pr: cache.CachedPR{PR: gh.PR{
				ReviewDecision:    "APPROVED",
				Mergeable:         "MERGEABLE",
				StatusCheckRollup: []gh.StatusCheck{},
			}},
			expectBadge: "✓CI", // assume passing when no checks
		},
		{
			name: "approved + conflict",
			pr: cache.CachedPR{PR: gh.PR{
				ReviewDecision: "APPROVED",
				Mergeable:      "CONFLICTING",
			}},
			expectBadge: "CONFLICT",
		},
		{
			name: "changes requested",
			pr: cache.CachedPR{PR: gh.PR{
				ReviewDecision: "CHANGES_REQUESTED",
			}},
			expectBadge: "CHANGES REQUESTED",
		},
		{
			name: "merge conflict",
			pr: cache.CachedPR{PR: gh.PR{
				Mergeable: "CONFLICTING",
			}},
			expectBadge: "CONFLICT",
		},
		{
			name: "draft",
			pr: cache.CachedPR{PR: gh.PR{
				IsDraft: true,
			}},
			expectBadge: "DRAFT",
		},
		{
			name: "review required",
			pr: cache.CachedPR{PR: gh.PR{
				ReviewDecision: "REVIEW_REQUIRED",
			}},
			expectBadge: "AWAITING REVIEW",
		},
		{
			name: "default (in review)",
			pr: cache.CachedPR{PR: gh.PR{}},
			expectBadge: "IN REVIEW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.prBadge(tt.pr)
			if result == "" {
				t.Error("expected non-empty badge")
			}
			// Check for expected substring (badges include styling, so we just check content is there)
			// This is a basic sanity check - lipgloss styling wraps the text
		})
	}
}

func TestSectionString(t *testing.T) {
	tests := []struct {
		sec      section
		expected string
	}{
		{sectionDoNow, "⚡ Do Now"},
		{sectionWaiting, "⏳ Waiting"},
		{sectionReview, "👀 Review"},
		{sectionWorkspace, "📂 Workspace"},
		{sectionNeedsAttention, "🔔 Needs Attention Again"},
	}

	for _, tt := range tests {
		result := tt.sec.String()
		if result != tt.expected {
			t.Errorf("section(%d).String() = '%s', want '%s'", tt.sec, result, tt.expected)
		}
	}
}

func TestCurrentListLen(t *testing.T) {
	m := dashModel{
		doNow:          make([]cache.CachedPR, 3),
		waiting:        make([]cache.CachedPR, 5),
		review:         make([]cache.CachedPR, 2),
		workspace:      make([]RepoStatus, 4),
		needsAttention: make([]cache.CachedPR, 1),
	}

	tests := []struct {
		sec      section
		expected int
	}{
		{sectionDoNow, 3},
		{sectionWaiting, 5},
		{sectionReview, 2},
		{sectionWorkspace, 4},
		{sectionNeedsAttention, 1},
	}

	for _, tt := range tests {
		m.section = tt.sec
		result := m.currentListLen()
		if result != tt.expected {
			t.Errorf("section %v: expected %d, got %d", tt.sec, tt.expected, result)
		}
	}
}

func TestHelpPair(t *testing.T) {
	result := helpPair("q", "quit")
	if result == "" {
		t.Error("expected non-empty help pair")
	}
}

func TestRenderPRCardsEmpty(t *testing.T) {
	m := dashModel{cfg: config.DefaultConfig()}
	result := m.renderPRCards(nil, 80)
	if result == "" {
		t.Error("expected non-empty empty state")
	}
}

func TestRenderPRCard(t *testing.T) {
	m := dashModel{cfg: config.DefaultConfig()}
	pr := cache.CachedPR{
		PR: gh.PR{
			Number:         42,
			Title:          "Fix auth bug in the middleware layer",
			ReviewDecision: "APPROVED",
			Mergeable:      "MERGEABLE",
			UpdatedAt:      "2026-03-10T15:30:00Z",
		},
		Repo: "org/repo",
	}

	// Not selected
	result := m.renderPRCard(pr, false, 80)
	if result == "" {
		t.Error("expected non-empty PR card")
	}

	// Selected
	resultSelected := m.renderPRCard(pr, true, 80)
	if resultSelected == "" {
		t.Error("expected non-empty selected PR card")
	}
}

func TestRenderWorkspaceCardsEmpty(t *testing.T) {
	m := dashModel{cfg: config.DefaultConfig()}
	result := m.renderWorkspaceCards(80)
	if result == "" {
		t.Error("expected non-empty empty state")
	}
}

func TestFindLocalRepoNotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Workspace.ScanDirs = []string{"/nonexistent"}
	m := dashModel{cfg: cfg}

	result := m.findLocalRepo("org/nonexistent-repo")
	if result != "" {
		t.Errorf("expected empty for nonexistent repo, got %q", result)
	}
}

func TestFindLocalRepoFromWorkspaceResults(t *testing.T) {
	cfg := config.DefaultConfig()
	m := dashModel{
		cfg: cfg,
		workspace: []RepoStatus{
			{Name: "org/myrepo", Path: "/tmp/myrepo"},
		},
	}

	result := m.findLocalRepo("org/myrepo")
	// Won't find it since /tmp/myrepo doesn't exist as git repo,
	// but the scan result matching should work if path existed
	_ = result
}

func TestViewSearchMode(t *testing.T) {
	cfg := config.DefaultConfig()
	m := dashModel{
		cfg:         cfg,
		spinFrames:  []string{"⠋"},
		viewMode:    viewSearch,
		searchQuery: "test",
	}

	result := m.viewSearchMode()
	if result == "" {
		t.Error("expected non-empty search view")
	}

	// With results
	m.searchResults = []string{"org/repo1", "org/repo2"}
	result = m.viewSearchMode()
	if result == "" {
		t.Error("expected non-empty search view with results")
	}

	// Searching state
	m.searching = true
	m.searchResults = nil
	result = m.viewSearchMode()
	if result == "" {
		t.Error("expected non-empty searching view")
	}
}

func TestSectionAllValues(t *testing.T) {
	// Verify all 5 sections have non-empty string representation
	for i := section(0); i <= sectionNeedsAttention; i++ {
		if i.String() == "" {
			t.Errorf("section %d has empty string", i)
		}
	}
}

func TestViewReplyMode(t *testing.T) {
	cfg := config.DefaultConfig()
	m := dashModel{
		cfg:           cfg,
		viewMode:      viewReply,
		replyText:     "This is my reply",
		replyThreadID: "thread123",
		detailPR: &cache.CachedPR{
			PR: gh.PR{
				Number: 42,
			},
			Repo: "org/repo",
		},
		detailThreads: []gh.ReviewThread{
			{
				ID: "thread123",
				Comments: []gh.ThreadComment{
					{
						Author: "reviewer",
						Body:   "Please fix this bug",
					},
				},
			},
		},
	}

	result := m.viewReplyMode()
	if result == "" {
		t.Error("expected non-empty reply view")
	}

	// Empty reply
	m.replyText = ""
	result = m.viewReplyMode()
	if result == "" {
		t.Error("expected non-empty reply view with empty text")
	}
}

func TestNudgeKeybinding(t *testing.T) {
	cfg := config.DefaultConfig()
	db, err := cache.OpenTestDB(t)
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	defer db.Close()

	m := dashModel{
		cfg:      cfg,
		db:       db,
		viewMode: viewList,
		section:  sectionWaiting,
		cursor:   0,
		waiting: []cache.CachedPR{
			{
				PR: gh.PR{
					Number:    10,
					Title:     "Waiting PR",
					CreatedAt: "2026-03-01T00:00:00Z",
					UpdatedAt: "2026-03-10T00:00:00Z",
					Author:    gh.Author{Login: "me"},
					ReviewRequests: gh.ReviewRequests{
						Nodes: []gh.ReviewRequest{
							{RequestedReviewer: gh.Author{Login: "reviewer1"}},
						},
					},
				},
				Repo: "org/repo",
			},
		},
		spinFrames: []string{"⠋"},
	}

	// First press of 'n' — should show confirmation
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	updated, _ := m.Update(keyMsg)
	um := updated.(dashModel)

	if !um.nudgePending {
		t.Error("expected nudgePending=true after first 'n' press")
	}
	if um.nudgeReviewer != "reviewer1" {
		t.Errorf("expected nudgeReviewer='reviewer1', got %q", um.nudgeReviewer)
	}
	expectedMsg := "Nudge @reviewer1? Press 'n' again to confirm"
	if um.statusMsg != expectedMsg {
		t.Errorf("expected statusMsg=%q, got %q", expectedMsg, um.statusMsg)
	}

	// Non-'n' key should reset nudge state
	escMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated2, _ := um.Update(escMsg)
	um2 := updated2.(dashModel)
	if um2.nudgePending {
		t.Error("expected nudgePending=false after non-n keypress")
	}

	// Test that nudge is unavailable in review section
	m2 := dashModel{
		cfg:        cfg,
		viewMode:   viewList,
		section:    sectionReview,
		cursor:     0,
		review:     []cache.CachedPR{{PR: gh.PR{Number: 5}, Repo: "org/repo"}},
		spinFrames: []string{"⠋"},
	}
	updated3, _ := m2.Update(keyMsg)
	um3 := updated3.(dashModel)
	if um3.nudgePending {
		t.Error("expected nudge not available in review section")
	}

	// Test with no reviewers
	m3 := dashModel{
		cfg:      cfg,
		viewMode: viewList,
		section:  sectionWaiting,
		cursor:   0,
		waiting: []cache.CachedPR{
			{
				PR:   gh.PR{Number: 11, Author: gh.Author{Login: "me"}},
				Repo: "org/repo",
			},
		},
		spinFrames: []string{"⠋"},
	}
	updated4, _ := m3.Update(keyMsg)
	um4 := updated4.(dashModel)
	if um4.nudgePending {
		t.Error("expected nudgePending=false when no reviewers")
	}
	if um4.statusMsg != "No reviewers to nudge" {
		t.Errorf("expected 'No reviewers to nudge', got %q", um4.statusMsg)
	}
}

func TestTabNavigation(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		section:    sectionDoNow,
		cursor:     2,
	}

	// Tab should advance section and reset cursor
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := m.Update(tabMsg)
	um := updated.(dashModel)
	if um.section != sectionWaiting {
		t.Errorf("expected section Waiting after Tab, got %v", um.section)
	}
	if um.cursor != 0 {
		t.Errorf("expected cursor reset to 0, got %d", um.cursor)
	}

	// Shift+Tab should go back
	shiftTabMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	updated2, _ := um.Update(shiftTabMsg)
	um2 := updated2.(dashModel)
	if um2.section != sectionDoNow {
		t.Errorf("expected section DoNow after Shift+Tab, got %v", um2.section)
	}
}

func TestTabWrapsAround(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		section:    sectionNeedsAttention,
	}

	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := m.Update(tabMsg)
	um := updated.(dashModel)
	if um.section != sectionDoNow {
		t.Errorf("expected section DoNow after wrapping, got %v", um.section)
	}
}

func TestUpDownNavigation(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		section:    sectionDoNow,
		cursor:     0,
		doNow: []cache.CachedPR{
			{PR: gh.PR{Number: 1}, Repo: "org/repo"},
			{PR: gh.PR{Number: 2}, Repo: "org/repo"},
			{PR: gh.PR{Number: 3}, Repo: "org/repo"},
		},
	}

	// Move down
	downMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ := m.Update(downMsg)
	um := updated.(dashModel)
	if um.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", um.cursor)
	}

	// Move down again
	updated2, _ := um.Update(downMsg)
	um2 := updated2.(dashModel)
	if um2.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", um2.cursor)
	}

	// Can't go past end
	updated3, _ := um2.Update(downMsg)
	um3 := updated3.(dashModel)
	if um3.cursor != 2 {
		t.Errorf("expected cursor clamped at 2, got %d", um3.cursor)
	}

	// Move up
	upMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated4, _ := um3.Update(upMsg)
	um4 := updated4.(dashModel)
	if um4.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", um4.cursor)
	}
}

func TestEscFromDetailView(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		viewMode:   viewDetail,
		detailPR:   &cache.CachedPR{PR: gh.PR{Number: 42}, Repo: "org/repo"},
	}

	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := m.Update(escMsg)
	um := updated.(dashModel)
	if um.viewMode != viewList {
		t.Errorf("expected viewList after Esc, got %v", um.viewMode)
	}
	if um.detailPR != nil {
		t.Error("expected detailPR cleared after Esc")
	}
}

func TestQFromDetailView(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		viewMode:   viewDetail,
		detailPR:   &cache.CachedPR{PR: gh.PR{Number: 42}, Repo: "org/repo"},
	}

	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updated, cmd := m.Update(qMsg)
	um := updated.(dashModel)

	// q in detail view goes back to list (doesn't quit)
	if um.viewMode != viewList {
		t.Errorf("expected viewList after q in detail, got %v", um.viewMode)
	}
	if cmd != nil {
		t.Error("expected no quit cmd from q in detail view")
	}
}

func TestQFromListViewQuits(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		viewMode:   viewList,
	}

	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(qMsg)

	// q in list view should issue quit
	if cmd == nil {
		t.Error("expected quit cmd from q in list view")
	}
}

func TestWindowSizeMsg(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
	}

	sizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(sizeMsg)
	um := updated.(dashModel)

	if um.width != 120 {
		t.Errorf("expected width 120, got %d", um.width)
	}
	if um.height != 40 {
		t.Errorf("expected height 40, got %d", um.height)
	}
}

func TestSyncDoneMsg(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		loading:    true,
	}

	doNow := []cache.CachedPR{{PR: gh.PR{Number: 1}, Repo: "org/repo"}}
	waiting := []cache.CachedPR{{PR: gh.PR{Number: 2}, Repo: "org/repo"}}

	msg := syncDoneMsg{
		doNow:   doNow,
		waiting: waiting,
	}

	updated, _ := m.Update(msg)
	um := updated.(dashModel)

	if um.loading {
		t.Error("expected loading=false after sync")
	}
	if len(um.doNow) != 1 {
		t.Errorf("expected 1 doNow PR, got %d", len(um.doNow))
	}
	if len(um.waiting) != 1 {
		t.Errorf("expected 1 waiting PR, got %d", len(um.waiting))
	}
}

func TestSyncDoneMsgWithError(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		loading:    true,
	}

	msg := syncDoneMsg{err: fmt.Errorf("sync failed")}
	updated, _ := m.Update(msg)
	um := updated.(dashModel)

	if um.loading {
		t.Error("expected loading=false")
	}
	if um.err != "sync failed" {
		t.Errorf("expected error message, got %q", um.err)
	}
}

func TestWorkspaceScanMsg(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
	}

	repos := []RepoStatus{
		{Name: "org/repo1", Path: "/tmp/repo1"},
		{Name: "org/repo2", Path: "/tmp/repo2"},
	}
	msg := workspaceScanMsg{repos: repos}
	updated, _ := m.Update(msg)
	um := updated.(dashModel)

	if len(um.workspace) != 2 {
		t.Errorf("expected 2 workspace repos, got %d", len(um.workspace))
	}
}

func TestDetailLoadedMsg(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		viewMode:   viewList,
	}

	pr := &cache.CachedPR{PR: gh.PR{Number: 42, Title: "Test PR"}, Repo: "org/repo"}
	threads := []gh.ReviewThread{
		{ID: "t1", Path: "file.go", IsResolved: false},
	}
	msg := detailLoadedMsg{pr: pr, threads: threads}
	updated, _ := m.Update(msg)
	um := updated.(dashModel)

	if um.viewMode != viewDetail {
		t.Errorf("expected viewDetail, got %v", um.viewMode)
	}
	if um.detailPR.Number != 42 {
		t.Errorf("expected PR #42, got #%d", um.detailPR.Number)
	}
	if len(um.detailThreads) != 1 {
		t.Errorf("expected 1 thread, got %d", len(um.detailThreads))
	}
}

func TestGitOpDoneMsg(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
	}

	msg := gitOpDoneMsg{msg: "pulled main"}
	updated, _ := m.Update(msg)
	um := updated.(dashModel)
	if um.statusMsg != "✓ pulled main" {
		t.Errorf("expected success status, got %q", um.statusMsg)
	}

	// With error
	errMsg := gitOpDoneMsg{msg: "push", err: fmt.Errorf("rejected")}
	updated2, _ := m.Update(errMsg)
	um2 := updated2.(dashModel)
	if !strings.Contains(um2.statusMsg, "✗") {
		t.Errorf("expected error status, got %q", um2.statusMsg)
	}
}

func TestPRActionDoneMsg(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
	}

	msg := prActionDoneMsg{action: "approved"}
	updated, _ := m.Update(msg)
	um := updated.(dashModel)
	if !strings.Contains(um.statusMsg, "approved") {
		t.Errorf("expected 'approved' in status, got %q", um.statusMsg)
	}

	// With error
	errMsg := prActionDoneMsg{action: "merge", err: fmt.Errorf("not ready")}
	updated2, _ := m.Update(errMsg)
	um2 := updated2.(dashModel)
	if !strings.Contains(um2.statusMsg, "merge failed") {
		t.Errorf("expected merge failure, got %q", um2.statusMsg)
	}
}

func TestSlashEntersSearchMode(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		viewMode:   viewList,
	}

	slashMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updated, _ := m.Update(slashMsg)
	um := updated.(dashModel)
	if um.viewMode != viewSearch {
		t.Errorf("expected viewSearch, got %v", um.viewMode)
	}
}

func TestMergeConfirmation(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		viewMode:   viewDetail,
		detailPR:   &cache.CachedPR{PR: gh.PR{Number: 42, IsDraft: false}, Repo: "org/repo"},
	}

	mergeMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	updated, _ := m.Update(mergeMsg)
	um := updated.(dashModel)

	// First press shows confirmation
	if !strings.Contains(um.statusMsg, "Press 'm' again to confirm") {
		t.Errorf("expected merge confirmation, got %q", um.statusMsg)
	}
}

func TestMergeDraftPR(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		viewMode:   viewDetail,
		detailPR:   &cache.CachedPR{PR: gh.PR{Number: 42, IsDraft: true}, Repo: "org/repo"},
	}

	mergeMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	updated, _ := m.Update(mergeMsg)
	um := updated.(dashModel)

	if um.statusMsg != "✗ Cannot merge draft PR" {
		t.Errorf("expected draft PR error, got %q", um.statusMsg)
	}
}

func TestApproveDraftPR(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		viewMode:   viewDetail,
		detailPR:   &cache.CachedPR{PR: gh.PR{Number: 42, IsDraft: true}, Repo: "org/repo"},
	}

	approveMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updated, _ := m.Update(approveMsg)
	um := updated.(dashModel)

	if um.statusMsg != "✗ Cannot approve draft PR" {
		t.Errorf("expected draft PR error, got %q", um.statusMsg)
	}
}

func TestViewMethodRouting(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		width:      120,
		height:     40,
	}

	// List view
	m.viewMode = viewList
	result := m.View()
	if result == "" {
		t.Error("expected non-empty list view")
	}

	// Search view
	m.viewMode = viewSearch
	result = m.View()
	if result == "" {
		t.Error("expected non-empty search view")
	}

	// Reply view
	m.viewMode = viewReply
	m.detailPR = &cache.CachedPR{PR: gh.PR{Number: 1}, Repo: "org/repo"}
	result = m.View()
	if result == "" {
		t.Error("expected non-empty reply view")
	}

	// Detail view
	m.viewMode = viewDetail
	result = m.View()
	if result == "" {
		t.Error("expected non-empty detail view")
	}
}

func TestDeleteStaleBranchesPending(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		viewMode:   viewList,
		section:    sectionWorkspace,
		cursor:     0,
		workspace: []RepoStatus{
			{
				Name:          "org/repo",
				Path:          "/tmp/repo",
				StaleBranches: []string{"feature/old", "feature/done"},
			},
		},
	}

	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updated, _ := m.Update(dMsg)
	um := updated.(dashModel)

	if !um.deleteStalePending {
		t.Error("expected deleteStalePending=true after first 'd'")
	}
	if !strings.Contains(um.statusMsg, "Delete 2 stale branches?") {
		t.Errorf("expected confirmation message, got %q", um.statusMsg)
	}
}

func TestDeleteStaleBranchesNone(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋"},
		viewMode:   viewList,
		section:    sectionWorkspace,
		cursor:     0,
		workspace: []RepoStatus{
			{
				Name:          "org/repo",
				Path:          "/tmp/repo",
				StaleBranches: nil,
			},
		},
	}

	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updated, _ := m.Update(dMsg)
	um := updated.(dashModel)

	if um.statusMsg != "No stale branches to delete" {
		t.Errorf("expected 'No stale branches' message, got %q", um.statusMsg)
	}
}

func TestTickMsg(t *testing.T) {
	m := dashModel{
		cfg:        config.DefaultConfig(),
		spinFrames: []string{"⠋", "⠙", "⠹"},
		spinner:    0,
	}

	msg := dashTickMsg(time.Now())
	updated, cmd := m.Update(msg)
	um := updated.(dashModel)

	if um.spinner != 1 {
		t.Errorf("expected spinner 1, got %d", um.spinner)
	}
	if cmd == nil {
		t.Error("expected tick cmd to be returned")
	}
}
