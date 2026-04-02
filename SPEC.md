# PRFlow — Terminal-First GitHub PR Dashboard

## Overview
A TUI application built in Go using Bubbletea that wraps the `gh` CLI to give developers a unified, action-oriented view of all their GitHub PRs. Think "lazygit for PR management."

## Core Principle
PRFlow does NOT talk to GitHub API directly. It shells out to `gh` CLI for everything. If `gh auth status` works, PRFlow works. Every item in the TUI provides a GitHub.com link.

## Tech Stack
- **Language:** Go
- **TUI:** Bubbletea (github.com/charmbracelet/bubbletea)
- **Styling:** Lipgloss (github.com/charmbracelet/lipgloss)
- **Lists/Tables:** Bubbles components (github.com/charmbracelet/bubbles)
- **Cache:** SQLite (github.com/mattn/go-sqlite3) for fast offline TUI
- **Auth:** Piggyback on `gh` CLI auth (no separate OAuth)
- **Config:** YAML file at `~/.config/prflow/config.yaml`

## Architecture

```
Human -> TUI (Bubbletea) -> PRFlow Logic -> gh CLI -> GitHub API
                              |
                        SQLite Cache
```

## Onboarding TUI (First Run)

When no config exists (`~/.config/prflow/config.yaml` not found), launch onboarding:

```
+--------------------------------------------------+
|  Welcome to PRFlow! Let's get you set up.         |
+--------------------------------------------------+
|                                                    |
|  Step 1/3: GitHub Authentication                   |
|                                                    |
|  Checking gh CLI... authenticated as @nagaconda    |
|                                                    |
|  Step 2/3: Select your repos                       |
|                                                    |
|  Scanning your recent activity...                  |
|                                                    |
|  Found 12 repos with your PRs:                     |
|  [x] juniper/mist-api        (4 open PRs)         |
|  [x] hpe/wifi-engine         (3 open PRs)         |
|  [ ] hpe/wan-core            (2 open PRs)         |
|  [x] hpe/iot-dash            (1 open PR)          |
|  [ ] personal/dotfiles       (0 open PRs)         |
|  ...                                               |
|                                                    |
|  [space] toggle  [a] select all  [enter] next      |
|                                                    |
|  Step 3/3: Set favorites                           |
|                                                    |
|  Star repos you want detailed tracking for:        |
|  * juniper/mist-api                                |
|  * hpe/wifi-engine                                 |
|    hpe/iot-dash                                    |
|                                                    |
|  [space] toggle star  [enter] finish               |
|                                                    |
|  Config saved to ~/.config/prflow/config.yaml      |
|  Initial sync complete (14 PRs loaded)             |
|                                                    |
|  Press [enter] to launch PRFlow!                   |
+--------------------------------------------------+
```

### Onboarding Steps:
1. **Check `gh auth status`** -- if not authenticated, show instructions to run `gh auth login`
2. **Scan repos** -- run `gh api user/repos` + search for repos where user has open PRs. Show multi-select list.
3. **Pick favorites** -- from selected repos, pick which to star for detailed tracking
4. **Write config** -- save to `~/.config/prflow/config.yaml`
5. **Initial sync** -- fetch all PR data into SQLite cache
6. **Launch main TUI**

## Main TUI Layout

Two-panel layout:
- **Left sidebar** (narrow): Section navigation + Favorites list
- **Right main panel** (wide): PR list for selected section

### Sections (left sidebar)
1. **Do Now** -- PRs needing YOUR action (unresolved comments, ready to merge, conflicts, CI failures)
2. **Waiting** -- PRs blocked on other people (with who + how long)
3. **Review** -- PRs where you're a requested reviewer
4. **Workspace** -- Local git repo status (branch, behind/ahead, dirty files, stale branches)
5. **Needs Attention Again** -- PRs you reviewed that have been updated since your last review

