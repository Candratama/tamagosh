package keygen

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// Generate writes a new ed25519 SSH keypair to privPath (mode 0600) and
// privPath+".pub" (mode 0644). If passphrase is non-empty, the private key is
// encrypted. Refuses to overwrite existing files.
func Generate(privPath, passphrase string) error {
	if _, err := os.Stat(privPath); err == nil {
		return fmt.Errorf("refusing to overwrite existing file: %s", privPath)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("ed25519 generate: %w", err)
	}

	var block *pem.Block
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(priv, "tamagosh")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, "tamagosh", []byte(passphrase))
	}
	if err != nil {
		return fmt.Errorf("marshal private: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(privPath), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(privPath, pem.EncodeToMemory(block), 0o600); err != nil {
		return err
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		os.Remove(privPath)
		return fmt.Errorf("derive public: %w", err)
	}
	pubBytes := ssh.MarshalAuthorizedKey(sshPub)
	if err := os.WriteFile(privPath+".pub", pubBytes, 0o644); err != nil {
		os.Remove(privPath)
		return err
	}
	return nil
}
