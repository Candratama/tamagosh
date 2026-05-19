# TAMAGOSH 🐣

A cozy little terminal SSH manager with a built-in SFTP browser. Pet your servers, don't lose them.

Tamagosh keeps your SSH connections in one place, opens them with a single keypress, and includes a dual-pane SFTP browser for moving files around without leaving the terminal. Passwords are encrypted locally with AES-GCM — no `pass`, no GPG, no system keyring required.

---

## Preview

**Connection list** — browse, filter, hit Enter to SSH

![connection list demo](docs/vhs/list.gif)

**SSH connect** — one keypress to a live shell

![ssh demo](docs/vhs/ssh.gif)

**Add / edit form** — fill in name/host/port/user/password, Tab to move

![form demo](docs/vhs/form.gif)

**SFTP browser** — dual-pane, navigate, help overlay, file info

![sftp demo](docs/vhs/sftp.gif)

---



## What you get

- **Connection list** — name, host, port, user. Add / edit / delete with one key.
- **One-press SSH connect** — `Enter` and you're in. Password injected via `sshpass`.
- **Built-in SFTP browser** — dual-pane (local ↔ remote), Norton-style.
  - Copy files **and** whole directories (recursive, with a real byte-level progress bar).
  - Multi-select, delete, rename, mkdir, chmod, goto-path, sort, hidden toggle, in-pane search.
  - Bookmarks per connection.
  - Open any file in your `$EDITOR` — remote files are auto-downloaded, edited, and re-uploaded if they changed.
- **Mouse support** — click to focus, double-click to open folders, wheel to scroll.
- **Gruvbox material dark hard** palette. Easy on the eyes.
- **Encrypted local secrets** — AES-256-GCM, key in `~/.config/tamagosh/key` (mode 0600).
- **Auto-migration** from old `~/.config/sshm` if you ever used the original name.
- **Single static binary** — no Python, no Node, no runtime, no surprises.

---

## Install

You need **Go 1.22+** and the `sshpass` binary.

```bash
# 1. install sshpass (macOS)
brew install hudochenkov/sshpass/sshpass

# 1. install sshpass (Linux)
sudo apt install sshpass        # Debian/Ubuntu
sudo pacman -S sshpass          # Arch
sudo dnf install sshpass        # Fedora

# 2. install tamagosh
go install github.com/Candratama/tamagosh@latest
```

Make sure `~/go/bin` is in your `PATH`:

```bash
# add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/go/bin:$PATH"
```

Then just run:

```bash
tamagosh
```

That's it. First run creates `~/.config/tamagosh/` automatically with a fresh encryption key.

### Updating

```bash
go install github.com/Candratama/tamagosh@latest
```

If the Go module proxy is slow to pick up new tags (sometimes takes a few minutes), pin the version:

```bash
go install github.com/Candratama/tamagosh@v0.4.0
```

---

## First run

1. Launch `tamagosh`.
2. Press **`n`** to add a new connection.
3. Fill in: name / host / port / user / password — Tab to move, Enter to save.
4. Highlight the connection, hit **`Enter`** to SSH in, or **`f`** to open the SFTP browser.

Passwords are encrypted with the AES key in `~/.config/tamagosh/key` and stored in `~/.config/tamagosh/secrets.json`. **Never edit those files by hand.** Don't commit them either.

---

## Keyboard reference

### Connection list

| Key | Action |
|---|---|
| `Enter` | SSH into selected |
| `f` | open SFTP browser |
| `n` | new connection |
| `e` | edit selected |
| `d` | delete selected (with confirmation) |
| `/` | filter by name/host |
| `q` | quit |
| wheel | scroll cursor |

### SFTP browser

