package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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
	if _, err := exec.LookPath("ssh"); err != nil {
		return fmt.Errorf("ssh not found in PATH — install OpenSSH client")
	}
	if _, err := exec.LookPath("sshpass"); err != nil {
		hint := "brew install hudochenkov/sshpass/sshpass  (macOS)"
		switch runtime.GOOS {
		case "linux":
			hint = "apt-get install sshpass  /  pacman -S sshpass  /  dnf install sshpass"
		}
		return fmt.Errorf("sshpass not installed — %s", hint)
	}

	cfgPath, err := config.DefaultPath()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return fmt.Errorf("cannot create config dir %s: %w", cfgDir, err)
	}
	probe := filepath.Join(cfgDir, ".write-probe")
	if err := os.WriteFile(probe, []byte{}, 0o600); err != nil {
		return fmt.Errorf("config dir %s not writable: %w", cfgDir, err)
	}
	_ = os.Remove(probe)

	if os.Getenv("VISUAL") == "" && os.Getenv("EDITOR") == "" {
		found := ""
		for _, c := range []string{"nvim", "vim", "nano", "vi"} {
			if _, err := exec.LookPath(c); err == nil {
				found = c
				break
			}
		}
		shell := filepath.Base(os.Getenv("SHELL"))
		rc := "~/.bashrc"
		switch shell {
		case "zsh":
			rc = "~/.zshrc"
		case "fish":
			rc = "~/.config/fish/config.fish (use: set -Ux EDITOR nvim)"
		}
		fmt.Fprintln(os.Stderr, "warning: $EDITOR not set")
		if found != "" {
			fmt.Fprintf(os.Stderr, "  → using fallback: %s\n", found)
		} else {
			fmt.Fprintln(os.Stderr, "  → no editor in PATH — `e` key will fail")
		}
		fmt.Fprintf(os.Stderr, "  → set in shell rc: echo 'export EDITOR=nvim' >> %s\n", rc)
	}
	return nil
}
