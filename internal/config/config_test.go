package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")

	want := &Store{Connections: []Connection{
		{Name: "atlantic", Host: "43.228.213.209", Port: 2255, User: "candra", PassKey: "ssh/atlantic"},
	}}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
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

func TestDefaultPath(t *testing.T) {
	t.Setenv("HOME", "/tmp/fakehome")
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	want := "/tmp/fakehome/.config/tamagosh/connections.json"
	if p != want {
		t.Fatalf("got %q want %q", p, want)
	}
}
