package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Candratama/tamagosh/internal/config"
)

func makeStore() *config.Store {
	return &config.Store{Connections: []config.Connection{
		{Name: "atlantic", Host: "43.228.213.209", Port: 2255, User: "candra"},
		{Name: "tencent", Host: "43.157.195.32", Port: 22, User: "candra"},
		{Name: "paringin", Host: "10.0.7.210", Port: 22, User: "candra"},
	}}
}

func TestListInitialSelection(t *testing.T) {
	m := NewListModel(makeStore())
	if m.Cursor != 0 {
		t.Fatalf("cursor=%d", m.Cursor)
	}
	if m.Selected().Name != "atlantic" {
		t.Fatalf("selected=%+v", m.Selected())
	}
}

func TestListMoveDownClamps(t *testing.T) {
	m := NewListModel(makeStore())
	for i := 0; i < 10; i++ {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = nm.(ListModel)
	}
	if m.Cursor != 2 {
		t.Fatalf("cursor=%d want 2", m.Cursor)
	}
}

func TestListFilter(t *testing.T) {
	m := NewListModel(makeStore())
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = nm.(ListModel)
	if !m.Filtering {
		t.Fatalf("not in filter mode")
	}
	for _, r := range "ten" {
		nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = nm.(ListModel)
	}
	if got := m.Visible(); len(got) != 1 || got[0].Name != "tencent" {
		t.Fatalf("visible=%+v", got)
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(ListModel)
	if m.Filtering {
		t.Fatalf("still filtering")
	}
	if len(m.Visible()) != 3 {
		t.Fatalf("filter not cleared")
	}
}

func TestListEnterEmits(t *testing.T) {
	m := NewListModel(makeStore())
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(ListModel)
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	msg := cmd()
	cm, ok := msg.(ConnectMsg)
	if !ok {
		t.Fatalf("got %T want ConnectMsg", msg)
	}
	if cm.Conn.Name != "atlantic" {
		t.Fatalf("conn=%+v", cm.Conn)
	}
}
