package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nagarjun226/prflow/internal/config"
	"github.com/nagarjun226/prflow/internal/deps"
	"github.com/nagarjun226/prflow/internal/gh"
)

type onboardStep int

const (
	stepCheckGH onboardStep = iota
	stepInstallGH
	stepLoginGH
	stepScanRepos
	stepSelectRepos
	stepSelectFavorites
	stepDone
)

type onboardModel struct {
	step     onboardStep
	username string
	authErr  string
	scanErr  string

	// gh status
	ghInstalled bool
	ghLoggedIn  bool

	// Repos found
	allRepos []repoItem
	cursor   int

	// For favorites step
	favCursor int

	// Spinner
	spinner    int
	spinFrames []string

	width  int
	height int
}

type repoItem struct {
	name     string
	selected bool
	starred  bool
	prCount  int
}

type authCheckMsg struct {
	installed bool
	username  string
	err       error
}

type ghInstalledMsg struct{ err error }
type ghLoginDoneMsg struct {
	username string
	err      error
}

type repoScanMsg struct {
	repos []string
	prs   map[string]int
	err   error
}

type tickMsg time.Time

func RunOnboarding() error {
	m := onboardModel{
		step:       stepCheckGH,
		spinFrames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m onboardModel) Init() tea.Cmd {
	return tea.Batch(checkGH, tickCmd())
}

func checkGH() tea.Msg {
	// Check if gh is installed
	_, err := exec.LookPath("gh")
	if err != nil {
		return authCheckMsg{installed: false, err: fmt.Errorf("gh CLI not found")}
	}

	// Check if authenticated
	username, err := gh.CheckAuth()
	if err != nil {
		return authCheckMsg{installed: true, err: err}
	}
	return authCheckMsg{installed: true, username: username}
}

func installGH() tea.Msg {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("brew", "install", "gh")
	case "linux":
		// Try apt first, then snap
		cmd = exec.Command("sh", "-c", "sudo apt install -y gh 2>/dev/null || sudo snap install gh 2>/dev/null || (curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && echo 'deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main' | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null && sudo apt update && sudo apt install gh -y)")
	default:
		return ghInstalledMsg{err: fmt.Errorf("unsupported OS: %s. Install gh manually: https://cli.github.com", runtime.GOOS)}
	}
	err := cmd.Run()
	return ghInstalledMsg{err: err}
}

func loginGH() tea.Msg {
	// We can't run interactive gh auth login inside Bubbletea's alt screen.
	// Instead, try web-based auth which opens browser
	cmd := exec.Command("gh", "auth", "login", "--web", "--git-protocol", "https")
	cmd.Stdin = nil
	err := cmd.Run()
	if err != nil {
		return ghLoginDoneMsg{err: err}
	}
	// Verify login worked
	username, err := gh.CheckAuth()
	return ghLoginDoneMsg{username: username, err: err}
}

func scanRepos() tea.Msg {
	repoMap := make(map[string]int)

	// First: get repos from user's open PRs (fast, targeted)
	prs, err := gh.SearchMyPRs()
	if err != nil {
		// If search fails, try listing repos instead
		repos, listErr := gh.ListUserRepos()
		if listErr != nil {
			return repoScanMsg{err: fmt.Errorf("scan failed: %v", err)}
		}
		for _, r := range repos {
			repoMap[r] = 0
		}
	} else {
		for _, pr := range prs {
			repoMap[pr.Repository.NameWithOwner]++
		}
	}

	// Also get review-requested PRs (finds repos you contribute to but don't own)
	reviewPRs, _ := gh.SearchReviewRequests()
	for _, pr := range reviewPRs {
		if _, exists := repoMap[pr.Repository.NameWithOwner]; !exists {
			repoMap[pr.Repository.NameWithOwner] = 0
		}
	}

	var repoNames []string
	for name := range repoMap {
		repoNames = append(repoNames, name)
	}

	// If we found nothing from PRs, fall back to listing repos (but limit it)
	if len(repoNames) == 0 {
		repos, _ := gh.ListUserRepos()
		for _, r := range repos {
			repoNames = append(repoNames, r)
			repoMap[r] = 0
		}
	}

	return repoScanMsg{repos: repoNames, prs: repoMap}
}

