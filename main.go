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
		case "askpass":
			// Internal subcommand invoked by ssh via SSH_ASKPASS.
			// Prints $TAMAGOSH_PASSPHRASE followed by newline to stdout.
			//
			// Weak guard: ssh sets SSH_ASKPASS to the path of the askpass
			// helper before exec. If a user invokes `tamagosh askpass`
			// directly from a shell, SSH_ASKPASS will be unset (or point
			// elsewhere) — refuse to print anything. Defense-in-depth only;
			// the real protection is process credentials (env is readable
			// only by the owning user).
			if os.Getenv("SSH_ASKPASS") == "" {
				fmt.Fprintln(os.Stderr, "askpass: internal subcommand, do not invoke directly")
				os.Exit(2)
			}
			fmt.Println(os.Getenv("TAMAGOSH_PASSPHRASE"))
			return
		case "uninstall":
			uninstall()
			return
		case "--help", "-h", "help":
			usage()
			return
		case "--version", "-v", "version":
			fmt.Println("tamagosh")
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
  tamagosh uninstall   remove config dir + binary (interactive)
  tamagosh help        this message
  tamagosh version     show name

env:
  TAMAGOSH_HOME        override default config dir (~/.config/tamagosh)
  EDITOR / VISUAL      editor for [e] key in SFTP browser`)
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
	// sshpass is only required for password-auth connections. Key auth works
	// without it. Warn instead of fail so key-only users aren't forced to
	// install it. SSH connect will emit a clear error if a password connection
	// is selected and sshpass isn't present.
	if _, err := exec.LookPath("sshpass"); err != nil {
		hint := "brew install hudochenkov/sshpass/sshpass  (macOS)"
		switch runtime.GOOS {
		case "linux":
			hint = "apt-get install sshpass  /  pacman -S sshpass  /  dnf install sshpass"
		}
		fmt.Fprintf(os.Stderr, "note: sshpass not installed — password auth disabled, key auth works (%s)\n", hint)
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
