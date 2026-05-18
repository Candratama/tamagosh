package ssh

import (
	"testing"

	"github.com/candratama/sshm/internal/config"
)

func TestBuildCommand(t *testing.T) {
	conn := config.Connection{Name: "atlantic", Host: "43.228.213.209", Port: 2255, User: "candra"}
	name, args := BuildCommand(conn, "hunter2")
	if name != "sshpass" {
		t.Fatalf("name=%q", name)
	}
	want := []string{"-p", "hunter2", "/usr/bin/ssh", "-p", "2255",
		"-o", "StrictHostKeyChecking=accept-new",
		"candra@43.228.213.209"}
	if len(args) != len(want) {
		t.Fatalf("args len=%d want=%d (%v)", len(args), len(want), args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d]=%q want=%q", i, args[i], want[i])
		}
	}
}

func TestBuildCommandDefaultPort(t *testing.T) {
	conn := config.Connection{Name: "x", Host: "h", Port: 0, User: "u"}
	_, args := BuildCommand(conn, "p")
	found := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-p" && args[i+1] == "22" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected default port 22 in args: %v", args)
	}
}