func (m onboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.step != stepSelectRepos && m.step != stepSelectFavorites {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.spinner = (m.spinner + 1) % len(m.spinFrames)
		return m, tickCmd()
	}

	switch m.step {
	case stepCheckGH:
		return m.updateCheckGH(msg)
	case stepInstallGH:
		return m.updateInstallGH(msg)
	case stepLoginGH:
		return m.updateLoginGH(msg)
	case stepScanRepos:
		return m.updateScan(msg)
	case stepSelectRepos:
		return m.updateSelectRepos(msg)
	case stepSelectFavorites:
		return m.updateSelectFavorites(msg)
	case stepDone:
		return m.updateDone(msg)
	}

	return m, nil
}

func (m onboardModel) updateCheckGH(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case authCheckMsg:
		m.ghInstalled = msg.installed
		if !msg.installed {
			// gh not installed — offer to install
			m.step = stepInstallGH
			return m, nil
		}
		if msg.err != nil {
			// gh installed but not logged in
			m.ghLoggedIn = false
			m.step = stepLoginGH
			return m, nil
		}
		// All good
		m.ghLoggedIn = true
		m.username = msg.username
		m.step = stepScanRepos
		return m, scanRepos
	}
	return m, nil
}

func (m onboardModel) updateInstallGH(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			m.authErr = "installing..."
			return m, installGH
		case "n", "N":
			m.authErr = "gh CLI required. Install manually: https://cli.github.com"
			return m, nil
		}
	case ghInstalledMsg:
		if msg.err != nil {
			m.authErr = fmt.Sprintf("Install failed: %v\nInstall manually: https://cli.github.com", msg.err)
			return m, nil
		}
		m.ghInstalled = true
		m.authErr = ""
		m.step = stepLoginGH
		return m, nil
	}
	return m, nil
}

func (m onboardModel) updateLoginGH(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.authErr = "opening browser for GitHub login..."
			return m, loginGH
		}
	case ghLoginDoneMsg:
		if msg.err != nil {
			m.authErr = fmt.Sprintf("Login failed: %v\nTry manually: gh auth login", msg.err)
			return m, nil
		}
		m.ghLoggedIn = true
		m.username = msg.username
		m.authErr = ""
		m.step = stepScanRepos
		return m, scanRepos
	}
	return m, nil
}

func (m onboardModel) updateScan(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case repoScanMsg:
		if msg.err != nil {
			m.scanErr = fmt.Sprintf("Scan failed: %v\n\nPress [r] to retry or [m] to add repos manually.", msg.err)
			return m, nil
		}
		if len(msg.repos) == 0 {
			m.scanErr = "No repos found. Press [m] to add repos manually."
			return m, nil
		}
		for _, name := range msg.repos {
			prCount := msg.prs[name]
			item := repoItem{name: name, selected: prCount > 0, prCount: prCount}
			m.allRepos = append(m.allRepos, item)
		}
		// Sort: repos with PRs first
		m.sortRepos()
		m.step = stepSelectRepos
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.scanErr = ""
			return m, scanRepos
		case "m":
			m.step = stepSelectRepos
			return m, nil
		}
	}
	return m, nil
}

func (m *onboardModel) sortRepos() {
	// Simple bubble sort: repos with PRs first, then alphabetical
	for i := 0; i < len(m.allRepos); i++ {
		for j := i + 1; j < len(m.allRepos); j++ {
			swap := false
			if m.allRepos[i].prCount == 0 && m.allRepos[j].prCount > 0 {
				swap = true
			} else if m.allRepos[i].prCount == m.allRepos[j].prCount && m.allRepos[i].name > m.allRepos[j].name {
				swap = true
			}
			if swap {
				m.allRepos[i], m.allRepos[j] = m.allRepos[j], m.allRepos[i]
			}
		}
	}
}

func (m onboardModel) updateSelectRepos(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.allRepos)-1 {
				m.cursor++
			}
		case " ":
			if len(m.allRepos) > 0 {
				m.allRepos[m.cursor].selected = !m.allRepos[m.cursor].selected
			}
		case "a":
			allSelected := true
			for _, r := range m.allRepos {
				if !r.selected {
					allSelected = false
					break
				}
			}
			for i := range m.allRepos {
				m.allRepos[i].selected = !allSelected
			}
		case "enter":
			selected := m.selectedRepos()
			if len(selected) == 0 {
				// Must select at least one
				return m, nil
			}
			m.step = stepSelectFavorites
			m.favCursor = 0
			return m, nil
		}
	}
	return m, nil
}

