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
Human → TUI (Bubbletea) → PRFlow Logic → gh CLI → GitHub API
                              ↕
                        SQLite Cache
```

## Onboarding TUI (First Run)

When no config exists (`~/.config/prflow/config.yaml` not found), launch onboarding:

```
╔══════════════════════════════════════════════════╗
║  Welcome to PRFlow! Let's get you set up. 🚀     ║
╠══════════════════════════════════════════════════╣
║                                                  ║
║  Step 1/3: GitHub Authentication                 ║
║                                                  ║
║  Checking gh CLI... ✓ authenticated as @nagaconda║
║                                                  ║
║  Step 2/3: Select your repos                     ║
║                                                  ║
║  Scanning your recent activity...                ║
║                                                  ║
║  Found 12 repos with your PRs:                   ║
║  [x] juniper/mist-api        (4 open PRs)       ║
║  [x] hpe/wifi-engine         (3 open PRs)       ║
║  [ ] hpe/wan-core            (2 open PRs)       ║
║  [x] hpe/iot-dash            (1 open PR)        ║
║  [ ] personal/dotfiles       (0 open PRs)       ║
║  ...                                             ║
║                                                  ║
║  [space] toggle  [a] select all  [enter] next    ║
║                                                  ║
║  Step 3/3: Set favorites (★)                     ║
║                                                  ║
║  Star repos you want detailed tracking for:      ║
║  ★ juniper/mist-api                              ║
║  ★ hpe/wifi-engine                               ║
║    hpe/iot-dash                                   ║
║                                                  ║
║  [space] toggle star  [enter] finish             ║
║                                                  ║
║  ✓ Config saved to ~/.config/prflow/config.yaml  ║
║  ✓ Initial sync complete (14 PRs loaded)         ║
║                                                  ║
║  Press [enter] to launch PRFlow!                 ║
╚══════════════════════════════════════════════════╝
```

### Onboarding Steps:
1. **Check `gh auth status`** — if not authenticated, show instructions to run `gh auth login`
2. **Scan repos** — run `gh api user/repos` + search for repos where user has open PRs. Show multi-select list.
3. **Pick favorites** — from selected repos, pick which to star for detailed tracking
4. **Write config** — save to `~/.config/prflow/config.yaml`
5. **Initial sync** — fetch all PR data into SQLite cache
6. **Launch main TUI**

## Main TUI Layout

Two-panel layout:
- **Left sidebar** (narrow): Section navigation + Favorites list
- **Right main panel** (wide): PR list for selected section

### Sections (left sidebar)
1. ⚡ **Do Now** — PRs needing YOUR action (unresolved comments, ready to merge, conflicts, CI failures)
2. ⏳ **Waiting** — PRs blocked on other people (with who + how long)
3. 👀 **Review** — PRs where you're a requested reviewer
4. ★ **Favorites** — List of favorited repos (expandable)
5. ✅ **Done** — Recently merged/closed PRs

### PR List (right panel)
Each PR row shows:
- Repo name + PR number
- Title (truncated)
- Key status indicator (why it's in this section)
- GitHub.com link (shown subtly)
- Time since last activity

### Section Logic

**⚡ Do Now** (sorted by urgency):
- PRs with unresolved review comments addressed to you (show latest comment preview)
- PRs that are approved + CI green (show "READY TO MERGE")
- PRs with merge conflicts (show conflicting files)
- PRs with failing CI (show which checks failed)

**⏳ Waiting** (sorted by wait time):
- PRs where you're the author and waiting for reviewers
- Show each reviewer + how long they've been sitting on it
- Color: 🟢 < 1 day, 🟡 1-3 days, 🔴 3+ days

**👀 Review** (sorted by time waiting):
- PRs where review-requested includes you
- Show author + how long they've been waiting for you

**★ Favorites:**
- Each favorite repo shows: open PR count, last activity
- Selecting a repo shows all its PRs (not just yours)

**✅ Done:**
- Last 20 merged/closed PRs
- When merged, by whom

## Expanded PR View (press Enter on a PR)

Shows full detail:
- PR title, description (first few lines)
- Branch: `feature/xyz → main`
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
- `[r]` opens a text input to reply (posts via `gh api`)

## Reply Feature

Simple text input box:
- Type your reply
- `[ctrl+enter]` or `[enter]` to send (via `gh api`)
- `[esc]` to cancel
- `[tab]` to also resolve the thread when sending

## Key Bindings (Global)

| Key | Action |
|-----|--------|
| ↑/↓ or j/k | Navigate items |
| Tab | Switch sections |
| Enter | Expand/collapse PR or thread |
| o | Open current item on GitHub.com |
| r | Reply to selected comment thread |
| m | Merge selected PR (if ready) |
| c | Checkout PR locally (`gh pr checkout`) |
| n | Nudge reviewer (comment @mention) |
| ★ or f | Toggle favorite on repo |
| / | Search/filter |
| R | Force refresh |
| q | Quit |
| ? | Help overlay |

## `gh` Command Mapping

```
Action                    → gh command
──────────────────────────────────────────────────────
List my authored PRs      → gh search prs --author=@me --state=open --json ...
List review requests      → gh search prs --review-requested=@me --state=open --json ...
PR details                → gh pr view <num> -R <repo> --json title,body,state,reviews,reviewRequests,statusCheckRollup,mergeable,comments,files,...
PR review comments        → gh api repos/{owner}/{repo}/pulls/{num}/comments
PR review threads         → gh api graphql (pullRequest.reviewThreads)
Reply to comment          → gh api -X POST repos/{owner}/{repo}/pulls/{num}/comments -f body="..."
Merge PR                  → gh pr merge <num> -R <repo> --squash
Checkout PR               → gh pr checkout <num> -R <repo>
Nudge reviewer            → gh pr comment <num> -R <repo> --body "@reviewer friendly ping 👋"
List user repos           → gh repo list --json name,owner --limit 100
Recently merged           → gh search prs --author=@me --state=closed --merged --json ...
CI status                 → gh pr checks <num> -R <repo>
```

## SQLite Cache Schema

```sql
CREATE TABLE prs (
    id INTEGER PRIMARY KEY,
    repo TEXT NOT NULL,          -- "org/repo"
    number INTEGER NOT NULL,
    title TEXT,
    state TEXT,                  -- open, closed, merged
    author TEXT,
    branch TEXT,
    base_branch TEXT,
    url TEXT,                    -- github.com link
    created_at TEXT,
    updated_at TEXT,
    mergeable TEXT,              -- MERGEABLE, CONFLICTING, UNKNOWN
    review_decision TEXT,        -- APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED
    ci_status TEXT,              -- SUCCESS, FAILURE, PENDING
    section TEXT,                -- computed: do_now, waiting, review, done
    raw_json TEXT,               -- full gh output for detail view
    fetched_at TEXT,
    UNIQUE(repo, number)
);

