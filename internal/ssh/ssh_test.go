package ssh

import (
	"strings"
	"testing"

	"github.com/Candratama/tamagosh/internal/config"
)

func TestBuildCommandPasswordAuth(t *testing.T) {
	conn := config.Connection{Name: "a", Host: "h", Port: 2255, User: "u", AuthMethod: "password"}
	name, args := BuildCommand(conn)
	if name != "sshpass" {
		t.Fatalf("name=%q want sshpass", name)
	}
	if args[0] != "-e" {
		t.Fatalf("args[0]=%q want -e", args[0])
	}
	if !strings.HasSuffix(args[1], "ssh") {
		t.Fatalf("ssh binary missing: %q", args[1])
	}
}

func TestBuildCommandKeyAuthNoPassphrase(t *testing.T) {
	conn := config.Connection{
		Name: "k", Host: "h", Port: 22, User: "u",
		AuthMethod: "key", KeyPath: "/home/u/.ssh/id_ed25519",
	}
	name, args := BuildCommand(conn)
	if !strings.HasSuffix(name, "ssh") {
		t.Fatalf("name=%q want ssh binary", name)
	}
	found := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-i" && args[i+1] == "/home/u/.ssh/id_ed25519" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected -i <keypath> in args: %v", args)
	}
	for _, a := range args {
		if a == "sshpass" || a == "-e" {
			t.Fatalf("key auth must not use sshpass: %v", args)
		}
	}
}

func TestBuildCommandDefaultPort(t *testing.T) {
	conn := config.Connection{Name: "x", Host: "h", Port: 0, User: "u", AuthMethod: "password"}
	_, args := BuildCommand(conn)
	found := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-p" && args[i+1] == "22" {
			found = true
		}
	}
	if !found {
		t.Fatalf("default port 22 missing: %v", args)
	}
}
