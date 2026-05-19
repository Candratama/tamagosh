package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeygenModelTargetPath(t *testing.T) {
	dir := t.TempDir()
	m := NewKeygenModel(dir)
	m.NameField.Value = "test-key"
	got := m.TargetPath()
	want := filepath.Join(dir, "keys", "test-key")
	if got != want {
		t.Fatalf("TargetPath=%q want %q", got, want)
	}
}

func TestKeygenModelGeneratesFile(t *testing.T) {
	dir := t.TempDir()
	m := NewKeygenModel(dir)
	m.NameField.Value = "k"
	m.PassphraseField.Value = ""
	if err := m.Generate(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "keys", "k")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "keys", "k.pub")); err != nil {
		t.Fatal(err)
	}
}

func TestKeygenModelRejectsPathSeparators(t *testing.T) {
	dir := t.TempDir()
	m := NewKeygenModel(dir)
	m.NameField.Value = "../escape"
	if err := m.Generate(); err == nil {
		t.Fatal("expected error for name containing path separator")
	}
}

func TestKeygenModelRejectsEmptyName(t *testing.T) {
	dir := t.TempDir()
	m := NewKeygenModel(dir)
	m.NameField.Value = ""
	if err := m.Generate(); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestKeygenModelRejectsBadNames(t *testing.T) {
	cases := []string{".", "..", ".hidden", "foo bar", "foo/bar", "foo\\bar", "foo\x00bar", "-leading-hyphen"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			m := NewKeygenModel(dir)
			m.NameField.Value = name
			if err := m.Generate(); err == nil {
				t.Fatalf("expected error for name %q", name)
			}
		})
	}
}

func TestKeygenModelAcceptsValidNames(t *testing.T) {
	cases := []string{"id_ed25519", "my-key", "prod.staging", "key1", "_underscore"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			m := NewKeygenModel(dir)
			m.NameField.Value = name
			if err := m.Generate(); err != nil {
				t.Fatalf("name %q unexpectedly rejected: %v", name, err)
			}
		})
	}
}
