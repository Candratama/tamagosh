package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Candratama/tamagosh/internal/config"
)

type fakePass struct {
	values map[string]string
	setLog [][2]string
	delLog []string
	getErr error
}

func (f *fakePass) Get(k string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	return f.values[k], nil
}
func (f *fakePass) Set(k, v string) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[k] = v
	f.setLog = append(f.setLog, [2]string{k, v})
	return nil
}
func (f *fakePass) Delete(k string) error {
	delete(f.values, k)
	f.delLog = append(f.delLog, k)
	return nil
}
func (f *fakePass) GetPassphrase(k string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	return f.values[k+":passphrase"], nil
}
func (f *fakePass) SetPassphrase(k, v string) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	key := k + ":passphrase"
	f.values[key] = v
	f.setLog = append(f.setLog, [2]string{key, v})
	return nil
}
func (f *fakePass) DeletePassphrase(k string) error {
	key := k + ":passphrase"
	delete(f.values, key)
	f.delLog = append(f.delLog, key)
	return nil
}

func TestAppRoutesNewFormMsg(t *testing.T) {
	tmpPath := t.TempDir() + "/c.json"
	a := NewApp(&config.Store{}, &fakePass{}, nil, tmpPath)
	a.Mode = ModeList
	if a.View() == "" {
		t.Fatalf("empty view")
	}
	nm, _ := a.Update(NewFormMsg{})
	a = nm.(AppModel)
	if a.Mode != ModeForm {
		t.Fatalf("mode=%v want form", a.Mode)
	}
}

func TestAppFormSubmitAddsConnection(t *testing.T) {
	tmpPath := t.TempDir() + "/c.json"
	p := &fakePass{}
	a := NewApp(&config.Store{}, p, nil, tmpPath)
	a.Mode = ModeList
	a, _ = updateApp(a, NewFormMsg{})
	a, _ = updateApp(a, FormSubmitMsg{
		IsEdit: false,
		Conn:   config.Connection{Name: "a", Host: "h", Port: 22, User: "u", PassKey: "ssh/a"},
		Secret: FormSecret{Password: "pw"},
	})
	if len(a.Store.Connections) != 1 || a.Store.Connections[0].Name != "a" {
		t.Fatalf("store=%+v", a.Store)
	}
	if len(p.setLog) != 1 || p.setLog[0][0] != "ssh/a" || p.setLog[0][1] != "pw" {
		t.Fatalf("setLog=%+v", p.setLog)
	}
	if a.Mode != ModeList {
		t.Fatalf("mode=%v want list", a.Mode)
	}
}

func TestAppDeleteMsgRemovesConnection(t *testing.T) {
	tmpPath := t.TempDir() + "/c.json"
	p := &fakePass{values: map[string]string{"ssh/a": "pw"}}
	s := &config.Store{Connections: []config.Connection{{Name: "a", PassKey: "ssh/a"}}}
	a := NewApp(s, p, nil, tmpPath)
	a.Mode = ModeList
	a, _ = updateApp(a, DeleteMsg{Conn: s.Connections[0]})
	if a.Mode != ModeConfirmDelete {
		t.Fatalf("mode=%v", a.Mode)
	}
	a, _ = updateApp(a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if len(a.Store.Connections) != 0 {
		t.Fatalf("not deleted: %+v", a.Store)
	}
	if len(p.delLog) != 2 || p.delLog[0] != "ssh/a" || p.delLog[1] != "ssh/a:passphrase" {
		t.Fatalf("delLog=%+v", p.delLog)
	}
}

func TestAppEditSwitchPasswordToKeyClearsStalePassword(t *testing.T) {
	tmpPath := t.TempDir() + "/c.json"
	// Existing connection saved with password auth.
	p := &fakePass{values: map[string]string{"ssh/a": "old-pw"}}
	s := &config.Store{Connections: []config.Connection{{
		Name: "a", Host: "h", Port: 22, User: "u",
		PassKey: "ssh/a", AuthMethod: "password",
	}}}
	a := NewApp(s, p, nil, tmpPath)
	a.Mode = ModeList

	// Edit same connection, switch to key auth with a passphrase.
	a, _ = updateApp(a, FormSubmitMsg{
		IsEdit:   true,
		Original: "a",
		Conn: config.Connection{
			Name: "a", Host: "h", Port: 22, User: "u",
			PassKey: "ssh/a", AuthMethod: "key", KeyPath: "/home/u/.ssh/id_ed25519",
		},
		Secret: FormSecret{Passphrase: "secret-pp"},
	})
	if _, exists := p.values["ssh/a"]; exists {
		t.Fatal("stale password not cleared after switching to key auth")
	}
	if p.values["ssh/a:passphrase"] != "secret-pp" {
		t.Fatalf("passphrase not stored: %q", p.values["ssh/a:passphrase"])
	}
}

func TestAppEditSwitchKeyToPasswordClearsStalePassphrase(t *testing.T) {
	tmpPath := t.TempDir() + "/c.json"
	p := &fakePass{values: map[string]string{"ssh/a:passphrase": "old-pp"}}
	s := &config.Store{Connections: []config.Connection{{
		Name: "a", Host: "h", Port: 22, User: "u",
		PassKey: "ssh/a", AuthMethod: "key", KeyPath: "/home/u/.ssh/id_ed25519",
	}}}
	a := NewApp(s, p, nil, tmpPath)
	a.Mode = ModeList

	a, _ = updateApp(a, FormSubmitMsg{
		IsEdit:   true,
		Original: "a",
		Conn: config.Connection{
			Name: "a", Host: "h", Port: 22, User: "u",
			PassKey: "ssh/a", AuthMethod: "password",
		},
		Secret: FormSecret{Password: "new-pw"},
	})
	if _, exists := p.values["ssh/a:passphrase"]; exists {
		t.Fatal("stale passphrase not cleared after switching to password auth")
	}
	if p.values["ssh/a"] != "new-pw" {
		t.Fatalf("password not stored: %q", p.values["ssh/a"])
	}
}

func updateApp(a AppModel, msg tea.Msg) (AppModel, tea.Cmd) {
	nm, cmd := a.Update(msg)
	return nm.(AppModel), cmd
}
