package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Candratama/tamagosh/internal/config"
)

func BuildCommand(c config.Connection, password string) (string, []string) {
	port := c.Port
	if port == 0 {
		port = 22
	}
	if os.Getenv("TAMAGOSH_DEMO") == "1" {
		banner := fmt.Sprintf(
			"\033[1;33mconnected to %s@%s:%d\033[0m\n\n"+
				"Linux %s 6.1.0-tamagosh-demo #1 x86_64 GNU/Linux\n"+
				"Last login: %s from 198.51.100.1\n\n"+
				"\033[2m(demo mode — no real SSH session)\033[0m\n"+
				"\033[2mtype 'exit' or press Ctrl-D to return\033[0m\n\n"+
				"%s@%s:~$ ",
			c.User, c.Host, port,
			c.Name,
			time.Now().Format("Mon Jan 2 15:04:05 2006"),
			c.User, c.Name,
		)
		return "sh", []string{"-c", fmt.Sprintf("clear; printf %q; cat >/dev/null", banner)}
	}
	sshBin := "ssh"
	if p, err := exec.LookPath("ssh"); err == nil {
		sshBin = p
	}
	args := []string{
		"-p", password,
		sshBin,
		"-p", fmt.Sprintf("%d", port),
		"-o", "StrictHostKeyChecking=accept-new",
		fmt.Sprintf("%s@%s", c.User, c.Host),
	}
	return "sshpass", args
}

type ExitMsg struct {
	Err error
}

func ConnectCmd(c config.Connection, password string) tea.Cmd {
	name, args := BuildCommand(c, password)
	cmd := exec.Command(name, args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return ExitMsg{Err: err}
	})
}
