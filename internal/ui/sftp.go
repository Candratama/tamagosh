package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sftppkg "github.com/candratama/sshm/internal/sftp"
)

type Pane int

const (
	PaneLocal Pane = iota
	PaneRemote
)

type SftpQuitMsg struct{}
type SftpErrorMsg struct{ Err error }
type SftpRefreshMsg struct{}

type SftpModel struct {
	Client        *sftppkg.Client
	LocalDir      string
	RemoteDir     string
	LocalEntries  []sftppkg.Entry
	RemoteEntries []sftppkg.Entry
	LocalCursor   int
	RemoteCursor  int
	LocalScroll   int
	RemoteScroll  int
	Active        Pane
	Width         int
	Height        int
	Err           string
}

func NewSftpModel(client *sftppkg.Client, localDir, remoteDir string) SftpModel {
	m := SftpModel{
		Client:    client,
		LocalDir:  localDir,
		RemoteDir: remoteDir,
		Active:    PaneLocal,
		Width:     80,
		Height:    24,
	}
	m.refreshLocal()
	m.refreshRemote()
	return m
}

func (m *SftpModel) refreshLocal() {
	infos, err := os.ReadDir(m.LocalDir)
	if err != nil {
		m.Err = err.Error()
		m.LocalEntries = nil
		return
	}
	entries := make([]sftppkg.Entry, 0, len(infos))
	for _, fi := range infos {
		info, _ := fi.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		entries = append(entries, sftppkg.Entry{Name: fi.Name(), IsDir: fi.IsDir(), Size: size})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	m.LocalEntries = entries
	if m.LocalCursor >= len(entries) {
		m.LocalCursor = 0
	}
	m.LocalScroll = 0
}

func (m *SftpModel) refreshRemote() {
	if m.Client == nil {
		return
	}
	entries, err := m.Client.List(m.RemoteDir)
	if err != nil {
		m.Err = err.Error()
		m.RemoteEntries = nil
		return
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	m.RemoteEntries = entries
	if m.RemoteCursor >= len(entries) {
		m.RemoteCursor = 0
	}
	m.RemoteScroll = 0
}

func (m SftpModel) Init() tea.Cmd { return nil }

func (m SftpModel) paneBodyHeight() int {
	h := m.Height - 4
	if h < 3 {
		h = 3
	}
	return h
}

func (m *SftpModel) clampScroll() {
	body := m.paneBodyHeight()
	if m.LocalCursor < m.LocalScroll {
		m.LocalScroll = m.LocalCursor
	} else if m.LocalCursor >= m.LocalScroll+body {
		m.LocalScroll = m.LocalCursor - body + 1
	}
	if m.LocalScroll < 0 {
		m.LocalScroll = 0
	}
	if m.RemoteCursor < m.RemoteScroll {
		m.RemoteScroll = m.RemoteCursor
	} else if m.RemoteCursor >= m.RemoteScroll+body {
		m.RemoteScroll = m.RemoteCursor - body + 1
	}
	if m.RemoteScroll < 0 {
		m.RemoteScroll = 0
	}
}

func (m SftpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.clampScroll()
		return m, nil
	case SftpRefreshMsg:
		m.refreshLocal()
		m.refreshRemote()
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			if m.Active == PaneLocal {
				m.Active = PaneRemote
			} else {
				m.Active = PaneLocal
			}
		case tea.KeyUp:
			if m.Active == PaneLocal && m.LocalCursor > 0 {
				m.LocalCursor--
			} else if m.Active == PaneRemote && m.RemoteCursor > 0 {
				m.RemoteCursor--
			}
			m.clampScroll()
		case tea.KeyDown:
			if m.Active == PaneLocal && m.LocalCursor < len(m.LocalEntries)-1 {
				m.LocalCursor++
			} else if m.Active == PaneRemote && m.RemoteCursor < len(m.RemoteEntries)-1 {
				m.RemoteCursor++
			}
			m.clampScroll()
		case tea.KeyPgUp:
			body := m.paneBodyHeight()
			if m.Active == PaneLocal {
				m.LocalCursor -= body
				if m.LocalCursor < 0 {
					m.LocalCursor = 0
				}
			} else {
				m.RemoteCursor -= body
				if m.RemoteCursor < 0 {
					m.RemoteCursor = 0
				}
			}
			m.clampScroll()
		case tea.KeyPgDown:
			body := m.paneBodyHeight()
			if m.Active == PaneLocal {
				m.LocalCursor += body
				if m.LocalCursor > len(m.LocalEntries)-1 {
					m.LocalCursor = len(m.LocalEntries) - 1
				}
				if m.LocalCursor < 0 {
					m.LocalCursor = 0
				}
			} else {
				m.RemoteCursor += body
				if m.RemoteCursor > len(m.RemoteEntries)-1 {
					m.RemoteCursor = len(m.RemoteEntries) - 1
				}
				if m.RemoteCursor < 0 {
					m.RemoteCursor = 0
				}
			}
			m.clampScroll()
		case tea.KeyHome:
			if m.Active == PaneLocal {
				m.LocalCursor = 0
			} else {
				m.RemoteCursor = 0
			}
			m.clampScroll()
		case tea.KeyEnd:
			if m.Active == PaneLocal {
				m.LocalCursor = len(m.LocalEntries) - 1
			} else {
				m.RemoteCursor = len(m.RemoteEntries) - 1
			}
			if m.LocalCursor < 0 {
				m.LocalCursor = 0
			}
			if m.RemoteCursor < 0 {
				m.RemoteCursor = 0
			}
			m.clampScroll()
		case tea.KeyEnter:
			cmd := m.descend()
			m.clampScroll()
			return m, cmd
		case tea.KeyBackspace:
			cmd := m.ascend()
			m.clampScroll()
			return m, cmd
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "q":
				return m, func() tea.Msg { return SftpQuitMsg{} }
			case "c":
				return m, m.copy()
			case "d":
				return m, m.delete()
			case "r":
				m.refreshLocal()
				m.refreshRemote()
			}
		}
	}
	return m, nil
}

