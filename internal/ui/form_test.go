package ui

import (
	"testing"

	"github.com/Candratama/tamagosh/internal/config"
)

func emptyConn() config.Connection { return config.Connection{} }

func fillBaseFields(m *FormModel) {
	m.Fields[FieldName].Value = "a"
	m.Fields[FieldHost].Value = "h"
	m.Fields[FieldPort].Value = "22"
	m.Fields[FieldUser].Value = "u"
}

func TestFormBuildPasswordAuth(t *testing.T) {
	m := NewFormModel(emptyConn(), false)
	fillBaseFields(&m)
	c, _, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	if c.AuthMethod != "password" {
		t.Fatalf("AuthMethod=%q want password", c.AuthMethod)
	}
}

func TestFormToggleKeyAuthShowsKeyFields(t *testing.T) {
	m := NewFormModel(emptyConn(), false)
	m.SetAuth("key")
	if !m.HasField(FieldKeyPath) {
		t.Fatal("key mode must show KeyPath field")
	}
	if !m.HasField(FieldPassphrase) {
		t.Fatal("key mode must show Passphrase field")
	}
	if m.HasField(FieldPassword) {
		t.Fatal("key mode must hide Password field")
	}
}

func TestFormBuildKeyAuthRequiresKeyPath(t *testing.T) {
	m := NewFormModel(emptyConn(), false)
	m.SetAuth("key")
	fillBaseFields(&m)
	if _, _, err := m.Build(); err == nil {
		t.Fatal("expected error for empty KeyPath")
	}
}

func TestFormBuildKeyAuthSuccess(t *testing.T) {
	m := NewFormModel(emptyConn(), false)
	m.SetAuth("key")
	fillBaseFields(&m)
	m.Fields[FieldKeyPath].Value = "/home/u/.ssh/id_ed25519"
	m.Fields[FieldPassphrase].Value = "hunter2"
	c, sec, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	if c.AuthMethod != "key" {
		t.Fatalf("AuthMethod=%q want key", c.AuthMethod)
	}
	if c.KeyPath != "/home/u/.ssh/id_ed25519" {
		t.Fatalf("KeyPath=%q", c.KeyPath)
	}
	if sec.Passphrase != "hunter2" {
		t.Fatalf("Passphrase=%q", sec.Passphrase)
	}
	if sec.Password != "" {
		t.Fatalf("Password should be empty for key auth, got %q", sec.Password)
	}
}

func TestFormBuildRejectsColonInName(t *testing.T) {
	m := NewFormModel(emptyConn(), false)
	m.Fields[FieldName].Value = "bad:name"
	m.Fields[FieldHost].Value = "h"
	m.Fields[FieldPort].Value = "22"
	m.Fields[FieldUser].Value = "u"
	if _, _, err := m.Build(); err == nil {
		t.Fatal("expected error for ':' in connection name")
	}
}

func TestFormEditPreservesAuthMethodAndKeyPath(t *testing.T) {
	c := config.Connection{
		Name: "k", Host: "h", Port: 22, User: "u",
		AuthMethod: "key", KeyPath: "/home/u/.ssh/id_ed25519",
	}
	m := NewFormModel(c, true)
	if m.Auth != "key" {
		t.Fatalf("Auth=%q want key", m.Auth)
	}
	if m.Fields[FieldKeyPath].Value != "/home/u/.ssh/id_ed25519" {
		t.Fatalf("KeyPath not preloaded: %q", m.Fields[FieldKeyPath].Value)
	}
	if !m.HasField(FieldKeyPath) || !m.HasField(FieldPassphrase) {
		t.Fatal("edit-mode key form must show key fields")
	}
}