### PR List (right panel)
Each PR row shows:
- Repo name + PR number
- Title (truncated)
- Key status indicator (why it's in this section)
- GitHub.com link (shown subtly)
- Time since last activity

### Section Logic

**Do Now** (sorted by urgency):
- PRs with unresolved review comments addressed to you (show latest comment preview)
- PRs that are approved + CI green (show "READY TO MERGE")
- PRs with merge conflicts (show conflicting files)
- PRs with failing CI (show which checks failed)

**Waiting** (sorted by wait time):
- PRs where you're the author and waiting for reviewers
- Show each reviewer + how long they've been sitting on it
- Color: green < 1 day, yellow 1-3 days, red 3+ days

**Review** (sorted by time waiting):
- PRs where review-requested includes you
- Show author + how long they've been waiting for you

**Workspace:**
- Auto-scans configured directories for git repos
- Per-repo: branch, behind/ahead, modified/staged/untracked files, stale branches
- Links local branches to GitHub PRs
- Color coding: green (clean), yellow (changes), red (far behind/conflicts)

**Needs Attention Again:**
- PRs you previously reviewed that have been updated since your last review
- Filtered to exclude PRs you authored

## Expanded PR View (press Enter on a PR)

Shows full detail:
- PR title, description (first few lines)
- Branch: `feature/xyz -> main`
- Status: Draft / Open / Changes Requested / Approved
- CI checks: list each check + status
- Reviewers: who + their verdict
- Unresolved comment threads (expandable)
- Resolved comment threads (collapsed)
- Files changed summary
- GitHub.com link (prominent)

## Comment Thread View

When expanding a comment thread:
- Show file path + line number
- Show conversation (author, comment text, replies)
- Highlight which comments need YOUR response
- `[o]` opens that specific comment on GitHub.com
- `[r]` resolves the thread
- `[R]` opens a text input to reply

## Reply Feature

Simple text input box:
- Type your reply
- `[enter]` to send (via `gh api` GraphQL)
- `[esc]` to cancel

## Key Bindings

### Navigation
| Key | Action |
|-----|--------|
| up/down or j/k | Navigate items |
| Tab | Next section |
| Shift+Tab | Previous section |
| Enter | Expand PR detail |
| Esc | Back to list |
| q | Quit |

### Actions
| Key | Action |
|-----|--------|
| o | Open in GitHub (browser) |
| c | Checkout PR branch locally |
| C | Clone PR's repo |
| / | Search org repos to clone |
| a | Approve PR (detail view) |
| m | Merge PR (detail view, double-press to confirm) |
| r | Resolve review thread (detail view) |
| R | Reply to review thread (detail view) / Refresh (list view) |
| n | Nudge stale reviewer (waiting/do-now sections, with cooldown) |
| A | AI analysis (detail view, requires Claude Code) |

### Workspace
| Key | Action |
|-----|--------|
| p | `git pull` current branch |
| P | `git push` current branch |
| f | Fetch all repos |
| d | Delete stale (merged) branches |

## `gh` Command Mapping

```
Action                    -> gh command
-----------------------------------------------------------
List my authored PRs      -> gh search prs --author=@me --state=open --json ...
List review requests      -> gh search prs --review-requested=@me --state=open --json ...
List reviewed PRs         -> gh search prs --reviewed-by=@me --state=open --json ...
PR details                -> gh pr view <num> -R <repo> --json title,body,state,reviews,...
PR review threads         -> gh api graphql (pullRequest.reviewThreads)
Reply to thread           -> gh api graphql (addPullRequestReviewThreadReply)
Resolve thread            -> gh api graphql (resolveReviewThread)
Approve PR                -> gh pr review <num> -R <repo> --approve
Merge PR                  -> gh pr merge <num> -R <repo> --squash|--merge|--rebase
Checkout PR               -> gh pr checkout <num> -R <repo>
Nudge reviewer            -> gh pr comment <num> -R <repo> --body "@reviewer ..."
List user repos           -> gh repo list --json name,owner --limit 100
Search repos              -> gh search repos <query> --json nameWithOwner
Clone repo                -> gh repo clone <repo> <dest>
Get PR diff               -> gh pr diff <num> -R <repo>
```

## SQLite Cache Schema

```sql
CREATE TABLE prs (
    id INTEGER PRIMARY KEY,
    repo TEXT NOT NULL,
    number INTEGER NOT NULL,
    title TEXT,
    state TEXT,
    author TEXT,
    branch TEXT,
    base_branch TEXT,
    url TEXT,
    created_at TEXT,
    updated_at TEXT,
    mergeable TEXT,
    review_decision TEXT,
    ci_status TEXT,
    section TEXT,
    raw_json TEXT,
    fetched_at TEXT,
    UNIQUE(repo, number)
);

CREATE TABLE review_threads (
    id INTEGER PRIMARY KEY,
    pr_id INTEGER REFERENCES prs(id),
    thread_id TEXT,
    path TEXT,
    line INTEGER,
    is_resolved BOOLEAN,
    last_author TEXT,
    last_body TEXT,
    needs_my_reply BOOLEAN,
    url TEXT,
    raw_json TEXT,
    UNIQUE(pr_id, thread_id)
);

CREATE TABLE reviewers (
    id INTEGER PRIMARY KEY,
    pr_id INTEGER REFERENCES prs(id),
    login TEXT,
    state TEXT,
    requested_at TEXT,
    UNIQUE(pr_id, login)
);

CREATE TABLE favorites (
    repo TEXT PRIMARY KEY,
    added_at TEXT
);

CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value TEXT
);
```

## Config File

```yaml
# ~/.config/prflow/config.yaml
repos:
  - org/repo-one
  - org/repo-two

favorites:
  - org/repo-one

workspace:
  scan_dirs:
    - ~/repos
    - ~/Projects
    - ~/work
  repos:
    org/repo-one: ~/repos/repo-one

settings:
  refresh_interval: 2m
  stale_threshold: 3d
  editor: vim
  repos_dir: ~/repos
  merge_method: squash
  page_size: 50
  theme: auto
  watch_interval: 2m
```

## CLI Interface

```bash
prflow              # Launch TUI dashboard
prflow setup        # Re-run onboarding wizard
prflow sync         # Force refresh PR cache
prflow ls           # Quick list (no TUI)
prflow ls --json    # JSON output for scripting
prflow open #42     # Open PR in browser
prflow open org/repo#42  # Open specific PR
prflow open org/repo     # Open repo's PR list
prflow watch        # Background mode with OS notifications
prflow watch 5m     # Custom poll interval
prflow config       # Show config file path
prflow doctor       # Check dependencies (gh, git, claude)
prflow version      # Print version
```

## Build & Install

```bash
go build -o prflow .
```

## Project Structure

```
prflow/
├── main.go                        # Entry point
├── cmd/
│   └── root.go                    # CLI command dispatch
├── internal/
│   ├── ai/
│   │   └── analyze.go             # Claude Code integration (optional)
│   ├── cache/
│   │   └── db.go                  # SQLite cache operations
│   ├── config/
│   │   └── config.go              # YAML config read/write/validate
│   ├── deps/
│   │   └── check.go               # Dependency checking (gh, git, claude)
│   ├── gh/
│   │   ├── client.go              # gh CLI wrapper (queries + actions)
│   │   └── runner.go              # Command execution (mockable)
│   ├── notify/
│   │   └── notify.go              # Cross-platform desktop notifications
│   ├── tui/
│   │   ├── dashboard.go           # Main Bubbletea model + event loop
│   │   ├── dashboard_actions.go   # Key bindings & inline actions
│   │   ├── dashboard_render.go    # View rendering (sidebar, cards, detail)
│   │   ├── dashboard_sync.go      # Background data fetching
│   │   ├── onboard.go             # Onboarding wizard TUI
│   │   ├── workspace.go           # Local git repo scanning
│   │   ├── styles.go              # Lipgloss styles
│   │   ├── theme.go               # Auto dark/light theme detection
│   │   ├── sorting.go             # PR urgency sorting
│   │   ├── reviewers.go           # Reviewer state & wait time
│   │   └── refresh.go             # Auto-refresh ticker
│   └── watch/
│       └── watcher.go             # Background polling + OS notifications
├── .github/workflows/
│   ├── ci.yml                     # Build + test + lint
│   └── release.yml                # GoReleaser cross-compile
├── .goreleaser.yaml               # Release config
├── go.mod / go.sum
├── README.md
├── SPEC.md
└── WORKSPACE_SPEC.md
```

## Version History

### v0.1.0 (Current)
All features listed above are implemented and shipping:
- Full TUI with 5 sections (Do Now, Waiting, Review, Workspace, Needs Attention)
- Inline PR actions (approve, merge, checkout, clone, reply, resolve, nudge)
- Workspace with git status, stale branch cleanup
- AI-powered PR analysis (optional, requires Claude Code)
- Background watch mode with OS notifications
- Configurable themes (auto/dark/light)
- Cross-platform builds via GoReleaser (Linux + macOS, amd64 + arm64)

### Future Ideas
- Draft PR creation from TUI
- Advanced filtering and search within PR lists
- Custom keybinding configuration
- IDE integration plugins
