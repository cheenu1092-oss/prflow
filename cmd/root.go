package cmd

import (
	"fmt"
	"os"

	"github.com/cheenu1092-oss/prflow/internal/ai"
	"github.com/cheenu1092-oss/prflow/internal/config"
	"github.com/cheenu1092-oss/prflow/internal/deps"
	"github.com/cheenu1092-oss/prflow/internal/tui"
)

func Execute() error {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Println("prflow v0.1.0")
			return nil
		case "setup":
			return tui.RunOnboarding()
		case "sync":
			return runSync()
		case "ls":
			return runList()
		case "config":
			return runConfig()
		case "doctor":
			return runDoctor()
		default:
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			printUsage()
			return nil
		}
	}

	// Default: launch TUI
	cfg, err := config.Load()
	if err != nil || len(cfg.Repos) == 0 {
		// First run — launch onboarding
		if err := tui.RunOnboarding(); err != nil {
			return err
		}
		// Reload config after onboarding
		cfg, err = config.Load()
		if err != nil || len(cfg.Repos) == 0 {
			fmt.Println("Setup complete. Run 'prflow' again to launch the dashboard.")
			return nil
		}
	}
	return tui.RunDashboard(cfg)
}

func runSync() error {
	fmt.Println("Syncing PRs...")
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("no config found, run 'prflow setup' first")
	}
	_ = cfg
	fmt.Println("Sync complete.")
	return nil
}

func runList() error {
	fmt.Println("prflow ls — quick list (TODO)")
	return nil
}

func runConfig() error {
	cfgPath := config.Path()
	fmt.Printf("Config: %s\n", cfgPath)
	fmt.Println("Open with: $EDITOR " + cfgPath)
	return nil
}

func runDoctor() error {
	fmt.Println(deps.PrintStatus())

	if ai.Available() {
		fmt.Println("🤖 AI features: ENABLED")
		fmt.Println("   Claude Code detected — PR analysis, review assistance, and auto-fix available.")
	} else {
		fmt.Println("🤖 AI features: DISABLED (optional)")
		fmt.Println("   Install Claude Code for AI-powered PR analysis:")
		fmt.Println("   npm install -g @anthropic-ai/claude-code")
		fmt.Println("   Then run: claude  (to complete auth)")
		fmt.Println("")
		fmt.Println("   Without it, PRFlow works as a standard PR dashboard.")
	}

	if err := deps.CheckRequired(); err != nil {
		fmt.Printf("\n⚠️  %v\n", err)
		return err
	}

	fmt.Println("\n✓ All required dependencies OK")
	return nil
}

func printUsage() {
	fmt.Println(`Usage: prflow [command]

Commands:
  (none)    Launch TUI dashboard
  setup     Run onboarding wizard
  sync      Force refresh PR cache
  ls        Quick list (no TUI)
  config    Show config path
  doctor    Check dependencies (gh, git, claude)
  version   Print version`)
}
