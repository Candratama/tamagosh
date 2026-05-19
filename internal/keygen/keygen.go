package keygen

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// Generate writes a new ed25519 SSH keypair to privPath (mode 0600) and
// privPath+".pub" (mode 0644). If passphrase is non-empty, the private key is
// encrypted. The private key is created with O_EXCL to refuse overwriting
// atomically (no TOCTOU window between check and write).
func Generate(privPath, passphrase string) error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("ed25519 generate: %w", err)
	}

	var block *pem.Block
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(priv, "")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte(passphrase))
	}
	if err != nil {
		return fmt.Errorf("marshal private: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(privPath), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(privPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("refusing to overwrite existing file: %s", privPath)
		}
		return err
	}
	if _, err := f.Write(pem.EncodeToMemory(block)); err != nil {
		f.Close()
		os.Remove(privPath)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(privPath)
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
