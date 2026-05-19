package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Store struct {
	keyPath     string
	secretsPath string
	key         []byte
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &Store{
		keyPath:     filepath.Join(dir, "key"),
		secretsPath: filepath.Join(dir, "secrets.json"),
	}
	key, err := s.loadOrCreateKey()
	if err != nil {
		return nil, err
	}
	s.key = key
	return s, nil
}

func (s *Store) loadOrCreateKey() ([]byte, error) {
	data, err := os.ReadFile(s.keyPath)
	if errors.Is(err, fs.ErrNotExist) {
		k := make([]byte, 32)
		if _, err := rand.Read(k); err != nil {
			return nil, err
		}
		if err := os.WriteFile(s.keyPath, k, 0o600); err != nil {
			return nil, err
		}
		return k, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) != 32 {
		return nil, fmt.Errorf("invalid key length: %d", len(data))
	}
	return data, nil
}

func (s *Store) readSecrets() (map[string]string, error) {
	data, err := os.ReadFile(s.secretsPath)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) writeSecrets(m map[string]string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.secretsPath, data, 0o600)
}

func (s *Store) encrypt(plain string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := g.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func (s *Store) decrypt(enc string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := g.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := g.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func (s *Store) Get(key string) (string, error) {
	m, err := s.readSecrets()
	if err != nil {
		return "", err
	}
	enc, ok := m[key]
	if !ok {
		return "", fmt.Errorf("secret %q not found", key)
	}
	return s.decrypt(enc)
}

func (s *Store) Set(key, value string) error {
	m, err := s.readSecrets()
	if err != nil {
		return err
	}
	enc, err := s.encrypt(value)
	if err != nil {
		return err
	}
	m[key] = enc
	return s.writeSecrets(m)
}

func (s *Store) Delete(key string) error {
	m, err := s.readSecrets()
	if err != nil {
		return err
	}
	delete(m, key)
	return s.writeSecrets(m)
}

// passphraseSuffix namespaces SSH key passphrases under the same encrypted
// store as passwords. Callers must not pass keys already ending in this
// suffix; the form layer rejects ':' in connection names to keep this safe.
const passphraseSuffix = ":passphrase"

func (s *Store) GetPassphrase(key string) (string, error) {
	return s.Get(key + passphraseSuffix)
}

func (s *Store) SetPassphrase(key, value string) error {
	return s.Set(key+passphraseSuffix, value)
}

func (s *Store) DeletePassphrase(key string) error {
	return s.Delete(key + passphraseSuffix)
}
