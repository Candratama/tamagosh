package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Candratama/tamagosh/internal/bookmark"
	"github.com/Candratama/tamagosh/internal/config"
	sftppkg "github.com/Candratama/tamagosh/internal/sftp"
	"github.com/Candratama/tamagosh/internal/ssh"
)

type Mode int

const (
	ModeList Mode = iota
	ModeForm
	ModeConfirmDelete
	ModeSftp
	ModeConnecting
	ModeSplash
	ModeKeygen
)

type SplashDoneMsg struct{}

type startSshMsg struct {
	conn config.Connection
	pwd  string
}

type PassStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
	GetPassphrase(key string) (string, error)
	SetPassphrase(key, value string) error
	DeletePassphrase(key string) error
}

type AppModel struct {
	Mode      Mode
	Store     *config.Store
	StorePath string
	Pass      PassStore
	Bookmark  *bookmark.Store
	List      ListModel
	Form      FormModel
	Sftp      SftpModel
	Keygen    KeygenModel
	Pending   config.Connection
	Width     int
	Height    int
	Err       string
}

func NewApp(store *config.Store, pass PassStore, bm *bookmark.Store, storePath string) AppModel {
	return AppModel{
		Mode:      ModeList,
		Store:     store,
		StorePath: storePath,
		Pass:      pass,
		Bookmark:  bm,
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
	case startSshMsg:
		return a, ssh.ConnectCmd(m.conn, m.pwd)
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
	case KeygenStartMsg:
		cfgDir := filepath.Dir(a.StorePath)
		a.Keygen = NewKeygenModel(cfgDir)
		a.Mode = ModeKeygen
		return a, nil
	case KeygenCancelMsg:
		a.Mode = ModeList
		return a, nil
	case KeygenDoneMsg:
		if m.Err != nil {
			a.Keygen.Err = m.Err.Error()
			return a, nil
		}
		a.List.Err = ""
		a.List.Info = "generated: " + m.Path
		a.Mode = ModeList
		return a, nil
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
	case ModeKeygen:
		nm, cmd := a.Keygen.Update(msg)
		a.Keygen = nm.(KeygenModel)
		return a, cmd
	}
	return a, nil
}

func (a AppModel) handleSubmit(m FormSubmitMsg) (tea.Model, tea.Cmd) {
	if m.IsEdit {
		if m.Conn.Name != m.Original {
			// Fetch the stored old PassKey rather than reconstructing —
			// keeps cleanup correct if PassKey ever decouples from Name.
			oldKey := "ssh/" + m.Original
			if oldConn, _, ok := a.Store.Find(m.Original); ok && oldConn.PassKey != "" {
				oldKey = oldConn.PassKey
			}
			_ = a.Pass.Delete(oldKey)
			_ = a.Pass.DeletePassphrase(oldKey)
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
	} else {
		if err := a.Store.Add(m.Conn); err != nil {
			a.Form.Err = err.Error()
			return a, nil
		}
	}

	// Persist the right secret type for the auth method; clear the other.
	switch m.Conn.AuthMethod {
	case "key":
		if m.Secret.Passphrase != "" {
			if err := a.Pass.SetPassphrase(m.Conn.PassKey, m.Secret.Passphrase); err != nil {
				a.Form.Err = err.Error()
				return a, nil
			}
		} else {
			_ = a.Pass.DeletePassphrase(m.Conn.PassKey)
		}
		_ = a.Pass.Delete(m.Conn.PassKey) // clear any stale password
	default:
		if m.Secret.Password != "" {
			if err := a.Pass.Set(m.Conn.PassKey, m.Secret.Password); err != nil {
				a.Form.Err = err.Error()
				return a, nil
			}
		}
		_ = a.Pass.DeletePassphrase(m.Conn.PassKey) // clear any stale passphrase
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
	_ = a.Pass.Delete(a.Pending.PassKey)
	_ = a.Pass.DeletePassphrase(a.Pending.PassKey)
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
	var secret string
	switch c.AuthMethod {
	case "key":
		// Empty passphrase is valid (unencrypted key). Ignore lookup error.
		if pp, err := a.Pass.GetPassphrase(c.PassKey); err == nil {
			secret = pp
		}
	default:
		pwd, err := a.Pass.Get(c.PassKey)
		if err != nil {
			a.List.Err = "pass: " + err.Error()
			return a, nil
		}
		secret = pwd
	}
	a.Pending = c
	a.Mode = ModeConnecting
	return a, tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
		return startSshMsg{conn: c, pwd: secret}
	})
}

func (a AppModel) handleSftp(c config.Connection) (tea.Model, tea.Cmd) {
	auth := sftppkg.Auth{Method: c.AuthMethod}
	switch c.AuthMethod {
	case "key":
		auth.KeyPath = c.KeyPath
		if pp, err := a.Pass.GetPassphrase(c.PassKey); err == nil {
			auth.Passphrase = pp
		}
	default:
		pwd, err := a.Pass.Get(c.PassKey)
		if err != nil {
			a.List.Err = "pass: " + err.Error()
			return a, nil
		}
		auth.Password = pwd
	}
	client, err := sftppkg.Connect(c, auth)
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
	scope := fmt.Sprintf("remote:%s@%s:%d", c.User, c.Host, c.Port)
	a.Sftp = NewSftpModel(client, localHome, remoteHome, a.Bookmark, scope)
	if a.Width > 0 && a.Height > 0 {
		a.Sftp.Width = a.Width
		a.Sftp.Height = a.Height
	}
	a.Mode = ModeSftp
	return a, nil
}

func (a AppModel) View() string {
	w := a.Width
	h := a.Height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	centered := func(inner string) string {
		header := centerBlockHoriz(renderHeader(), w)
		body := centerBlockHoriz(inner, w)
		block := header + "\n\n" + body
		return centerBlockVert(block, h)
	}

	switch a.Mode {
	case ModeForm:
		return centered(a.Form.View())
	case ModeKeygen:
		return centered(a.Keygen.View())
	case ModeSftp:
		return a.Sftp.View()
	case ModeConfirmDelete:
		box := StyleBorder.Render(
			StyleTitle.Render("Delete connection") + "\n\n" +
				fmt.Sprintf("  Delete %q? [y/N]", a.Pending.Name),
		)
		return centered(box)
	case ModeConnecting:
		body := StyleTitle.Render("Connecting") + "\n\n" +
			fmt.Sprintf("  → %s@%s:%d", a.Pending.User, a.Pending.Host, a.Pending.Port) + "\n" +
			StyleHelp.Render("  establishing SSH session...")
		return centered(StyleBorder.Render(body))
	default:
		block := centerBlockHoriz(a.List.View(), w)
		return centerBlockVert(block, h)
	}
}

func centerBlockHoriz(block string, w int) string {
	lines := strings.Split(block, "\n")
	maxW := 0
	for _, l := range lines {
		if lw := lipgloss.Width(l); lw > maxW {
			maxW = lw
		}
	}
	pad := (w - maxW) / 2
	if pad < 0 {
		pad = 0
	}
	p := strings.Repeat(" ", pad)
	for i := range lines {
		lines[i] = p + lines[i]
	}
	return strings.Join(lines, "\n")
}

func centerBlockVert(block string, h int) string {
	lines := strings.Split(block, "\n")
	pad := (h - len(lines)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat("\n", pad) + block
}
