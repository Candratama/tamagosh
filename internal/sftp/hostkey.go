package sftp

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// hostKeyCallback returns an ssh.HostKeyCallback that:
//   - verifies against the known_hosts at khPath when an entry exists
//   - appends a new entry on first connect (parity with ssh's accept-new)
//   - returns an error on host key mismatch (does NOT silently accept)
//
// Concurrency note: two simultaneous first-connects to the same new host can
// both pass the "no entry" check and both append, producing a duplicate line.
// OpenSSH tolerates duplicates and tamagosh opens one SFTP session at a time
// per app instance, so this is accepted rather than locked.
func hostKeyCallback(khPath string) (ssh.HostKeyCallback, error) {
	if err := os.MkdirAll(filepath.Dir(khPath), 0o700); err != nil {
		return nil, err
	}
	if _, err := os.Stat(khPath); errors.Is(err, fs.ErrNotExist) {
		if f, err := os.OpenFile(khPath, os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
			f.Close()
		}
	}
	verify, err := knownhosts.New(khPath)
	if err != nil {
		return nil, err
	}
	return func(host string, remote net.Addr, key ssh.PublicKey) error {
		err := verify(host, remote, key)
		if err == nil {
			return nil
		}
		var kerr *knownhosts.KeyError
		if errors.As(err, &kerr) {
			if len(kerr.Want) == 0 {
				return appendKnownHost(khPath, host, key)
			}
			return fmt.Errorf("host key mismatch for %s — possible MITM (remove offending line from %s if intentional)", host, khPath)
		}
		return err
	}, nil
}

func appendKnownHost(path, host string, key ssh.PublicKey) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	line := knownhosts.Line([]string{host}, key)
	if _, err := fmt.Fprintln(f, line); err != nil {
		return err
	}
	return nil
}
