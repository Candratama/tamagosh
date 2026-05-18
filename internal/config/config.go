package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Connection struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	User    string `json:"user"`
	PassKey string `json:"pass_key"`
}

type Store struct {
	Connections []Connection `json:"connections"`
}

func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Store{Connections: []Connection{}}, nil
	}
	if err != nil {
		return nil, err
	}
	s := &Store{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.Connections == nil {
		s.Connections = []Connection{}
	}
	return s, nil
}

func Save(path string, s *Store) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *Store) Find(name string) (Connection, int, bool) {
	for i, c := range s.Connections {
		if c.Name == name {
			return c, i, true
		}
	}
	return Connection{}, -1, false
}

func (s *Store) Add(c Connection) error {
	if _, _, ok := s.Find(c.Name); ok {
		return fmt.Errorf("connection %q already exists", c.Name)
	}
	s.Connections = append(s.Connections, c)
	return nil
}

func (s *Store) Update(name string, c Connection) error {
	_, idx, ok := s.Find(name)
	if !ok {
		return fmt.Errorf("connection %q not found", name)
	}
	s.Connections[idx] = c
	return nil
}

func (s *Store) Delete(name string) error {
	_, idx, ok := s.Find(name)
	if !ok {
		return fmt.Errorf("connection %q not found", name)
	}
	s.Connections = append(s.Connections[:idx], s.Connections[idx+1:]...)
	return nil
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "sshm", "connections.json"), nil
}