func (m *SftpModel) descend() tea.Cmd {
	if m.Active == PaneLocal {
		if m.LocalCursor >= len(m.LocalEntries) {
			return nil
		}
		e := m.LocalEntries[m.LocalCursor]
		if !e.IsDir {
			return nil
		}
		m.LocalDir = filepath.Join(m.LocalDir, e.Name)
		m.LocalCursor = 0
		m.refreshLocal()
	} else {
		if m.RemoteCursor >= len(m.RemoteEntries) {
			return nil
		}
		e := m.RemoteEntries[m.RemoteCursor]
		if !e.IsDir {
			return nil
		}
		m.RemoteDir = sftppkg.Join(m.RemoteDir, e.Name)
		m.RemoteCursor = 0
		m.refreshRemote()
	}
	return nil
}

func (m *SftpModel) ascend() tea.Cmd {
	if m.Active == PaneLocal {
		m.LocalDir = filepath.Dir(m.LocalDir)
		m.LocalCursor = 0
		m.refreshLocal()
	} else {
		m.RemoteDir = sftppkg.Parent(m.RemoteDir)
		m.RemoteCursor = 0
		m.refreshRemote()
	}
	return nil
}

func (m *SftpModel) copy() tea.Cmd {
	if m.Client == nil {
		return nil
	}
	if m.Active == PaneLocal {
		if m.LocalCursor >= len(m.LocalEntries) {
			return nil
		}
		e := m.LocalEntries[m.LocalCursor]
		if e.IsDir {
			m.Err = "directory copy not supported"
			return nil
		}
		src := filepath.Join(m.LocalDir, e.Name)
		dst := sftppkg.Join(m.RemoteDir, e.Name)
		if err := m.Client.Upload(src, dst); err != nil {
			m.Err = err.Error()
			return nil
		}
		m.refreshRemote()
	} else {
		if m.RemoteCursor >= len(m.RemoteEntries) {
			return nil
		}
		e := m.RemoteEntries[m.RemoteCursor]
		if e.IsDir {
			m.Err = "directory copy not supported"
			return nil
		}
		src := sftppkg.Join(m.RemoteDir, e.Name)
		dst := filepath.Join(m.LocalDir, e.Name)
		if err := m.Client.Download(src, dst); err != nil {
			m.Err = err.Error()
			return nil
		}
		m.refreshLocal()
	}
	m.Err = ""
	return nil
}

func (m *SftpModel) delete() tea.Cmd {
	if m.Active == PaneLocal {
		if m.LocalCursor >= len(m.LocalEntries) {
			return nil
		}
		e := m.LocalEntries[m.LocalCursor]
		target := filepath.Join(m.LocalDir, e.Name)
		if err := os.Remove(target); err != nil {
			m.Err = err.Error()
			return nil
		}
		m.refreshLocal()
	} else {
		if m.Client == nil || m.RemoteCursor >= len(m.RemoteEntries) {
			return nil
		}
		e := m.RemoteEntries[m.RemoteCursor]
		target := sftppkg.Join(m.RemoteDir, e.Name)
		if err := m.Client.Delete(target); err != nil {
			m.Err = err.Error()
			return nil
		}
		m.refreshRemote()
	}
	m.Err = ""
	return nil
}

func (m SftpModel) View() string {
	paneW := (m.Width - 2) / 2
	if paneW < 20 {
		paneW = 20
	}
	body := m.paneBodyHeight()
	left := m.renderPane("Local: "+truncate(m.LocalDir, paneW-10), m.LocalEntries, m.LocalCursor, m.LocalScroll, m.Active == PaneLocal, paneW, body)
	right := m.renderPane("Remote: "+truncate(m.RemoteDir, paneW-10), m.RemoteEntries, m.RemoteCursor, m.RemoteScroll, m.Active == PaneRemote, paneW, body)
	joined := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	help := StyleHelp.Render("[Tab] switch  [Enter] open  [Bksp] up  [c] copy  [d] del  [r] refresh  [PgUp/PgDn] page  [q] back")
	if m.Err != "" {
		help = StyleError.Render(m.Err) + "\n" + help
	}
	return joined + "\n" + help
}

func (m SftpModel) renderPane(title string, entries []sftppkg.Entry, cursor, scroll int, active bool, width, body int) string {
	var b strings.Builder
	b.WriteString(StyleTitle.Render(title))
	b.WriteString("\n")
	if len(entries) == 0 {
		b.WriteString(StyleHelp.Render("  (empty)"))
		b.WriteString("\n")
	} else {
		end := scroll + body
		if end > len(entries) {
			end = len(entries)
		}
		nameW := width - 6
		if nameW < 8 {
			nameW = 8
		}
		for i := scroll; i < end; i++ {
			e := entries[i]
			name := e.Name
			if e.IsDir {
				name += "/"
			}
			name = truncate(name, nameW)
			line := fmt.Sprintf("  %s", name)
			if i == cursor {
				line = StyleSelected.Render("> " + name)
			} else {
				line = StyleNormal.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		if len(entries) > body {
			b.WriteString(StyleHelp.Render(fmt.Sprintf("  [%d/%d]", cursor+1, len(entries))))
		}
	}
	style := StylePaneInactive
	if active {
		style = StylePaneActive
	}
	return style.Width(width).Height(body + 2).Render(b.String())
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return "..." + s[len(s)-(n-3):]
}
