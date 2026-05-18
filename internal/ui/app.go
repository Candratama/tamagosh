package ui

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/candratama/sshm/internal/config"
	sftppkg "github.com/candratama/sshm/internal/sftp"
	"github.com/candratama/sshm/internal/ssh"
)

type Mode int

const (
	ModeList Mode = iota
	ModeForm
	ModeConfirmDelete
	ModeSftp
)

type PassStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

type AppModel struct {
	Mode      Mode
	Store     *config.Store
	StorePath string
	Pass      PassStore
	List      ListModel
	Form      FormModel
	Sftp      SftpModel
	Pending   config.Connection
	Width     int
	Height    int
	Err       string
}

func NewApp(store *config.Store, pass PassStore, storePath string) AppModel {
	return AppModel{
		Mode:      ModeList,
		Store:     store,
		StorePath: storePath,
		Pass:      pass,
		List:      NewListModel(store),
	}
}

func (a AppModel) Init() tea.Cmd { return nil }

func (a AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		a.Width = ws.Width
		a.Height = ws.Height
	}
	switch m := msg.(type) {
	case NewFormMsg:
		a.Form = NewFormModel(config.Connection{}, false)
		a.Mode = ModeForm
		return a, nil
	case EditFormMsg:
		a.Form = NewFormModel(m.Conn, true)
		a.Mode = ModeForm
		return a, nil
	case FormCancelMsg:
		a.Mode = ModeList
		return a, nil
	case FormSubmitMsg:
		return a.handleSubmit(m)
	case DeleteMsg:
		a.Pending = m.Conn
		a.Mode = ModeConfirmDelete
		return a, nil
	case ConnectMsg:
		return a.handleConnect(m.Conn)
	case ssh.ExitMsg:
		if m.Err != nil {
			a.List.Err = "ssh exited: " + m.Err.Error()
		} else {
			a.List.Err = ""
		}
		a.Mode = ModeList
		return a, nil
	case OpenSftpMsg:
		return a.handleSftp(m.Conn)
	case SftpQuitMsg:
		if a.Sftp.Client != nil {
			a.Sftp.Client.Close()
		}
		a.Mode = ModeList
		return a, nil
	case tea.KeyMsg:
		if a.Mode == ModeConfirmDelete {
			switch string(m.Runes) {
			case "y", "Y":
				return a.confirmDelete()
			}
			if m.Type == tea.KeyEsc || string(m.Runes) == "n" || string(m.Runes) == "N" {
				a.Mode = ModeList
				return a, nil
			}
			return a, nil
		}
	}

	switch a.Mode {
	case ModeList:
		nm, cmd := a.List.Update(msg)
		a.List = nm.(ListModel)
		return a, cmd
	case ModeForm:
		nm, cmd := a.Form.Update(msg)
		a.Form = nm.(FormModel)
		return a, cmd
	case ModeSftp:
		nm, cmd := a.Sftp.Update(msg)
		a.Sftp = nm.(SftpModel)
		return a, cmd
	}
	return a, nil
}

func (a AppModel) handleSubmit(m FormSubmitMsg) (tea.Model, tea.Cmd) {
	if m.IsEdit {
		if m.Conn.Name != m.Original {
			if err := a.Pass.Delete("ssh/" + m.Original); err != nil {
				a.Form.Err = err.Error()
				return a, nil
			}
			if err := a.Store.Delete(m.Original); err != nil {
				a.Form.Err = err.Error()
				return a, nil
			}
			if err := a.Store.Add(m.Conn); err != nil {
				a.Form.Err = err.Error()
				return a, nil
			}
		} else {
			if err := a.Store.Update(m.Original, m.Conn); err != nil {
				a.Form.Err = err.Error()
				return a, nil
			}
		}
		if m.Password != "" {
			if err := a.Pass.Set(m.Conn.PassKey, m.Password); err != nil {
				a.Form.Err = err.Error()
				return a, nil
			}
		}
	} else {
		if err := a.Store.Add(m.Conn); err != nil {
			a.Form.Err = err.Error()
			return a, nil
		}
		if err := a.Pass.Set(m.Conn.PassKey, m.Password); err != nil {
			a.Form.Err = err.Error()
			return a, nil
		}
	}
	if err := config.Save(a.StorePath, a.Store); err != nil {
		a.Form.Err = err.Error()
		return a, nil
	}
	a.List = NewListModel(a.Store)
	a.Mode = ModeList
	return a, nil
}

func (a AppModel) confirmDelete() (tea.Model, tea.Cmd) {
	if err := a.Pass.Delete(a.Pending.PassKey); err != nil {
		a.List.Err = err.Error()
		a.Mode = ModeList
		return a, nil
	}
	if err := a.Store.Delete(a.Pending.Name); err != nil {
		a.List.Err = err.Error()
		a.Mode = ModeList
		return a, nil
	}
	if err := config.Save(a.StorePath, a.Store); err != nil {
		a.List.Err = err.Error()
	}
	a.List = NewListModel(a.Store)
	a.Mode = ModeList
	return a, nil
}

func (a AppModel) handleConnect(c config.Connection) (tea.Model, tea.Cmd) {
	pwd, err := a.Pass.Get(c.PassKey)
	if err != nil {
		a.List.Err = "pass: " + err.Error()
		return a, nil
	}
	return a, ssh.ConnectCmd(c, pwd)
}

func (a AppModel) handleSftp(c config.Connection) (tea.Model, tea.Cmd) {
	pwd, err := a.Pass.Get(c.PassKey)
	if err != nil {
		a.List.Err = "pass: " + err.Error()
		return a, nil
	}
	client, err := sftppkg.Connect(c, pwd)
	if err != nil {
		a.List.Err = "sftp: " + err.Error()
		return a, nil
	}
	remoteHome, err := client.Home()
	if err != nil {
		remoteHome = "/"
	}
	localHome, err := os.UserHomeDir()
	if err != nil {
		localHome, _ = filepath.Abs(".")
	}
	a.Sftp = NewSftpModel(client, localHome, remoteHome)
	if a.Width > 0 && a.Height > 0 {
		a.Sftp.Width = a.Width
		a.Sftp.Height = a.Height
	}
	a.Mode = ModeSftp
	return a, nil
}

func (a AppModel) View() string {
	switch a.Mode {
	case ModeForm:
		return a.Form.View()
	case ModeSftp:
		return a.Sftp.View()
	case ModeConfirmDelete:
		return StyleBorder.Render(
			StyleTitle.Render("Delete connection") + "\n\n" +
				fmt.Sprintf("  Delete %q? [y/N]", a.Pending.Name),
		)
	default:
		return a.List.View()
	}
}