| Key | Action |
|---|---|
| `Tab` | switch active pane |
| `→` / `Enter` | open folder under cursor |
| `←` | back to previous folder (per-pane history) |
| `Bksp` | parent folder |
| `↑ ↓` | move cursor |
| `PgUp / PgDn` | page through entries |
| `Home / End` | first / last entry |
| `Space` | toggle select (files only) |
| `a` / `A` | select all visible files / clear selection |
| `c` | copy cursor item or all selected (multi-select prompts confirm; directories recurse) |
| `d` | delete (always prompts confirm) |
| `R` | rename cursor item |
| `m` | mkdir (prompts for name) |
| `e` | open cursor file in `$EDITOR` (remote: downloads → edits → re-uploads on save) |
| `x` | chmod (prompts for octal mode, e.g. `644`) |
| `g` | goto path (jump directly without navigating) |
| `b` | bookmark current dir |
| `'` | open bookmark list |
| `s` | cycle sort: name → size → mtime |
| `S` | toggle sort direction |
| `i` | show file info (size, mtime, full path) |
| `/` | filter pane by filename |
| `.` | toggle hidden files |
| `r` | refresh both panes |
| `h` or `?` | full keyboard reference (overlay) |
| `q` | back to connection list |
| `..` | navigate up (appears as first entry when not at root) |

### Mouse

| Gesture | Action |
|---|---|
| left-click on entry | focus that pane + move cursor |
| left-click on title/border | switch focus to that pane |
| double-left-click | open folder under cursor |
| right-click on file | toggle select |
| wheel up / down | scroll cursor in active pane |

In tmux, enable mouse forwarding: `set -g mouse on` in `~/.tmux.conf`.

### Confirm/prompt dialogs

| Key | Action |
|---|---|
| `y` / `Y` | confirm |
| `n` / `N` / `Esc` | cancel |
| `Enter` | submit prompt input |

---

## Editor integration (the `e` key)

Picks the first available editor in this order:

1. `$VISUAL` (if set)
2. `$EDITOR` (if set)
3. `nvim`
4. `vim`
5. `nano`
6. `vi`

Want a specific one? Add to your shell rc:

```bash
echo 'export EDITOR=nvim' >> ~/.zshrc
```

For **remote files**, tamagosh:
1. Downloads to `/tmp/tamagosh-edit-XXXXX/`
2. Opens your editor
3. On save → uploads back to the server
4. On exit without saving → skips upload (no false alerts)
5. Cleans up the temp file

If upload fails (permission denied, network drop), the local edits are **preserved** at the temp path so you can recover them manually.

---

## Configuration location

By default everything lives in `~/.config/tamagosh/`:

```
~/.config/tamagosh/
├── connections.json    # host/user/port (no passwords)
├── secrets.json        # AES-GCM encrypted passwords
├── key                 # 32-byte AES key (mode 0600)
└── bookmarks.json      # SFTP bookmarks per connection
```

Want it somewhere else? Set `TAMAGOSH_HOME`:

```bash
export TAMAGOSH_HOME=~/Dropbox/tamagosh
tamagosh
```

