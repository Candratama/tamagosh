package secret

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.Set("ssh/a", "hunter2"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := s.Get("ssh/a")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "hunter2" {
		t.Fatalf("got %q want hunter2", got)
	}
}

func TestPersistAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	s1, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s1.Set("k", "v"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	s2, err := New(dir)
	if err != nil {
		t.Fatalf("New2: %v", err)
	}
	got, err := s2.Get("k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "v" {
		t.Fatalf("got %q want v", got)
	}
}

func TestGetMissing(t *testing.T) {
	s, _ := New(t.TempDir())
	if _, err := s.Get("nope"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDelete(t *testing.T) {
	s, _ := New(t.TempDir())
	if err := s.Set("k", "v"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Delete("k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get("k"); err == nil {
		t.Fatalf("expected error after delete")
	}
}

func TestKeyFilePermissions(t *testing.T) {
	dir := t.TempDir()
	if _, err := New(dir); err != nil {
		t.Fatalf("New: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "key"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("key perm=%v want 0600", info.Mode().Perm())
	}
}
