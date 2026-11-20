// Command issuer is a terminal UI for browsing and managing issues across
// issue trackers. GitHub is supported first.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ljunker/issuer/internal/auth"
	"github.com/ljunker/issuer/internal/config"
	"github.com/ljunker/issuer/internal/provider/github"
	"github.com/ljunker/issuer/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "issuer:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	token, err := auth.GitHubToken()
	if err != nil {
		return err
	}
	p, err := github.New(token, os.Getenv("ISSUER_GITHUB_API_URL"))
	if err != nil {
		return err
	}

	// Detect the background before Bubble Tea takes over the terminal.
	style := "light"
	if lipgloss.HasDarkBackground() {
		style = "dark"
	}

	prog := tea.NewProgram(ui.New(p, cfg, style), tea.WithAltScreen())
	_, err = prog.Run()
	return err
}
