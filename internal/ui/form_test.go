package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Candratama/tamagosh/internal/config"
)

func TestFormValidation(t *testing.T) {
	m := NewFormModel(config.Connection{}, false)
	if _, err := m.Build(); err == nil {
		t.Fatalf("expected validation error for empty form")
	}

	m.Fields[FieldName].Value = "x"
	m.Fields[FieldHost].Value = "h"
	m.Fields[FieldPort].Value = "abc"
	m.Fields[FieldUser].Value = "u"
	if _, err := m.Build(); err == nil {
		t.Fatalf("expected port validation error")
	}

	m.Fields[FieldPort].Value = "22"
	c, err := m.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if c.Name != "x" || c.Host != "h" || c.Port != 22 || c.User != "u" {
		t.Fatalf("got %+v", c)
	}
	if c.PassKey != "ssh/x" {
		t.Fatalf("pass key=%q", c.PassKey)
	}
}

func TestFormTabCycles(t *testing.T) {
	m := NewFormModel(config.Connection{}, false)
	if m.Focus != 0 {
		t.Fatalf("focus=%d", m.Focus)
	}
	for i := 0; i < len(m.Fields); i++ {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = nm.(FormModel)
	}
	if m.Focus != 0 {
		t.Fatalf("expected wrap-around to 0, got %d", m.Focus)
	}
}

func TestFormRunesInsertIntoFocused(t *testing.T) {
	m := NewFormModel(config.Connection{}, false)
	for _, r := range "abc" {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = nm.(FormModel)
	}
	if m.Fields[FieldName].Value != "abc" {
		t.Fatalf("name=%q", m.Fields[FieldName].Value)
	}
}

func TestFormPrefillEdit(t *testing.T) {
	c := config.Connection{Name: "atlantic", Host: "h", Port: 2255, User: "u", PassKey: "ssh/atlantic"}
	m := NewFormModel(c, true)
	if m.Fields[FieldName].Value != "atlantic" {
		t.Fatalf("name not prefilled")
	}
	if m.Fields[FieldPort].Value != "2255" {
		t.Fatalf("port=%q", m.Fields[FieldPort].Value)
	}
	if !m.IsEdit {
		t.Fatalf("not in edit mode")
	}
}
