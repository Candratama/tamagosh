package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/candratama/sshm/internal/config"
	"github.com/candratama/sshm/internal/password"
	"github.com/candratama/sshm/internal/ui"
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

	app := ui.NewApp(store, password.New(), path)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tea:", err)
		os.Exit(1)
	}
}

func preflight() error {
	if _, err := exec.LookPath("pass"); err != nil {
		return fmt.Errorf("pass not installed, run: brew install pass")
	}
	if _, err := exec.LookPath("sshpass"); err != nil {
		return fmt.Errorf("sshpass not installed, run: brew install hudochenkov/sshpass/sshpass")
	}
	return nil
}
