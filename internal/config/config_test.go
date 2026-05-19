package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")

	src := &Store{Connections: []Connection{
		{Name: "atlantic", Host: "43.228.213.209", Port: 2255, User: "candra", PassKey: "ssh/atlantic"},
	}}

	if err := Save(path, src); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := &Store{Connections: []Connection{
		{Name: "atlantic", Host: "43.228.213.209", Port: 2255, User: "candra", PassKey: "ssh/atlantic", AuthMethod: "password"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.json")
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(s.Connections) != 0 {
		t.Fatalf("expected empty store, got %+v", s)
	}
}

func TestAddConnection(t *testing.T) {
	s := &Store{}
	c := Connection{Name: "a", Host: "h", Port: 22, User: "u", PassKey: "ssh/a"}
	if err := s.Add(c); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(s.Connections) != 1 || s.Connections[0].Name != "a" {
		t.Fatalf("unexpected store: %+v", s)
	}
	if err := s.Add(c); err == nil {
		t.Fatalf("expected duplicate-name error")
	}
}

func TestUpdateConnection(t *testing.T) {
	s := &Store{Connections: []Connection{{Name: "a", Host: "h1", Port: 22, User: "u", PassKey: "ssh/a"}}}
	if err := s.Update("a", Connection{Name: "a", Host: "h2", Port: 23, User: "u", PassKey: "ssh/a"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if s.Connections[0].Host != "h2" || s.Connections[0].Port != 23 {
		t.Fatalf("update did not apply: %+v", s.Connections[0])
	}
	if err := s.Update("missing", Connection{Name: "missing"}); err == nil {
		t.Fatalf("expected not-found error")
	}
}

func TestDeleteConnection(t *testing.T) {
	s := &Store{Connections: []Connection{{Name: "a"}, {Name: "b"}}}
	if err := s.Delete("a"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(s.Connections) != 1 || s.Connections[0].Name != "b" {
		t.Fatalf("delete failed: %+v", s)
	}
	if err := s.Delete("a"); err == nil {
		t.Fatalf("expected not-found error")
	}
}

func TestLoadLegacyConnectionMigratesToPasswordAuth(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "connections.json")
	legacy := `{"connections":[{"name":"old","host":"h","port":22,"user":"u","pass_key":"ssh/old"}]}`
	if err := os.WriteFile(p, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Connections) != 1 {
		t.Fatalf("want 1 conn, got %d", len(s.Connections))
	}
	c := s.Connections[0]
	if c.AuthMethod != "password" {
		t.Fatalf("AuthMethod=%q want 'password'", c.AuthMethod)
	}
	if c.KeyPath != "" {
		t.Fatalf("KeyPath=%q want empty", c.KeyPath)
	}
}

func TestLoadKeyAuthRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "connections.json")
	s := &Store{Connections: []Connection{{
		Name: "k", Host: "h", Port: 22, User: "u",
		AuthMethod: "key", KeyPath: "/home/u/.ssh/id_ed25519", PassKey: "ssh/k",
	}}}
	if err := Save(p, s); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	c := loaded.Connections[0]
	if c.AuthMethod != "key" || c.KeyPath != "/home/u/.ssh/id_ed25519" {
		t.Fatalf("round-trip lost fields: %+v", c)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("HOME", "/tmp/fakehome")
	t.Setenv("TAMAGOSH_HOME", "")
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	want := "/tmp/fakehome/.config/tamagosh/connections.json"
	if p != want {
		t.Fatalf("got %q want %q", p, want)
	}
}
