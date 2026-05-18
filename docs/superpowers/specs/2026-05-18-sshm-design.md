# sshm — SSH Connection Manager TUI

## Overview

Terminal UI app for managing SSH connections. Built with Go + Bubbletea. Standalone config (not tied to `~/.ssh/config`). Passwords stored in `pass`. Includes built-in SFTP split-pane browser.

## Data Model

**Storage**: `~/.config/sshm/connections.json`

```json
{
  "connections": [
    {
      "name": "atlantic",
      "host": "43.228.213.209",
      "port": 2255,
      "user": "candra",
      "pass_key": "ssh/atlantic"
    }
  ]
}
```

`pass_key` is a pointer to the `pass` store entry. Passwords never stored in JSON.

## Architecture

```
sshm/
├── main.go
├── internal/
│   ├── config/       # read/write connections.json
│   ├── password/     # wrapper around pass CLI
│   ├── ssh/          # spawn sshpass + /usr/bin/ssh
│   ├── sftp/         # SFTP client via crypto/ssh + pkg/sftp
│   └── ui/
│       ├── app.go    # root model, view routing
│       ├── list.go   # connection list view
│       ├── form.go   # add/edit form view
│       └── sftp.go   # split pane SFTP view
└── go.mod
```

## Views

### 1. Connection List (default)

```
┌─ SSHM ──────────────────────────────┐
│  > atlantic   43.228.213.209  :2255  │
│    tencent    43.157.195.32   :22    │
│    paringin   10.0.7.210      :22    │
│    sawahlunto 192.168.181.119 :50171 │
│                                      │
│  [n]ew [e]dit [d]el [f]sftp [/]find │
└──────────────────────────────────────┘
```

Shortcuts:
- `Enter` — connect SSH
- `f` — open SFTP split pane
- `n` — new connection form
- `e` — edit selected connection
- `d` — delete (with confirmation)
- `/` — live search/filter

### 2. Add/Edit Form

```
┌─ New Connection ────────────────────┐
│  Name     : atlantic                │
│  Host     : 43.228.213.209          │
│  Port     : 2255                    │
│  User     : candra                  │
│  Password : ********                │
│                                     │
│  [Enter] Save   [Esc] Cancel        │
└─────────────────────────────────────┘
```

On save: password written to `pass ssh/<name>` automatically.

### 3. SFTP Split Pane

```
┌─ Local ──────────┬─ Remote ─────────┐
│  ~/projects      │  /home/fauziah   │
│  > app/          │  > uploads/      │
│    README.md     │    data.csv      │
│    config.json   │    logs/         │
│                  │                  │
│ [Tab] switch  [Space] select  [c] copy  [d] del  [q] back │
└──────────────────┴──────────────────┘
```

## Connect Flows

### SSH Connect
```
Enter pressed
→ retrieve password: pass show ssh/<name>
→ suspend TUI (bubbletea tea.ExecProcess)
→ exec: sshpass -p "<pass>" /usr/bin/ssh -p <port> <user>@<host>
→ SSH session runs in current terminal
→ user exits SSH
→ TUI resumes
```

### SFTP Connect
```
f pressed
→ retrieve password
→ open SSH connection via crypto/ssh (in-process)
→ open SFTP session on top of SSH connection
→ render split pane view
→ q pressed → close SFTP + SSH connection → return to list
```

## Dependencies

```
github.com/charmbracelet/bubbletea   # TUI framework
github.com/charmbracelet/bubbles     # list, textinput components
github.com/charmbracelet/lipgloss    # styling
golang.org/x/crypto/ssh              # SSH client for SFTP
github.com/pkg/sftp                  # SFTP protocol
```

## Error Handling

- `pass` not found → show error: "pass not installed, run: brew install pass"
- `sshpass` not found → show error: "sshpass not installed, run: brew install sshpass"
- Connection refused / timeout → show error inline in list view
- Wrong password → show error from SSH stderr
- SFTP permission denied → show inline error in SFTP pane

## Out of Scope

- Key-based auth (passwords only via pass)
- Jump hosts / ProxyJump
- Port forwarding
- Terminal emulator (SSH spawns in current terminal)
