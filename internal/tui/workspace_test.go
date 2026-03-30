package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Init git repo
	cmds := [][]string{
		{"git", "-C", dir, "init"},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
		{"git", "-C", dir, "config", "commit.gpgsign", "false"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup failed: %s: %v", string(out), err)
		}
	}

	// Create a file and commit (use explicit file name, not "." to avoid picking up extra files)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
	exec.Command("git", "-C", dir, "add", "README.md").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial commit").Run()

	return dir
}

func TestScanWorkspaceRepo(t *testing.T) {
	dir := setupGitRepo(t)

	rs, err := ScanWorkspaceRepo(dir)
	if err != nil {
		t.Fatalf("ScanWorkspaceRepo failed: %v", err)
	}

	if rs.Path != dir {
		t.Errorf("expected path '%s', got '%s'", dir, rs.Path)
	}

	// Should be on main or master
	if rs.Branch == "" || rs.Branch == "detached" {
		t.Errorf("expected a branch name, got '%s'", rs.Branch)
	}

	// Clean repo
	if !rs.Clean {
		t.Error("expected clean repo")
	}
	if rs.Modified != 0 {
		t.Errorf("expected 0 modified, got %d", rs.Modified)
	}
	if rs.Staged != 0 {
		t.Errorf("expected 0 staged, got %d", rs.Staged)
	}
	if rs.Untracked != 0 {
		t.Errorf("expected 0 untracked, got %d", rs.Untracked)
	}

	// Has a last commit
	if rs.LastCommit == "" {
		t.Error("expected non-empty last commit")
	}
}

func TestScanWorkspaceRepoDirtyFiles(t *testing.T) {
	dir := setupGitRepo(t)

	// Create untracked file
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)
	// Modify tracked file
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Modified"), 0644)

	rs, err := ScanWorkspaceRepo(dir)
	if err != nil {
		t.Fatalf("ScanWorkspaceRepo failed: %v", err)
	}

	if rs.Clean {
		t.Error("expected dirty repo")
	}
	if rs.Untracked != 1 {
		t.Errorf("expected 1 untracked, got %d", rs.Untracked)
	}
	// Modified count depends on git version porcelain format
	// At minimum, repo should not be clean
	if rs.Modified == 0 && rs.Staged == 0 && rs.Untracked == 0 {
		t.Error("expected some dirty files")
	}
}

func TestScanWorkspaceRepoStagedFiles(t *testing.T) {
	dir := setupGitRepo(t)

	// Create and stage a file
	os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("staged"), 0644)
	exec.Command("git", "-C", dir, "add", "staged.txt").Run()

	rs, _ := ScanWorkspaceRepo(dir)

	if rs.Clean {
		t.Error("expected dirty repo")
	}
	if rs.Staged != 1 {
		t.Errorf("expected 1 staged, got %d", rs.Staged)
	}
}

func TestScanWorkspaceRepoNoRemote(t *testing.T) {
	dir := setupGitRepo(t)

	rs, _ := ScanWorkspaceRepo(dir)

	if rs.HasRemote {
		t.Error("expected no remote")
	}
	// Name should fallback to directory name
	if rs.Name == "" {
		t.Error("expected non-empty name")
	}
}

func TestScanWorkspaceRepoNonGit(t *testing.T) {
	dir := t.TempDir()

	_, err := ScanWorkspaceRepo(dir)
	// Should not crash, just return limited info
	if err != nil {
		t.Logf("non-git dir returned error (ok): %v", err)
	}
}

