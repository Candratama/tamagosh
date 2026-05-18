package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Candratama/tamagosh/internal/sftp"
)

func TestSftpTabSwitchesPane(t *testing.T) {
	m := NewSftpModel(nil, "/local", "/remote", nil, "")
	if m.Active != PaneLocal {
		t.Fatalf("active=%v", m.Active)
	}
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(SftpModel)
	if m.Active != PaneRemote {
		t.Fatalf("active=%v want remote", m.Active)
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(SftpModel)
	if m.Active != PaneLocal {
		t.Fatalf("active=%v want local", m.Active)
	}
}

func TestSftpCursorNavigatesActivePane(t *testing.T) {
	m := NewSftpModel(nil, "/local", "/remote", nil, "")
	m.LocalEntries = []sftp.Entry{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	m.RemoteEntries = []sftp.Entry{{Name: "x"}, {Name: "y"}}

	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(SftpModel)
	if m.LocalCursor != 1 || m.RemoteCursor != 0 {
		t.Fatalf("local=%d remote=%d", m.LocalCursor, m.RemoteCursor)
	}

	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(SftpModel)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(SftpModel)
	if m.RemoteCursor != 1 {
		t.Fatalf("remote=%d", m.RemoteCursor)
	}
}

func TestSftpQuitEmits(t *testing.T) {
	m := NewSftpModel(nil, "/local", "/remote", nil, "")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if _, ok := cmd().(SftpQuitMsg); !ok {
		t.Fatalf("got %T want SftpQuitMsg", cmd())
	}
}
