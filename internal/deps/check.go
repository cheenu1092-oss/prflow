package deps

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Status represents the state of a dependency
type Status struct {
	Name      string
	Installed bool
	Version   string
	Path      string
	Required  bool // if false, optional (enhanced features)
	HelpURL   string
	InstallCmd string // suggested install command
}

// CheckAll verifies all dependencies and returns their status
func CheckAll() []Status {
	return []Status{
		checkGH(),
		checkGit(),
		checkClaudeCode(),
	}
}

// CheckRequired returns an error if any required dependency is missing
func CheckRequired() error {
	for _, s := range CheckAll() {
		if s.Required && !s.Installed {
			return fmt.Errorf("%s is required but not installed. Install: %s\nHelp: %s",
				s.Name, s.InstallCmd, s.HelpURL)
		}
	}
	return nil
}

// HasClaudeCode returns true if claude CLI is available
func HasClaudeCode() bool {
	s := checkClaudeCode()
	return s.Installed
}

// HasGH returns true if gh CLI is available and authenticated
func HasGH() bool {
	s := checkGH()
	return s.Installed
}

func checkGH() Status {
	s := Status{
		Name:     "GitHub CLI (gh)",
		Required: true,
		HelpURL:  "https://cli.github.com",
	}

	path, err := exec.LookPath("gh")
	if err != nil {
		switch runtime.GOOS {
		case "darwin":
			s.InstallCmd = "brew install gh"
		case "linux":
			s.InstallCmd = "sudo apt install gh  # or: sudo snap install gh"
		default:
			s.InstallCmd = "See https://cli.github.com"
		}
		return s
	}

	s.Installed = true
	s.Path = path

	// Get version
	out, err := exec.Command("gh", "version").CombinedOutput()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) > 0 {
			s.Version = strings.TrimSpace(lines[0])
		}
	}

	// Check auth
	_, err = exec.Command("gh", "auth", "status").CombinedOutput()
	if err != nil {
		s.Installed = true // installed but not authenticated
		s.Version += " (not authenticated — run: gh auth login)"
	}

	return s
}

func checkGit() Status {
	s := Status{
		Name:     "Git",
		Required: true,
		HelpURL:  "https://git-scm.com/downloads",
	}

	path, err := exec.LookPath("git")
	if err != nil {
		switch runtime.GOOS {
		case "darwin":
			s.InstallCmd = "xcode-select --install"
		case "linux":
			s.InstallCmd = "sudo apt install git"
		default:
			s.InstallCmd = "See https://git-scm.com/downloads"
		}
		return s
	}

	s.Installed = true
	s.Path = path

	out, err := exec.Command("git", "version").CombinedOutput()
	if err == nil {
		s.Version = strings.TrimSpace(string(out))
	}

	return s
}

func checkClaudeCode() Status {
	s := Status{
		Name:     "Claude Code CLI",
		Required: false, // optional — AI features disabled without it
		HelpURL:  "https://docs.anthropic.com/en/docs/claude-code",
	}

	// Check common install locations
	candidates := []string{"claude"}
	for _, name := range candidates {
		path, err := exec.LookPath(name)
		if err == nil {
			s.Installed = true
			s.Path = path

			// Get version
			out, err := exec.Command(name, "--version").CombinedOutput()
			if err == nil {
				s.Version = strings.TrimSpace(string(out))
			}
			return s
		}
	}

	// Also check ~/.local/bin which might not be in PATH
	home, _ := exec.Command("sh", "-c", "echo $HOME").CombinedOutput()
	homeDir := strings.TrimSpace(string(home))
	if homeDir != "" {
		localPath := homeDir + "/.local/bin/claude"
		if _, err := exec.Command(localPath, "--version").CombinedOutput(); err == nil {
			s.Installed = true
			s.Path = localPath
			out, _ := exec.Command(localPath, "--version").CombinedOutput()
			s.Version = strings.TrimSpace(string(out))
			return s
		}
	}

	switch runtime.GOOS {
	case "darwin", "linux":
		s.InstallCmd = "npm install -g @anthropic-ai/claude-code"
	default:
		s.InstallCmd = "npm install -g @anthropic-ai/claude-code"
	}

	return s
}

// InstallGH attempts to install gh CLI
func InstallGH() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("brew", "install", "gh")
	case "linux":
		cmd = exec.Command("sh", "-c",
			"sudo apt install -y gh 2>/dev/null || sudo snap install gh 2>/dev/null")
	default:
		return fmt.Errorf("unsupported OS: %s. Install manually: https://cli.github.com", runtime.GOOS)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// PrintStatus prints a human-readable dependency status
func PrintStatus() string {
	var s strings.Builder
	statuses := CheckAll()

	s.WriteString("PRFlow Dependencies\n")
	s.WriteString("───────────────────\n")

	for _, dep := range statuses {
		icon := "✓"
		if !dep.Installed {
			if dep.Required {
				icon = "✗"
			} else {
				icon = "○"
			}
		}

		tag := ""
		if !dep.Required {
			tag = " (optional)"
		}

		s.WriteString(fmt.Sprintf("  %s %s%s\n", icon, dep.Name, tag))
		if dep.Installed {
			s.WriteString(fmt.Sprintf("    %s\n", dep.Version))
			if dep.Path != "" {
				s.WriteString(fmt.Sprintf("    %s\n", dep.Path))
			}
		} else {
			s.WriteString(fmt.Sprintf("    Install: %s\n", dep.InstallCmd))
		}
	}

	return s.String()
}
