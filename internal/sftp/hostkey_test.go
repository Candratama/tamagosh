package sftp

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

const testEd25519Key = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACCWSYgdyI4kPn33xhiH+C/2Ct0ll4g1pi9oFNY82Vk2SgAAAKDNIKGPzSCh
jwAAAAtzc2gtZWQyNTUxOQAAACCWSYgdyI4kPn33xhiH+C/2Ct0ll4g1pi9oFNY82Vk2Sg
AAAEAlCHuLU/DvjdmGd15Is/+p3F5v9B68ZK3EtfBnXnPB+JZJiB3IjiQ+fffGGIf4L/YK
3SWXiDWmL2gU1jzZWTZKAAAAF2NhbmRyYXRhbWFAQWlyLU0yLmxvY2FsAQIDBAUG
-----END OPENSSH PRIVATE KEY-----
`

func makeTestHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	priv, err := ssh.ParsePrivateKey([]byte(testEd25519Key))
	if err != nil {
		t.Fatal(err)
	}
	return priv.PublicKey()
}

func TestHostKeyCallbackAppendsOnFirstConnect(t *testing.T) {
	dir := t.TempDir()
	khPath := filepath.Join(dir, "known_hosts")
	cb, err := hostKeyCallback(khPath)
	if err != nil {
		t.Fatal(err)
	}
	pk := makeTestHostKey(t)
	addr, _ := net.ResolveTCPAddr("tcp", "192.0.2.10:22")
	if err := cb("192.0.2.10:22", addr, pk); err != nil {
		t.Fatalf("first connect: %v", err)
	}
	data, err := os.ReadFile(khPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "192.0.2.10") {
		t.Fatalf("known_hosts missing entry: %q", string(data))
	}
	cb2, err := hostKeyCallback(khPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := cb2("192.0.2.10:22", addr, pk); err != nil {
		t.Fatalf("second connect (same key): %v", err)
	}
}