CREATE TABLE review_threads (
    id INTEGER PRIMARY KEY,
    pr_id INTEGER REFERENCES prs(id),
    thread_id TEXT,
    path TEXT,                   -- file path
    line INTEGER,
    is_resolved BOOLEAN,
    last_author TEXT,
    last_body TEXT,
    needs_my_reply BOOLEAN,
    url TEXT,                    -- github.com link to this thread
    raw_json TEXT,
    UNIQUE(pr_id, thread_id)
);

CREATE TABLE reviewers (
    id INTEGER PRIMARY KEY,
    pr_id INTEGER REFERENCES prs(id),
    login TEXT,
    state TEXT,                  -- PENDING, APPROVED, CHANGES_REQUESTED, COMMENTED
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
  - juniper/mist-api
  - hpe/wifi-engine
  - hpe/wan-core
  - hpe/iot-dash

favorites:
  - juniper/mist-api
  - hpe/wifi-engine

settings:
  refresh_interval: 2m
  stale_threshold: 3d       # when reviewer wait turns red
  editor: cursor
  repos_dir: ~/repos        # for checkout
  merge_method: squash      # squash, merge, rebase
  page_size: 50             # max PRs to fetch per repo
```

## CLI Interface

```bash
prflow              # launch TUI
prflow setup        # re-run onboarding
prflow sync         # force refresh cache
prflow ls           # quick list (no TUI)
prflow ls --action  # only "do now" items  
prflow open <pr>    # open in browser
prflow config       # open config in $EDITOR
prflow version      # version info
```

## Build & Install

```bash
go build -o prflow .
# or
go install github.com/cheenu1092-oss/prflow@latest
```

## Project Structure

```
prflow/
├── main.go              # entry point, CLI parsing
├── cmd/
│   ├── tui.go           # main TUI command
│   ├── setup.go         # onboarding command
│   ├── list.go          # quick list command
│   └── sync.go          # force sync command
├── internal/
│   ├── gh/
│   │   ├── client.go    # gh CLI wrapper (exec commands, parse JSON)
│   │   └── types.go     # GitHub data types
│   ├── cache/
│   │   ├── db.go        # SQLite operations
│   │   └── sync.go      # sync logic (gh → cache)
│   ├── config/
│   │   └── config.go    # YAML config read/write
│   └── tui/
│       ├── app.go       # main Bubbletea model
│       ├── sidebar.go   # left panel (sections + favs)
│       ├── prlist.go    # right panel (PR list)
│       ├── prdetail.go  # expanded PR view
│       ├── threads.go   # comment thread view
│       ├── reply.go     # reply input
│       ├── onboard.go   # onboarding TUI
│       ├── styles.go    # Lipgloss styles
│       └── keys.go      # key bindings
├── go.mod
├── go.sum
├── README.md
└── SPEC.md
```

## v0.1 MVP Scope

For the first buildable version, focus on:
1. ✅ `gh auth` check
2. ✅ Onboarding TUI (scan repos, select, pick favorites, save config)
3. ✅ Fetch PRs via `gh` into SQLite cache
4. ✅ Main TUI with 4 sections (Do Now, Waiting, Review, Done)
5. ✅ PR list with status indicators
6. ✅ `[o]` open in browser for any PR
7. ✅ `[enter]` expand PR to see details + comment threads
8. ✅ Favorites sidebar

Defer to v0.2:
- Reply to comments
- Checkout integration
- Merge from TUI
- Nudge reviewers
- Background auto-refresh
