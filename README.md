# PRFlow ⚡

**Your morning coffee PR companion.** A terminal-first GitHub PR dashboard that tells you what needs your attention right now. No more context-switching to GitHub.com.

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green)

## Why?

AI-accelerated development means more PRs, more reviews, more context-switching. GitHub's notification firehose doesn't answer the simple question: **"What should I do right now?"**

PRFlow does. One command, one TUI, everything prioritized by action needed.

## Features

- **⚡ Do Now** — PRs needing YOUR action (unresolved comments, ready to merge, conflicts, CI failures)
- **⏳ Waiting** — PRs blocked on reviewers (with who + how long, color-coded)
- **👀 Review** — PRs where you're a requested reviewer
- **📂 Workspace** — Local git status for all your repos (branch, behind/ahead, dirty files, stale branches)
- **🔔 Needs Attention** — PRs you reviewed that have been updated since
- **★ Favorites** — Star repos for detailed sidebar tracking
- **🤖 AI Analysis** — Optional Claude Code integration for PR analysis and draft replies
- **🔔 Watch Mode** — Background OS notifications when PR state changes
- **🎨 Themes** — Auto dark/light detection, or set manually

## Install

### From Source

```bash
# Prerequisites: Go 1.24+, gh CLI authenticated
git clone https://github.com/nagarjun226/prflow.git
cd prflow
go build -o prflow .
mv prflow /usr/local/bin/
```

### Prerequisites

1. **Go 1.24+** — [golang.org/dl](https://golang.org/dl/)
2. **GitHub CLI (`gh`)** — [cli.github.com](https://cli.github.com)
3. **Authenticate gh:**
   ```bash
   gh auth login
   gh auth status  # verify it works
   ```

## Quick Start

```bash
# First run — launches onboarding wizard
prflow

# Or run setup explicitly
prflow setup
```

## Commands

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

## TUI Key Bindings

### Navigation
| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate items |
| `Tab` | Next section |
| `Shift+Tab` | Previous section |
| `Enter` | Expand PR detail |
| `Esc` | Back to list |
| `q` | Quit |

### Actions
| Key | Action |
|-----|--------|
| `o` | Open in GitHub (browser) |
| `c` | Checkout PR branch locally |
| `C` | Clone PR's repo |
| `/` | Search org repos to clone |
| `a` | Approve PR (detail view) |
| `m` | Merge PR (detail view, double-press to confirm) |
| `r` | Resolve review thread (detail view) |
| `R` | Reply to review thread (detail view) / Refresh (list view) |
| `n` | Nudge stale reviewer (waiting/do-now sections) |
| `A` | AI analysis (detail view, requires Claude Code) |

### Workspace (📂 section)
| Key | Action |
|-----|--------|
| `p` | `git pull` current branch |
| `P` | `git push` current branch |
| `f` | Fetch all repos |
| `d` | Delete stale (merged) branches |

## Configuration

Config lives at `~/.config/prflow/config.yaml`:

```yaml
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
  refresh_interval: 2m      # Auto-refresh interval
  stale_threshold: 3d        # When reviewer wait turns red
  editor: vim                # Default editor
  repos_dir: ~/repos         # Where to clone repos
  merge_method: squash       # Merge strategy (squash/merge/rebase)
  page_size: 50              # Max PRs per repo fetch
  theme: auto                # Theme: auto, dark, light
  watch_interval: 2m         # Watch mode poll interval
```

## Architecture

PRFlow is a thin wrapper around the `gh` CLI. It doesn't talk to the GitHub API directly — if `gh` works, PRFlow works.

```
Human → TUI (Bubbletea) → PRFlow → gh CLI → GitHub API
                             ↓
                        SQLite Cache
```

**No tokens. No OAuth. No API keys.** Just `gh auth login` and go.

## License

MIT — see [LICENSE](LICENSE)
