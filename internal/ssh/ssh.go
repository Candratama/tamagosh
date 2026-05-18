package ssh

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/candratama/sshm/internal/config"
)

func BuildCommand(c config.Connection, password string) (string, []string) {
	port := c.Port
	if port == 0 {
		port = 22
	}
	args := []string{
		"-p", password,
		"/usr/bin/ssh",
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
