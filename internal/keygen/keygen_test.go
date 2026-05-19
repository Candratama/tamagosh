package keygen

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateUnencrypted(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	if err := Generate(keyPath, ""); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("private key mode=%o want 0600", info.Mode().Perm())
	}
	pubInfo, err := os.Stat(keyPath + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	if pubInfo.Mode().Perm() != 0o644 {
		t.Fatalf("public key mode=%o want 0644", pubInfo.Mode().Perm())
	}
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ssh.ParsePrivateKey(data); err != nil {
		t.Fatalf("parse generated key: %v", err)
	}
}

func TestGenerateWithPassphrase(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	if err := Generate(keyPath, "hunter2"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ssh.ParsePrivateKey(data); err == nil {
		t.Fatal("encrypted key should not parse without passphrase")
	}
	if _, err := ssh.ParsePrivateKeyWithPassphrase(data, []byte("hunter2")); err != nil {
		t.Fatalf("parse with passphrase: %v", err)
	}
}

func TestGenerateRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(keyPath, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Generate(keyPath, ""); err == nil {
		t.Fatal("expected error on overwrite")
	}
}