Useful for syncing connections across machines (just don't sync `key` over an untrusted channel).

---

## Uninstall

```bash
tamagosh uninstall
```

That's it. The subcommand:

- Removes the config directory (with a confirmation prompt — including a warning that wiping the key makes saved passwords unrecoverable).
- Self-deletes the binary at `$GOBIN/tamagosh` (or `~/go/bin/tamagosh`).

If you prefer manual:

```bash
rm "$(go env GOBIN)/tamagosh" 2>/dev/null || rm ~/go/bin/tamagosh
rm -rf ~/.config/tamagosh        # or "$TAMAGOSH_HOME"

# alternative: go's own cleanup
go clean -i github.com/Candratama/tamagosh
```

---

## CLI commands

```
tamagosh             launch the TUI
tamagosh uninstall   remove config dir + binary
tamagosh help        show usage
tamagosh version     show name
```

---

## Troubleshooting

**"sshpass not installed"**
Install it — see the [Install](#install) section. macOS uses a custom tap because the formula was removed from core for security reasons.

**"ssh not found in PATH"**
Install OpenSSH client. macOS has it built in; on Linux: `sudo apt install openssh-client` (or your distro's equivalent).

**Mouse clicks do nothing**
- In tmux? Add `set -g mouse on` to `~/.tmux.conf` and reload (`tmux source ~/.tmux.conf`).
- In iTerm2 / Ghostty? Plain click forwards to the app. Hold Option to select text instead.

**Editor opens but shows a brief flash of the underlying terminal**
Known cosmetic glitch inherent to alt-screen TUIs running subprocesses. v0.3.7+ bridges the gap by briefly showing the SFTP pane in the main buffer before the editor starts. Not perfect but much smoother.

**Lost the AES key**
Saved passwords are unrecoverable. Re-add connections and re-enter passwords.

**Updated but the version didn't change**
Go module proxy caches `@latest`. Pin explicitly: `go install github.com/Candratama/tamagosh@v0.4.0`. Or bypass the proxy: `GOPROXY=direct go install github.com/Candratama/tamagosh@latest`.

---

## Security notes

- Passwords are encrypted with **AES-256-GCM** using a 32-byte key generated on first run.
- Key file (`key`) is created with mode `0600` (owner read/write only).
- Secrets file (`secrets.json`) is also `0600`.
- The key is **local-only** — never transmitted, never synced. If you sync `TAMAGOSH_HOME` across machines, sync the key over a trusted channel (e.g., encrypted cloud storage), or re-add connections per machine.
- SSH host key verification: `StrictHostKeyChecking=accept-new` — first connection auto-accepts, subsequent connections verify. Tamagosh doesn't manage `known_hosts` itself; it delegates to your OpenSSH client.
- SFTP host key: currently uses `InsecureIgnoreHostKey()` for the in-process SFTP session. **Don't use over untrusted networks.** This will tighten up in a future release.

---

## What's missing / on the roadmap

Things that would be nice but aren't built yet:

- **Sync directories** — mirror local ↔ remote with diff
- **Resume partial transfer** — pick up where a dropped upload left off
- **Parallel transfers** — copy multiple files concurrently
- **Background transfer queue** — keep using the TUI while uploads run
- **Image / file preview** — Kitty graphics protocol, like yazi
- **Bulk rename via editor**
- **Key-based SSH auth** — currently password-only
- **Jump hosts / ProxyJump**
- **Port forwarding**
- **Windows support** — needs `sshpass` replacement (in-process SSH PTY)

If any of these are blockers for you, open an issue.

---

## Contributing

Tamagosh is a personal project that grew. PRs welcome, but please:

- Keep changes small and focused.
- Run `go test ./...` and `go vet ./...` before submitting.
- The code is intentionally minimal — favour deleting code over adding it.

---

## Credits

Tamagosh stands on the shoulders of these excellent libraries:

**TUI framework & rendering**
- [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea) — Elm-inspired TUI framework. The event loop, alt-screen handling, mouse support, and `tea.ExecProcess` all come from here.
- [`charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss) — Style and layout primitives (borders, padding, alignment, color). Almost every visible character is styled via lipgloss.
- [`charmbracelet/x/ansi`](https://github.com/charmbracelet/x) — ANSI-aware string utilities. Used for the transparent overlay splicing that keeps pane borders visible behind popups.
- [`charmbracelet/bubbles`](https://github.com/charmbracelet/bubbles) — Reusable component library (transitively).

**SSH & SFTP**
- [`golang.org/x/crypto`](https://golang.org/x/crypto) — Official Go SSH client. Powers the in-process SFTP session.
- [`pkg/sftp`](https://github.com/pkg/sftp) — Pure-Go SFTP client/server. All remote file ops (read/write/stat/walk/chmod/rename/mkdir) ride on this.
- [`sshpass`](https://sourceforge.net/projects/sshpass/) — Non-interactive password injection for `ssh`. External binary, not a Go dep, but the whole "one-press connect" UX would fall apart without it.

**Design**
- [Gruvbox Material Dark Hard](https://github.com/sainnhe/gruvbox-material) by [sainnhe](https://github.com/sainnhe) — color palette.
- Inspired by classic dual-pane file managers (Midnight Commander, Norton Commander) and modern terminal file managers (yazi, ranger, lf).

Plus the indirect dependencies that make the above work — see [`go.mod`](./go.mod) for the full list with versions.

---

## License

MIT. Do what you want, but don't blame me if you `rm -rf` your home directory through the SFTP browser. (You won't — there's confirmation dialogs and path-traversal guards. But, you know.)
