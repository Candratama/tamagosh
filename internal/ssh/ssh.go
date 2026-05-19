package ssh

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Candratama/tamagosh/internal/config"
)

// BuildCommand returns (binary, args) for the SSH invocation.
//
// Password auth: returns ("sshpass", ["-e", ssh, "-p", port, "-o", opt, user@host]).
// Caller MUST set SSHPASS env on the resulting cmd (see ConnectCmd).
//
// Key auth: returns (ssh, ["-i", keypath, "-p", port, "-o", opts..., user@host]).
// If the key has a passphrase, ConnectCmd wires SSH_ASKPASS — that lands in Task 5.
func BuildCommand(c config.Connection) (string, []string) {
	port := c.Port
	if port == 0 {
		port = 22
	}
	sshBin := "ssh"
	if p, err := exec.LookPath("ssh"); err == nil {
		sshBin = p
	}
	target := fmt.Sprintf("%s@%s", c.User, c.Host)
	portStr := fmt.Sprintf("%d", port)

	if c.AuthMethod == "key" {
		args := []string{
			"-i", c.KeyPath,
			"-p", portStr,
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "IdentitiesOnly=yes",
			target,
		}
		return sshBin, args
	}

	args := []string{
		"-e",
		sshBin,
		"-p", portStr,
		"-o", "StrictHostKeyChecking=accept-new",
		target,
	}
	return "sshpass", args
}

type ExitMsg struct {
	Err error
}

// ConnectCmd builds the exec.Cmd with env wired for the auth method:
//   - password: SSHPASS env var (no plaintext in args)
//   - key: just exec for now (passphrase handling lands in Task 5)
//
// `secret` is the password for password auth, passphrase for key auth (empty if no passphrase).
func ConnectCmd(c config.Connection, secret string) tea.Cmd {
	name, args := BuildCommand(c)
	cmd := exec.Command(name, args...)
	env := os.Environ()

	switch c.AuthMethod {
	case "key":
		if secret != "" {
			self, err := os.Executable()
			if err != nil {
				break
			}
			// ssh execs SSH_ASKPASS directly (no shell, no args), so we need
			// a single executable. A tiny /bin/sh script that execs ourselves
			// with the `askpass` subcommand does the job.
			script := fmt.Sprintf("#!/bin/sh\nexec %q askpass\n", self)
			f, err := os.CreateTemp("", "tamagosh-askpass-*.sh")
			if err != nil {
				break
			}
			if _, err := f.WriteString(script); err != nil {
				f.Close()
				os.Remove(f.Name())
				break
			}
			f.Close()
			if err := os.Chmod(f.Name(), 0o700); err != nil {
				os.Remove(f.Name())
				break
			}
			// No cleanup: tea.ExecProcess takes ownership of the subprocess,
			// so there's no place to defer removal. The script lives in /tmp
			// with 0700 perms and contains only an exec stub (no secret on
			// disk). The system tmp reaper handles it. Intentional.
			//
			// Security note: TAMAGOSH_PASSPHRASE is inherited by the ssh child
			// process. This is unavoidable — ssh consults SSH_ASKPASS after
			// fork/exec, so the env must be set on the ssh process. The
			// passphrase is visible via /proc/<pid>/environ to the owning user
			// only (process credentials enforce read access). Remote shells do
			// NOT receive it: ssh does not forward env vars by default.
			env = append(env,
				"TAMAGOSH_PASSPHRASE="+secret,
				"SSH_ASKPASS="+f.Name(),
				"SSH_ASKPASS_REQUIRE=force",
			)
			// Only set DISPLAY if not already set — preserves the user's X11
			// configuration. SSH_ASKPASS_REQUIRE=force only consults DISPLAY
			// on older OpenSSH; modern versions ignore it when REQUIRE=force.
			if os.Getenv("DISPLAY") == "" {
				env = append(env, "DISPLAY=:0")
			}
			// ssh prefers the controlling TTY over SSH_ASKPASS. setsid detaches
			// the child from the TTY so ASKPASS is consulted. Best-effort —
			// if setsid is unavailable (rare on macOS Homebrew, mostly Linux),
			// fall through and let ssh prompt on the TTY (cosmetic glitch).
			if setsid, err := exec.LookPath("setsid"); err == nil {
				newArgs := append([]string{name}, args...)
				cmd = exec.Command(setsid, newArgs...)
			}
		}
	default:
		env = append(env, "SSHPASS="+secret)
	}

	cmd.Env = env
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return ExitMsg{Err: err}
	})
}
