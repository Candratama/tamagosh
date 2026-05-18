package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Candratama/tamagosh/internal/bookmark"
	"github.com/Candratama/tamagosh/internal/config"
	"github.com/Candratama/tamagosh/internal/secret"
	"github.com/Candratama/tamagosh/internal/ui"
)

func main() {
	if err := preflight(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	path, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "home dir:", err)
		os.Exit(1)
	}
	store, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load config:", err)
		os.Exit(1)
	}

	dir := filepath.Dir(path)
	sec, err := secret.New(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "secret:", err)
		os.Exit(1)
	}

	bm, err := bookmark.New(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bookmark:", err)
		os.Exit(1)
	}

	app := ui.NewApp(store, sec, bm, path)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tea:", err)
		os.Exit(1)
	}
}

func preflight() error {
	if _, err := exec.LookPath("sshpass"); err != nil {
		return fmt.Errorf("sshpass not installed, run: brew install hudochenkov/sshpass/sshpass")
	}
	return nil
}