func (m onboardModel) updateSelectFavorites(msg tea.Msg) (tea.Model, tea.Cmd) {
	selected := m.selectedRepos()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.favCursor > 0 {
				m.favCursor--
			}
		case "down", "j":
			if m.favCursor < len(selected)-1 {
				m.favCursor++
			}
		case " ":
			if m.favCursor < len(selected) {
				name := selected[m.favCursor]
				for i := range m.allRepos {
					if m.allRepos[i].name == name {
						m.allRepos[i].starred = !m.allRepos[i].starred
						break
					}
				}
			}
		case "enter":
			m.saveConfig()
			m.step = stepDone
			return m, nil
		case "s":
			// Skip favorites
			m.saveConfig()
			m.step = stepDone
			return m, nil
		}
	}
	return m, nil
}

func (m onboardModel) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m onboardModel) selectedRepos() []string {
	var selected []string
	for _, r := range m.allRepos {
		if r.selected {
			selected = append(selected, r.name)
		}
	}
	return selected
}

func (m onboardModel) starredRepos() []string {
	var starred []string
	for _, r := range m.allRepos {
		if r.starred {
			starred = append(starred, r.name)
		}
	}
	return starred
}

func (m onboardModel) saveConfig() {
	cfg := config.DefaultConfig()
	cfg.Repos = m.selectedRepos()
	cfg.Favorites = m.starredRepos()
	config.Save(cfg)
}

func (m onboardModel) spin() string {
	return m.spinFrames[m.spinner%len(m.spinFrames)]
}

func (m onboardModel) View() string {
	var s strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Padding(1, 0).
		Render("  Welcome to PRFlow! ⚡")

	s.WriteString(title + "\n\n")

	switch m.step {
	case stepCheckGH:
		s.WriteString(m.viewCheckGH())
	case stepInstallGH:
		s.WriteString(m.viewInstallGH())
	case stepLoginGH:
		s.WriteString(m.viewLoginGH())
	case stepScanRepos:
		s.WriteString(m.viewScan())
	case stepSelectRepos:
		s.WriteString(m.viewSelectRepos())
	case stepSelectFavorites:
		s.WriteString(m.viewSelectFavorites())
	case stepDone:
		s.WriteString(m.viewDone())
	}

	return s.String()
}

func (m onboardModel) viewCheckGH() string {
	return fmt.Sprintf("  %s Checking prerequisites...\n", m.spin())
}

func (m onboardModel) viewInstallGH() string {
	var s strings.Builder
	s.WriteString("  Step 1: GitHub CLI Setup\n\n")

	if m.authErr == "installing..." {
		s.WriteString(fmt.Sprintf("  %s Installing gh CLI...\n", m.spin()))
	} else if m.authErr != "" {
		s.WriteString(fmt.Sprintf("  ✗ %s\n", m.authErr))
	} else {
		s.WriteString("  ✗ GitHub CLI (gh) not found.\n\n")
		s.WriteString("  PRFlow needs gh to talk to GitHub.\n")
		s.WriteString("  Install it now?\n\n")
		switch runtime.GOOS {
		case "darwin":
			s.WriteString("  Will run: brew install gh\n\n")
		case "linux":
			s.WriteString("  Will run: apt/snap install gh\n\n")
		}
		s.WriteString(fmt.Sprintf("  %s\n", helpStyle.Render("[y] install · [n] skip (install manually)")))
	}
	return s.String()
}

func (m onboardModel) viewLoginGH() string {
	var s strings.Builder
	s.WriteString("  Step 1: GitHub Authentication\n\n")

	if m.authErr != "" && m.authErr != "opening browser for GitHub login..." {
		s.WriteString(fmt.Sprintf("  ✗ %s\n\n", m.authErr))
		s.WriteString(fmt.Sprintf("  %s\n", helpStyle.Render("[enter] retry · [q] quit")))
	} else if m.authErr == "opening browser for GitHub login..." {
		s.WriteString(fmt.Sprintf("  %s Opening browser for GitHub login...\n", m.spin()))
		s.WriteString("  Complete the authentication in your browser.\n")
	} else {
		s.WriteString("  ✓ gh CLI installed\n")
		s.WriteString("  ✗ Not logged in to GitHub\n\n")
		s.WriteString("  Press [enter] to open GitHub login in your browser.\n\n")
		s.WriteString(fmt.Sprintf("  %s\n", helpStyle.Render("[enter] login · [q] quit")))
	}
	return s.String()
}