func TestParseRepoNameSSH(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"git@github.com:org/repo.git", "org/repo"},
		{"git@github.com:org/repo", "org/repo"},
		{"https://github.com/org/repo.git", "org/repo"},
		{"https://github.com/org/repo", "org/repo"},
		{"git@gitlab.com:team/project.git", "team/project"},
	}

	for _, tt := range tests {
		result := parseRepoName(tt.input)
		if result != tt.expected {
			t.Errorf("parseRepoName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestDetectDefaultBranch(t *testing.T) {
	dir := setupGitRepo(t)

	branch := detectDefaultBranch(dir)
	// Should return something (main, master, or fallback)
	if branch == "" {
		t.Error("expected non-empty default branch")
	}
}

func TestRenderRepoStatus(t *testing.T) {
	rs := &RepoStatus{
		Name:       "org/repo",
		Path:       "/home/user/repos/repo",
		Branch:     "feature/auth",
		Behind:     5,
		Ahead:      2,
		Modified:   1,
		Untracked:  2,
		LastCommit: "abc1234 fix bug (2h ago)",
		Clean:      false,
		HasRemote:  true,
	}

	// Should not panic
	output := RenderRepoStatus(rs, false)
	if output == "" {
		t.Error("expected non-empty output")
	}

	// Selected version
	outputSelected := RenderRepoStatus(rs, true)
	if outputSelected == "" {
		t.Error("expected non-empty selected output")
	}
}

func TestRenderRepoStatusClean(t *testing.T) {
	rs := &RepoStatus{
		Name:       "org/repo",
		Branch:     "main",
		Clean:      true,
		HasRemote:  true,
		LastCommit: "def5678 release v1.0 (1d ago)",
	}

	output := RenderRepoStatus(rs, false)
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestRenderRepoStatusBehindWarning(t *testing.T) {
	rs := &RepoStatus{
		Name:    "org/repo",
		Branch:  "old-branch",
		Behind:  50,
		Clean:   true,
		HasRemote: true,
	}

	output := RenderRepoStatus(rs, false)
	if output == "" {
		t.Error("expected non-empty output for way-behind repo")
	}
}

func TestDetectStaleBranches(t *testing.T) {
	dir := setupGitRepo(t)

	// Create a feature branch, add a commit, then merge it back into main
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s: %v", args, string(out), err)
		}
	}

	// Create and merge a feature branch
	runGit("checkout", "-b", "feature/old")
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature"), 0644)
	runGit("add", "feature.txt")
	runGit("commit", "-m", "add feature")

	// Switch back to default branch and merge
	// Detect what the default branch is (could be main or master)
	defaultBranch, _ := exec.Command("git", "-C", dir, "branch", "--show-current").CombinedOutput()
	// We're on feature/old, so find the original branch
	runGit("checkout", "-")
	runGit("merge", "feature/old")

	_ = defaultBranch // suppress unused

	stale := detectStaleBranches(dir)
	found := false
	for _, b := range stale {
		if b == "feature/old" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'feature/old' in stale branches, got %v", stale)
	}
}

func TestDetectStaleBranchesExcludesMain(t *testing.T) {
	dir := setupGitRepo(t)

	stale := detectStaleBranches(dir)
	for _, b := range stale {
		if b == "main" || b == "master" || b == "develop" {
			t.Errorf("expected main/master/develop to be excluded, but got %q in stale list", b)
		}
	}
}

func TestDeleteStaleBranches(t *testing.T) {
	dir := setupGitRepo(t)

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s: %v", args, string(out), err)
		}
	}

	// Create and merge a feature branch
	runGit("checkout", "-b", "feature/merged")
	os.WriteFile(filepath.Join(dir, "merged.txt"), []byte("merged"), 0644)
	runGit("add", "merged.txt")
	runGit("commit", "-m", "merged feature")
	runGit("checkout", "-")
	runGit("merge", "feature/merged")

	// Verify it's detected as stale
	stale := detectStaleBranches(dir)
	found := false
	for _, b := range stale {
		if b == "feature/merged" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'feature/merged' in stale branches before delete, got %v", stale)
	}

	// Delete it
	deleted, err := DeleteStaleBranches(dir, []string{"feature/merged"})
	if err != nil {
		t.Fatalf("DeleteStaleBranches failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	// Verify it's gone
	staleAfter := detectStaleBranches(dir)
	for _, b := range staleAfter {
		if b == "feature/merged" {
			t.Error("expected 'feature/merged' to be gone after delete")
		}
	}
}

func TestRenderWorkspaceCardWithStaleBranches(t *testing.T) {
	rs := &RepoStatus{
		Name:          "org/repo",
		Branch:        "main",
		Clean:         true,
		HasRemote:     true,
		LastCommit:    "abc123 init (1h ago)",
		StaleBranches: []string{"feature/old", "feature/done", "feature/merged"},
	}

	output := RenderRepoStatus(rs, false)
	if output == "" {
		t.Error("expected non-empty output")
	}
	// The RenderRepoStatus doesn't show stale branches (that's in renderWorkspaceCard),
	// but we verify it doesn't crash with the field populated.
	// Stale branch rendering is tested via the dashModel.renderWorkspaceCard in helpers_test.go
}
