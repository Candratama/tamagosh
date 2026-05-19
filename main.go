package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Candratama/tamagosh/internal/bookmark"
	"github.com/Candratama/tamagosh/internal/config"
	"github.com/Candratama/tamagosh/internal/secret"
	"github.com/Candratama/tamagosh/internal/ui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "uninstall":
			uninstall()
			return
		case "--help", "-h", "help":
			usage()
			return
		case "--version", "-v", "version":
			fmt.Println("tamagosh")
			return
		case "--demo", "demo":
			runDemo()
			return
		}
	}
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

func usage() {
	fmt.Println(`tamagosh — SSH/SFTP terminal manager

usage:
  tamagosh             launch TUI
  tamagosh demo        launch with fake connections (for demos/screenshots)
  tamagosh uninstall   remove config dir + binary (interactive)
  tamagosh help        this message
  tamagosh version     show name

env:
  TAMAGOSH_HOME        override default config dir (~/.config/tamagosh)
  EDITOR / VISUAL      editor for [e] key in SFTP browser`)
}

type demoPass struct{}

func (demoPass) Get(string) (string, error) { return "demo", nil }
func (demoPass) Set(string, string) error   { return nil }
func (demoPass) Delete(string) error        { return nil }

func runDemo() {
	os.Setenv("TAMAGOSH_DEMO", "1")
	// RFC 5737 documentation IP ranges — safe to publish: 192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24
	store := &config.Store{Connections: []config.Connection{
		{Name: "digitalocean-sgp1", Host: "198.51.100.42", Port: 22, User: "root", PassKey: "ssh/digitalocean-sgp1"},
		{Name: "aws-ec2-tokyo", Host: "203.0.113.15", Port: 22, User: "ubuntu", PassKey: "ssh/aws-ec2-tokyo"},
		{Name: "vultr-singapore", Host: "192.0.2.88", Port: 22, User: "root", PassKey: "ssh/vultr-singapore"},
		{Name: "linode-frankfurt", Host: "198.51.100.171", Port: 2222, User: "admin", PassKey: "ssh/linode-frankfurt"},
		{Name: "hetzner-helsinki", Host: "203.0.113.99", Port: 22, User: "root", PassKey: "ssh/hetzner-helsinki"},
		{Name: "fly-iad", Host: "192.0.2.7", Port: 22, User: "deploy", PassKey: "ssh/fly-iad"},
		{Name: "scaleway-paris", Host: "203.0.113.221", Port: 22, User: "root", PassKey: "ssh/scaleway-paris"},
	}}

	tmp, err := os.MkdirTemp("", "tamagosh-demo-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "demo: tmp dir:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	bm, err := bookmark.New(tmp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "demo: bookmark:", err)
		os.Exit(1)
	}

	app := ui.NewApp(store, demoPass{}, bm, filepath.Join(tmp, "connections.json"))
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tea:", err)
		os.Exit(1)
	}
}

func uninstall() {
	cfgPath, _ := config.DefaultPath()
	cfgDir := filepath.Dir(cfgPath)
	exe, _ := os.Executable()

	fmt.Println("tamagosh uninstall — review what will be removed:")
	fmt.Printf("  config dir: %s\n", cfgDir)
	if exe != "" {
		fmt.Printf("  binary:     %s\n", exe)
	}
	fmt.Println()
	fmt.Println("warning: removing config dir wipes the AES key —")
	fmt.Println("         all saved passwords become unrecoverable.")
	fmt.Print("\nproceed? [y/N] ")

	var ans string
	_, _ = fmt.Scanln(&ans)
	ans = strings.ToLower(strings.TrimSpace(ans))
	if ans != "y" && ans != "yes" {
		fmt.Println("aborted")
		return
	}

	if err := os.RemoveAll(cfgDir); err != nil {
		fmt.Fprintln(os.Stderr, "remove config:", err)
	} else {
		fmt.Println("removed config dir")
	}

	if exe != "" {
		if err := os.Remove(exe); err != nil {
			fmt.Fprintln(os.Stderr, "remove binary:", err)
			fmt.Println("manual: rm", exe)
		} else {
			fmt.Println("removed binary")
		}
	}
	fmt.Println("done")
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