func (m onboardModel) viewScan() string {
	if m.scanErr != "" {
		return fmt.Sprintf("  Step 2: Scan Repos\n\n  %s\n\n  %s\n",
			m.scanErr,
			helpStyle.Render("[r] retry · [m] manual · [q] quit"))
	}
	return fmt.Sprintf("  Step 2: Scan Repos\n\n  %s Scanning your GitHub activity...\n  (fetching open PRs and review requests)\n", m.spin())
}

func (m onboardModel) viewSelectRepos() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("  Step 3: Select Repos (%d found)\n\n", len(m.allRepos)))

	if len(m.allRepos) == 0 {
		s.WriteString("  No repos found. Add repos manually in config:\n")
		s.WriteString(fmt.Sprintf("  %s\n", config.Path()))
		return s.String()
	}

	maxShow := m.height - 12
	if maxShow < 5 {
		maxShow = 15
	}
	start := 0
	if m.cursor >= maxShow {
		start = m.cursor - maxShow + 1
	}

	for i := start; i < len(m.allRepos) && i < start+maxShow; i++ {
		r := m.allRepos[i]
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}
		check := "[ ]"
		if r.selected {
			check = "[x]"
		}
		prInfo := ""
		if r.prCount > 0 {
			prInfo = fmt.Sprintf("  (%d open PRs)", r.prCount)
		}
		s.WriteString(fmt.Sprintf("  %s%s %s%s\n", cursor, check, r.name,
			repoStyle.Render(prInfo)))
	}

	if len(m.allRepos) > maxShow {
		s.WriteString(fmt.Sprintf("\n  ... %d more (scroll with ↑↓)\n", len(m.allRepos)-maxShow))
	}

	s.WriteString(fmt.Sprintf("\n  %s\n",
		helpStyle.Render("[space] toggle · [a] select all · [enter] next · [q] quit")))
	return s.String()
}

func (m onboardModel) viewSelectFavorites() string {
	var s strings.Builder
	selected := m.selectedRepos()
	s.WriteString(fmt.Sprintf("  Step 4: Star your favorites (%d repos selected)\n\n", len(selected)))
	s.WriteString("  ★ Favorites get detailed tracking in the sidebar.\n\n")

	maxShow := m.height - 12
	if maxShow < 5 {
		maxShow = 15
	}

	for i, name := range selected {
		if i >= maxShow {
			s.WriteString(fmt.Sprintf("  ... %d more\n", len(selected)-maxShow))
			break
		}
		cursor := "  "
		if i == m.favCursor {
			cursor = "▸ "
		}
		star := "  "
		for _, r := range m.allRepos {
			if r.name == name && r.starred {
				star = favHeaderStyle.Render("★ ")
				break
			}
		}
		s.WriteString(fmt.Sprintf("  %s%s%s\n", cursor, star, name))
	}

	s.WriteString(fmt.Sprintf("\n  %s\n",
		helpStyle.Render("[space] toggle star · [enter] finish · [s] skip favorites · [q] quit")))
	return s.String()
}

func (m onboardModel) viewDone() string {
	selected := m.selectedRepos()
	starred := m.starredRepos()

	aiStatus := "  ○ Claude Code not found (AI features disabled)\n    Install: npm install -g @anthropic-ai/claude-code\n"
	if deps.HasClaudeCode() {
		aiStatus = "  ✓ Claude Code detected — AI features enabled!\n"
	}

	return fmt.Sprintf(`  Setup Complete! ✓

  ✓ Authenticated as @%s
  ✓ Config saved to %s
  ✓ Tracking %d repos (%d favorites)
%s
  %s
`,
		m.username,
		config.Path(),
		len(selected),
		len(starred),
		aiStatus,
		helpStyle.Render("[enter] launch PRFlow!"))
}
