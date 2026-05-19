package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Candratama/tamagosh/internal/bookmark"
	sftppkg "github.com/Candratama/tamagosh/internal/sftp"
)

type Pane int

const (
	PaneLocal Pane = iota
	PaneRemote
)

type SftpQuitMsg struct{}
type SftpErrorMsg struct{ Err error }
type SftpRefreshMsg struct{}
type sftpTickMsg struct{}
type sftpTransferEndMsg struct{}
type editorDoneMsg struct {
	err     error
	pane    Pane
	cleanup string
}

type transferState struct {
	BytesDone  atomic.Int64
	BytesTotal atomic.Int64
	FileIdx    atomic.Int32
	FileTotal  atomic.Int32
	FileName   atomic.Pointer[string]
	Done       atomic.Bool
	Err        atomic.Pointer[string]
	Scanning   atomic.Bool
	Refresh    Pane
}

type SftpModel struct {
	Client         *sftppkg.Client
	LocalDir       string
	RemoteDir      string
	LocalEntries   []sftppkg.Entry
	RemoteEntries  []sftppkg.Entry
	LocalCursor    int
	RemoteCursor   int
	LocalScroll    int
	RemoteScroll   int
	LocalSelected  map[string]bool
	RemoteSelected map[string]bool
	LocalFilter    string
	RemoteFilter   string
	LocalHistory     []string
	RemoteHistory    []string
	LocalCursorMem   map[string]string
	RemoteCursorMem  map[string]string
	Filtering      bool
	ShowHidden     bool
	Active           Pane
	Width            int
	Height           int
	Err              string
	Info             string
	TransferActive bool
	Transfer       *transferState
	ConfirmAction  string
	ConfirmTargets []sftppkg.Entry
	PromptAction   string
	PromptInput    string
	PromptInitial  string
	SortMode       int
	SortAsc        bool
	ShowInfo       bool
	ShowHelp       bool
	HelpScroll     int
	InfoEntry      sftppkg.Entry
	Bookmark       *bookmark.Store
	BookmarkScope  string
	BookmarkList   []string
	BookmarkCursor int
	lastClickPane  Pane
	lastClickIdx   int
	lastClickTime  time.Time
}

const (
	SortName = iota
	SortSize
	SortMTime
)

func NewSftpModel(client *sftppkg.Client, localDir, remoteDir string, bm *bookmark.Store, scope string) SftpModel {
	m := SftpModel{
		Client:          client,
		LocalDir:        localDir,
		RemoteDir:       remoteDir,
		LocalSelected:   map[string]bool{},
		RemoteSelected:  map[string]bool{},
		LocalCursorMem:  map[string]string{},
		RemoteCursorMem: map[string]string{},
		Active:          PaneLocal,
		Width:           80,
		Height:          24,
		SortMode:        SortName,
		SortAsc:         true,
		Bookmark:        bm,
		BookmarkScope:   scope,
	}
	m.refreshLocal()
	m.refreshRemote()
	return m
}

func (m SftpModel) currentScope() string {
	if m.Active == PaneLocal {
		return "local"
	}
	return m.BookmarkScope
}

func (m *SftpModel) refreshLocal() {
	infos, err := os.ReadDir(m.LocalDir)
	if err != nil {
		m.Err = err.Error()
		m.LocalEntries = nil
		return
	}
	entries := make([]sftppkg.Entry, 0, len(infos))
	for _, fi := range infos {
		info, _ := fi.Info()
		size := int64(0)
		var mt time.Time
		if info != nil {
			size = info.Size()
			mt = info.ModTime()
		}
		entries = append(entries, sftppkg.Entry{Name: fi.Name(), IsDir: fi.IsDir(), Size: size, ModTime: mt})
	}
	m.sortEntriesSlice(entries)
	m.LocalEntries = entries
	m.LocalSelected = map[string]bool{}
	if m.LocalCursor >= len(entries) {
		m.LocalCursor = 0
	}
	if m.LocalScroll >= len(entries) {
		m.LocalScroll = 0
	}
}

func (m *SftpModel) refreshRemote() {
	if m.Client == nil {
		return
	}
	entries, err := m.Client.List(m.RemoteDir)
	if err != nil {
		m.Err = err.Error()
		m.RemoteEntries = nil
		return
	}
	m.sortEntriesSlice(entries)
	m.RemoteEntries = entries
	m.RemoteSelected = map[string]bool{}
	if m.RemoteCursor >= len(entries) {
		m.RemoteCursor = 0
	}
	if m.RemoteScroll >= len(entries) {
		m.RemoteScroll = 0
	}
}

func (m SftpModel) sortEntriesSlice(entries []sftppkg.Entry) {
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		var less bool
		switch m.SortMode {
		case SortSize:
			if a.Size == b.Size {
				less = a.Name < b.Name
			} else {
				less = a.Size < b.Size
			}
		case SortMTime:
			if a.ModTime.Equal(b.ModTime) {
				less = a.Name < b.Name
			} else {
				less = a.ModTime.Before(b.ModTime)
			}
		default:
			less = a.Name < b.Name
		}
		if !m.SortAsc {
			return !less
		}
		return less
	})
}

func (m SftpModel) sortLabel() string {
	mode := "name"
	switch m.SortMode {
	case SortSize:
		mode = "size"
	case SortMTime:
		mode = "mtime"
	}
	if m.SortAsc {
		return mode + " ↑"
	}
	return mode + " ↓"
}

func (m SftpModel) visible(pane Pane) []sftppkg.Entry {
	var all []sftppkg.Entry
	var filter string
	if pane == PaneLocal {
		all = m.LocalEntries
		filter = m.LocalFilter
	} else {
		all = m.RemoteEntries
		filter = m.RemoteFilter
	}
	out := make([]sftppkg.Entry, 0, len(all)+1)
	atRoot := false
	if pane == PaneLocal {
		atRoot = filepath.Dir(m.LocalDir) == m.LocalDir
	} else {
		atRoot = sftppkg.Parent(m.RemoteDir) == m.RemoteDir
	}
	if !atRoot && filter == "" {
		out = append(out, sftppkg.Entry{Name: "..", IsDir: true})
	}
	q := strings.ToLower(filter)
	for _, e := range all {
		if !m.ShowHidden && strings.HasPrefix(e.Name, ".") {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(e.Name), q) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (m SftpModel) activeFilter() string {
	if m.Active == PaneLocal {
		return m.LocalFilter
	}
	return m.RemoteFilter
}

func (m SftpModel) Init() tea.Cmd { return nil }

func (m SftpModel) paneBodyHeight() int {
	h := m.Height - 7
	if h < 3 {
		h = 3
	}
	return h
}

func (m *SftpModel) clampScroll() {
	body := m.paneBodyHeight()
	vl := len(m.visible(PaneLocal))
	vr := len(m.visible(PaneRemote))
	if m.LocalCursor >= vl {
		m.LocalCursor = vl - 1
	}
	if m.LocalCursor < 0 {
		m.LocalCursor = 0
	}
	if m.RemoteCursor >= vr {
		m.RemoteCursor = vr - 1
	}
	if m.RemoteCursor < 0 {
		m.RemoteCursor = 0
	}
	if m.LocalCursor < m.LocalScroll {
		m.LocalScroll = m.LocalCursor
	} else if m.LocalCursor >= m.LocalScroll+body {
		m.LocalScroll = m.LocalCursor - body + 1
	}
	if m.LocalScroll < 0 {
		m.LocalScroll = 0
	}
	if m.RemoteCursor < m.RemoteScroll {
		m.RemoteScroll = m.RemoteCursor
	} else if m.RemoteCursor >= m.RemoteScroll+body {
		m.RemoteScroll = m.RemoteCursor - body + 1
	}
	if m.RemoteScroll < 0 {
		m.RemoteScroll = 0
	}
}

func (m *SftpModel) appendFilter(s string) {
	if m.Active == PaneLocal {
		m.LocalFilter += s
		m.LocalCursor = 0
		m.LocalScroll = 0
	} else {
		m.RemoteFilter += s
		m.RemoteCursor = 0
		m.RemoteScroll = 0
	}
}

func (m *SftpModel) popFilter() {
	if m.Active == PaneLocal {
		if len(m.LocalFilter) > 0 {
			m.LocalFilter = m.LocalFilter[:len(m.LocalFilter)-1]
		}
	} else {
		if len(m.RemoteFilter) > 0 {
			m.RemoteFilter = m.RemoteFilter[:len(m.RemoteFilter)-1]
		}
	}
}

func (m *SftpModel) clearFilter() {
	if m.Active == PaneLocal {
		m.LocalFilter = ""
	} else {
		m.RemoteFilter = ""
	}
	m.LocalCursor = 0
	m.RemoteCursor = 0
	m.LocalScroll = 0
	m.RemoteScroll = 0
}

func (m SftpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.clampScroll()
		return m, nil
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case SftpRefreshMsg:
		m.refreshLocal()
		m.refreshRemote()
		return m, nil
	case sftpTickMsg:
		if m.Transfer == nil {
			return m, nil
		}
		if m.Transfer.Done.Load() {
			if errp := m.Transfer.Err.Load(); errp != nil {
				m.Err = *errp
			} else {
				m.Info = fmt.Sprintf("copied %d file(s)", m.Transfer.FileTotal.Load())
				m.clearSelection()
				if m.Transfer.Refresh == PaneRemote {
					m.refreshRemote()
				} else {
					m.refreshLocal()
				}
			}
			return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
				return sftpTransferEndMsg{}
			})
		}
		return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
			return sftpTickMsg{}
		})
	case sftpTransferEndMsg:
		m.TransferActive = false
		m.Transfer = nil
		return m, nil
	case editorDoneMsg:
		if msg.cleanup != "" {
			_ = os.RemoveAll(msg.cleanup)
		}
		if msg.err != nil {
			m.Err = msg.err.Error()
		} else {
			m.Info = "edit saved"
		}
		if msg.pane == PaneLocal {
			m.refreshLocal()
		} else {
			m.refreshRemote()
		}
		return m, tea.ClearScreen
	case tea.KeyMsg:
		if m.ShowInfo {
			m.ShowInfo = false
			return m, nil
		}
		if m.ShowHelp {
			ms := m.helpMaxScroll()
			switch msg.Type {
			case tea.KeyUp:
				if m.HelpScroll > 0 {
					m.HelpScroll--
				}
				return m, nil
			case tea.KeyDown:
				if m.HelpScroll < ms {
					m.HelpScroll++
				}
				return m, nil
			case tea.KeyPgUp:
				m.HelpScroll -= 5
				if m.HelpScroll < 0 {
					m.HelpScroll = 0
				}
				return m, nil
			case tea.KeyPgDown:
				m.HelpScroll += 5
				if m.HelpScroll > ms {
					m.HelpScroll = ms
				}
				return m, nil
			case tea.KeyHome:
				m.HelpScroll = 0
				return m, nil
			case tea.KeyEnd:
				m.HelpScroll = ms
				return m, nil
			}
			m.ShowHelp = false
			m.HelpScroll = 0
			return m, nil
		}
		if len(m.BookmarkList) > 0 {
			return m.handleBookmarkKey(msg)
		}
		if m.PromptAction != "" {
			return m.handlePromptKey(msg)
		}
		if m.ConfirmAction != "" {
			switch msg.Type {
			case tea.KeyEsc:
				m.ConfirmAction = ""
				m.ConfirmTargets = nil
				return m, nil
			case tea.KeyRunes:
				switch string(msg.Runes) {
				case "y", "Y":
					action := m.ConfirmAction
					targets := m.ConfirmTargets
					m.ConfirmAction = ""
					m.ConfirmTargets = nil
					if action == "delete" {
						m.executeDelete(targets)
						return m, nil
					}
					if action == "copy" {
						cmd := m.startCopy(targets)
						return m, cmd
					}
				case "n", "N":
					m.ConfirmAction = ""
					m.ConfirmTargets = nil
					return m, nil
				}
			}
			return m, nil
		}
		if m.Filtering {
			switch msg.Type {
			case tea.KeyEsc:
				m.clearFilter()
				m.Filtering = false
			case tea.KeyEnter:
				m.Filtering = false
			case tea.KeyBackspace:
				m.popFilter()
			case tea.KeyRunes:
				m.appendFilter(string(msg.Runes))
			case tea.KeySpace:
				m.appendFilter(" ")
			}
			m.clampScroll()
			return m, nil
		}
		switch msg.Type {
		case tea.KeyTab:
			if m.Active == PaneLocal {
				m.Active = PaneRemote
			} else {
				m.Active = PaneLocal
			}
		case tea.KeyUp:
			if m.Active == PaneLocal && m.LocalCursor > 0 {
				m.LocalCursor--
			} else if m.Active == PaneRemote && m.RemoteCursor > 0 {
				m.RemoteCursor--
			}
			m.clampScroll()
		case tea.KeyDown:
			vis := m.visible(m.Active)
			if m.Active == PaneLocal && m.LocalCursor < len(vis)-1 {
				m.LocalCursor++
			} else if m.Active == PaneRemote && m.RemoteCursor < len(vis)-1 {
				m.RemoteCursor++
			}
			m.clampScroll()
		case tea.KeyPgUp:
			body := m.paneBodyHeight()
			if m.Active == PaneLocal {
				m.LocalCursor -= body
			} else {
				m.RemoteCursor -= body
			}
			m.clampScroll()
		case tea.KeyPgDown:
			body := m.paneBodyHeight()
			if m.Active == PaneLocal {
				m.LocalCursor += body
			} else {
				m.RemoteCursor += body
			}
			m.clampScroll()
		case tea.KeyHome:
			if m.Active == PaneLocal {
				m.LocalCursor = 0
			} else {
				m.RemoteCursor = 0
			}
			m.clampScroll()
		case tea.KeyEnd:
			vis := m.visible(m.Active)
			if m.Active == PaneLocal {
				m.LocalCursor = len(vis) - 1
			} else {
				m.RemoteCursor = len(vis) - 1
			}
			m.clampScroll()
		case tea.KeyEnter, tea.KeyRight:
			m.descend()
			m.clampScroll()
		case tea.KeyBackspace:
			m.ascend()
			m.clampScroll()
		case tea.KeyLeft:
			if !m.navBack() {
				m.ascend()
			}
			m.clampScroll()
		case tea.KeySpace:
			m.toggleSelect()
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "q":
				return m, func() tea.Msg { return SftpQuitMsg{} }
			case "c":
				targets := m.gatherTargets(false)
				if len(targets) == 0 {
					return m, nil
				}
				if len(targets) > 1 {
					m.ConfirmAction = "copy"
					m.ConfirmTargets = targets
					return m, nil
				}
				cmd := m.startCopy(targets)
				return m, cmd
			case "d":
				targets := m.gatherTargets(false)
				if len(targets) == 0 {
					return m, nil
				}
				m.ConfirmAction = "delete"
				m.ConfirmTargets = targets
				return m, nil
			case "r":
				m.refreshLocal()
				m.refreshRemote()
			case "R":
				m.startPromptRename()
			case "m":
				m.startPrompt("mkdir", "", "create dir: ")
			case "g":
				curDir := m.LocalDir
				if m.Active == PaneRemote {
					curDir = m.RemoteDir
				}
				m.startPrompt("goto", curDir, "goto: ")
			case "s":
				m.SortMode = (m.SortMode + 1) % 3
				m.sortEntriesSlice(m.LocalEntries)
				m.sortEntriesSlice(m.RemoteEntries)
				m.Info = "sort: " + m.sortLabel()
			case "S":
				m.SortAsc = !m.SortAsc
				m.sortEntriesSlice(m.LocalEntries)
				m.sortEntriesSlice(m.RemoteEntries)
				m.Info = "sort: " + m.sortLabel()
			case "i":
				vis := m.visible(m.Active)
				cur := m.LocalCursor
				if m.Active == PaneRemote {
					cur = m.RemoteCursor
				}
				if cur >= 0 && cur < len(vis) {
					m.InfoEntry = vis[cur]
					m.ShowInfo = true
				}
			case "b":
				if m.Bookmark != nil {
					dir := m.LocalDir
					if m.Active == PaneRemote {
						dir = m.RemoteDir
					}
					if err := m.Bookmark.Add(m.currentScope(), dir); err != nil {
						m.Err = err.Error()
					} else {
						m.Info = "bookmarked: " + dir
					}
				}
			case "'":
				if m.Bookmark != nil {
					m.BookmarkList = m.Bookmark.List(m.currentScope())
					m.BookmarkCursor = 0
				}
			case "h", "?":
				m.ShowHelp = true
			case "e":
				return m, m.editCurrent()
			case "x":
				targets := m.gatherTargets(false)
				if len(targets) == 0 {
					return m, nil
				}
				m.ConfirmTargets = targets
				m.startPrompt("chmod", "644", "chmod: ")
			case ".":
				m.ShowHidden = !m.ShowHidden
				m.LocalCursor = 0
				m.RemoteCursor = 0
				m.LocalScroll = 0
				m.RemoteScroll = 0
			case "/":
				m.Filtering = true
				m.clearFilter()
			case "a":
				m.selectAll()
			case "A":
				m.clearSelection()
			}
		}
	}
	return m, nil
}

func (m *SftpModel) selectedAt(pane Pane) map[string]bool {
	if pane == PaneLocal {
		return m.LocalSelected
	}
	return m.RemoteSelected
}

func (m *SftpModel) toggleSelect() {
	vis := m.visible(m.Active)
	var cursor int
	if m.Active == PaneLocal {
		cursor = m.LocalCursor
	} else {
		cursor = m.RemoteCursor
	}
	if cursor < 0 || cursor >= len(vis) {
		return
	}
	e := vis[cursor]
	if e.IsDir {
		return
	}
	sel := m.selectedAt(m.Active)
	if sel[e.Name] {
		delete(sel, e.Name)
	} else {
		sel[e.Name] = true
	}
}

func (m *SftpModel) selectAll() {
	vis := m.visible(m.Active)
	sel := m.selectedAt(m.Active)
	for _, e := range vis {
		if !e.IsDir {
			sel[e.Name] = true
		}
	}
}

func (m *SftpModel) clearSelection() {
	if m.Active == PaneLocal {
		m.LocalSelected = map[string]bool{}
	} else {
		m.RemoteSelected = map[string]bool{}
	}
}

func (m *SftpModel) localGoto(newDir string, pushHistory bool) {
	// remember the name under the cursor before leaving
	vis := m.visible(PaneLocal)
	if m.LocalCursor >= 0 && m.LocalCursor < len(vis) {
		m.LocalCursorMem[m.LocalDir] = vis[m.LocalCursor].Name
	}
	if pushHistory {
		m.LocalHistory = append(m.LocalHistory, m.LocalDir)
	}
	m.LocalDir = newDir
	m.LocalFilter = ""
	m.LocalCursor = 0
	m.LocalScroll = 0
	m.refreshLocal()
	if name, ok := m.LocalCursorMem[newDir]; ok {
		newVis := m.visible(PaneLocal)
		for i, e := range newVis {
			if e.Name == name {
				m.LocalCursor = i
				break
			}
		}
	}
	m.clampScroll()
}

func (m *SftpModel) remoteGoto(newDir string, pushHistory bool) {
	vis := m.visible(PaneRemote)
	if m.RemoteCursor >= 0 && m.RemoteCursor < len(vis) {
		m.RemoteCursorMem[m.RemoteDir] = vis[m.RemoteCursor].Name
	}
	if pushHistory {
		m.RemoteHistory = append(m.RemoteHistory, m.RemoteDir)
	}
	m.RemoteDir = newDir
	m.RemoteFilter = ""
	m.RemoteCursor = 0
	m.RemoteScroll = 0
	m.refreshRemote()
	if name, ok := m.RemoteCursorMem[newDir]; ok {
		newVis := m.visible(PaneRemote)
		for i, e := range newVis {
			if e.Name == name {
				m.RemoteCursor = i
				break
			}
		}
	}
	m.clampScroll()
}

func (m *SftpModel) descend() {
	vis := m.visible(m.Active)
	if m.Active == PaneLocal {
		if m.LocalCursor >= len(vis) {
			return
		}
		e := vis[m.LocalCursor]
		if e.Name == ".." {
			m.ascend()
			return
		}
		if !e.IsDir {
			return
		}
		m.localGoto(filepath.Join(m.LocalDir, e.Name), true)
	} else {
		if m.RemoteCursor >= len(vis) {
			return
		}
		e := vis[m.RemoteCursor]
		if e.Name == ".." {
			m.ascend()
			return
		}
		if !e.IsDir {
			return
		}
		m.remoteGoto(sftppkg.Join(m.RemoteDir, e.Name), true)
	}
}

func (m *SftpModel) ascend() {
	if m.Active == PaneLocal {
		m.localGoto(filepath.Dir(m.LocalDir), true)
	} else {
		m.remoteGoto(sftppkg.Parent(m.RemoteDir), true)
	}
}

func (m *SftpModel) navBack() bool {
	if m.Active == PaneLocal {
		if len(m.LocalHistory) == 0 {
			return false
		}
		prev := m.LocalHistory[len(m.LocalHistory)-1]
		m.LocalHistory = m.LocalHistory[:len(m.LocalHistory)-1]
		m.localGoto(prev, false)
	} else {
		if len(m.RemoteHistory) == 0 {
			return false
		}
		prev := m.RemoteHistory[len(m.RemoteHistory)-1]
		m.RemoteHistory = m.RemoteHistory[:len(m.RemoteHistory)-1]
		m.remoteGoto(prev, false)
	}
	return true
}

func (m *SftpModel) gatherTargets(filesOnly bool) []sftppkg.Entry {
	sel := m.selectedAt(m.Active)
	vis := m.visible(m.Active)
	targets := []sftppkg.Entry{}
	if len(sel) > 0 {
		for _, e := range vis {
			if sel[e.Name] {
				if filesOnly && e.IsDir {
					continue
				}
				targets = append(targets, e)
			}
		}
		return targets
	}
	var cursor int
	if m.Active == PaneLocal {
		cursor = m.LocalCursor
	} else {
		cursor = m.RemoteCursor
	}
	if cursor < 0 || cursor >= len(vis) {
		return nil
	}
	e := vis[cursor]
	if e.Name == ".." {
		return nil
	}
	if filesOnly && e.IsDir {
		return nil
	}
	return []sftppkg.Entry{e}
}

type copyItem struct {
	relPath string
	size    int64
	isDir   bool
}

// safeRelPath rejects relative paths containing ".." segments or absolute paths.
// Returns ("", false) for unsafe paths so the caller can skip without writing.
func safeRelPath(rel string) (string, bool) {
	if rel == "" || rel == "." {
		return rel, true
	}
	if filepath.IsAbs(rel) {
		return "", false
	}
	cleaned := filepath.Clean(rel)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", false
	}
	return cleaned, true
}

func (m *SftpModel) expandCopyItems(targets []sftppkg.Entry) ([]copyItem, int64, int) {
	var items []copyItem
	var total int64
	skipped := 0
	for _, e := range targets {
		if m.Active == PaneLocal {
			base := filepath.Join(m.LocalDir, e.Name)
			if !e.IsDir {
				items = append(items, copyItem{relPath: e.Name, size: e.Size})
				total += e.Size
				continue
			}
			_ = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					skipped++
					return nil
				}
				rel, rerr := filepath.Rel(m.LocalDir, p)
				if rerr != nil {
					skipped++
					return nil
				}
				safe, ok := safeRelPath(rel)
				if !ok {
					skipped++
					return nil
				}
				if d.IsDir() {
					items = append(items, copyItem{relPath: safe, isDir: true})
					return nil
				}
				info, ierr := d.Info()
				var sz int64
				if ierr == nil {
					sz = info.Size()
				}
				items = append(items, copyItem{relPath: safe, size: sz})
				total += sz
				return nil
			})
		} else {
			base := sftppkg.Join(m.RemoteDir, e.Name)
			if !e.IsDir {
				items = append(items, copyItem{relPath: e.Name, size: e.Size})
				total += e.Size
				continue
			}
			prefix := strings.TrimSuffix(m.RemoteDir, "/") + "/"
			if m.RemoteDir == "" || m.RemoteDir == "/" {
				prefix = "/"
			}
			_ = m.Client.Walk(base, func(p string, isDir bool, size int64) error {
				rel := strings.TrimPrefix(p, prefix)
				safe, ok := safeRelPath(rel)
				if !ok {
					skipped++
					return nil
				}
				if isDir {
					items = append(items, copyItem{relPath: safe, isDir: true})
					return nil
				}
				items = append(items, copyItem{relPath: safe, size: size})
				total += size
				return nil
			})
		}
	}
	return items, total, skipped
}

func (m *SftpModel) startCopy(targets []sftppkg.Entry) tea.Cmd {
	if m.Client == nil || len(targets) == 0 {
		return nil
	}

	ts := &transferState{}
	if m.Active == PaneLocal {
		ts.Refresh = PaneRemote
	} else {
		ts.Refresh = PaneLocal
	}
	ts.Scanning.Store(true)

	m.Err = ""
	m.Info = ""
	m.TransferActive = true
	m.Transfer = ts

	dir := m.Active
	client := m.Client
	localDir := m.LocalDir
	remoteDir := m.RemoteDir
	snap := *m
	targetsCopy := append([]sftppkg.Entry{}, targets...)

	go func() {
		items, totalBytes, skipped := snap.expandCopyItems(targetsCopy)
		if skipped > 0 {
			msg := fmt.Sprintf("skipped %d unsafe/unreadable entries", skipped)
			ts.Err.Store(&msg)
		}
		if len(items) == 0 {
			ts.Done.Store(true)
			return
		}
		ts.BytesTotal.Store(totalBytes)
		ts.FileTotal.Store(int32(len(items)))
		first := items[0].relPath
		ts.FileName.Store(&first)
		ts.Scanning.Store(false)

		queue := items
		for i, it := range queue {
			name := it.relPath
			ts.FileName.Store(&name)
			ts.FileIdx.Store(int32(i + 1))
			cb := func(n int64) {
				ts.BytesDone.Add(n)
			}
			var err error
			if it.isDir {
				if dir == PaneLocal {
					target := sftppkg.Join(remoteDir, it.relPath)
					if _, statErr := client.Stat(target); statErr == nil {
						err = nil
					} else {
						err = client.Mkdir(target)
					}
				} else {
					err = os.MkdirAll(filepath.Join(localDir, it.relPath), 0o755)
					if errors.Is(err, fs.ErrExist) {
						err = nil
					}
				}
			} else if dir == PaneLocal {
				src := filepath.Join(localDir, it.relPath)
				dst := sftppkg.Join(remoteDir, it.relPath)
				err = client.UploadProgress(src, dst, cb)
			} else {
				src := sftppkg.Join(remoteDir, it.relPath)
				dst := filepath.Join(localDir, it.relPath)
				err = client.DownloadProgress(src, dst, cb)
			}
			if err != nil {
				es := err.Error()
				ts.Err.Store(&es)
				ts.Done.Store(true)
				return
			}
		}
		ts.Done.Store(true)
	}()

	return tea.Tick(60*time.Millisecond, func(time.Time) tea.Msg {
		return sftpTickMsg{}
	})
}

func (m *SftpModel) executeDelete(targets []sftppkg.Entry) {
	if len(targets) == 0 {
		return
	}
	ok := 0
	for _, e := range targets {
		if m.Active == PaneLocal {
			target := filepath.Join(m.LocalDir, e.Name)
			var err error
			if e.IsDir {
				err = os.RemoveAll(target)
			} else {
				err = os.Remove(target)
			}
			if err != nil {
				m.Err = err.Error()
				if ok > 0 {
					m.Info = fmt.Sprintf("deleted %d file(s) before error", ok)
				}
				m.refreshLocal()
				return
			}
		} else {
			if m.Client == nil {
				return
			}
			target := sftppkg.Join(m.RemoteDir, e.Name)
			if err := m.Client.Delete(target); err != nil {
				m.Err = err.Error()
				if ok > 0 {
					m.Info = fmt.Sprintf("deleted %d file(s) before error", ok)
				}
				m.refreshRemote()
				return
			}
		}
		ok++
	}
	m.Err = ""
	m.Info = fmt.Sprintf("deleted %d item(s)", ok)
	m.clearSelection()
	if m.Active == PaneLocal {
		m.refreshLocal()
	} else {
		m.refreshRemote()
	}
}

func (m SftpModel) View() string {
	paneW := (m.Width - 4) / 2
	if paneW < 24 {
		paneW = 24
	}

	hints := colorHints(shortKeys)
	if m.Filtering {
		hints = fmt.Sprintf("filter (%s): %s_  [Enter] apply  [Esc] cancel", paneLabel(m.Active), m.activeFilter())
	}
	var detail string
	visAct := m.visible(m.Active)
	var curIdx int
	if m.Active == PaneLocal {
		curIdx = m.LocalCursor
	} else {
		curIdx = m.RemoteCursor
	}
	if curIdx >= 0 && curIdx < len(visAct) {
		detail = StyleHelp.Render(paneLabel(m.Active) + " ▸ " + truncate(entryDetail(visAct[curIdx]), m.Width-4))
	}

	help := hints
	if m.TransferActive && m.Transfer != nil {
		done := m.Transfer.BytesDone.Load()
		total := m.Transfer.BytesTotal.Load()
		idx := m.Transfer.FileIdx.Load()
		tot := m.Transfer.FileTotal.Load()
		if m.Transfer.Scanning.Load() {
			name := "scanning…"
			if p := m.Transfer.FileName.Load(); p != nil && *p != "" {
				name = *p
			}
			help = StyleSelected.Render("scanning tree: "+name) + "\n" + help
		} else {
			name := ""
			if p := m.Transfer.FileName.Load(); p != nil {
				name = *p
			}
			pct := 0
			if total > 0 {
				pct = int(done * 100 / total)
				if pct > 100 {
					pct = 100
				}
			}
			bar := transferBar(done, total, 24)
			status := fmt.Sprintf("transferring %d/%d  %s  %d%%  %s/%s  %s",
				idx, tot, bar, pct, humanSize(done), humanSize(total), truncateMiddle(name, 30))
			help = StyleSelected.Render(status) + "\n" + help
		}
	} else if m.Err != "" {
		help = StyleError.Render(m.Err) + "\n" + help
	} else if m.Info != "" {
		help = StyleHelp.Render(m.Info) + "\n" + help
	}
	boxW := m.Width - 2
	if boxW < 30 {
		boxW = 30
	}

	parts := []string{}
	if detail != "" {
		parts = append(parts, detail)
	}
	parts = append(parts, help)
	bottomBlock := strings.Join(parts, "\n")
	bottomH := strings.Count(bottomBlock, "\n") + 1

	contentH := m.Height - bottomH
	if contentH < 6 {
		contentH = 6
	}

	leftTitle := "Local: " + truncate(m.LocalDir, paneW-10)
	rightTitle := "Remote: " + truncate(m.RemoteDir, paneW-11)
	left := m.renderPane(leftTitle, m.visible(PaneLocal), m.LocalCursor, m.LocalScroll, m.LocalSelected, m.LocalFilter, m.Active == PaneLocal, paneW, contentH)
	right := m.renderPane(rightTitle, m.visible(PaneRemote), m.RemoteCursor, m.RemoteScroll, m.RemoteSelected, m.RemoteFilter, m.Active == PaneRemote, paneW, contentH)
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	var overlay string
	switch {
	case m.ShowHelp:
		// reserve some space for borders + bottom rows
		maxRows := m.Height - 6
		if maxRows < 10 {
			maxRows = 10
		}
		overlay = renderHelpBox(min(m.Width-4, 78), maxRows, &m.HelpScroll)
	case m.ShowInfo:
		overlay = renderInfoBox(m.InfoEntry, m.Active, m.LocalDir, m.RemoteDir, min(boxW, 70))
	case len(m.BookmarkList) > 0:
		overlay = renderBookmarkBox(m.BookmarkList, m.BookmarkCursor, min(boxW, 70))
	case m.PromptAction != "":
		overlay = renderPromptBox(m.PromptAction, m.PromptInput, min(boxW, 60))
	case m.ConfirmAction != "":
		overlay = renderConfirmBox(m.ConfirmAction, m.ConfirmTargets, min(boxW, 60))
	}

	if overlay != "" {
		content = overlayCenter(content, overlay, m.Width, contentH)
	}
	return content + "\n" + bottomBlock
}

func (m SftpModel) paneLayout() (paneW int) {
	paneW = (m.Width - 4) / 2
	if paneW < 24 {
		paneW = 24
	}
	return
}

func (m SftpModel) hitTestEntry(x, y int) (Pane, int, bool) {
	paneW := m.paneLayout()
	paneFullW := paneW + 2

	var pane Pane
	var inX int
	if x < paneFullW {
		pane = PaneLocal
		inX = x
	} else if x < paneFullW*2 {
		pane = PaneRemote
		inX = x - paneFullW
	} else {
		return PaneLocal, 0, false
	}
	if inX < 1 || inX >= paneFullW-1 {
		return pane, 0, false
	}

	if y < 1 {
		return pane, 0, false
	}
	contentRow := y - 1
	if contentRow < 1 {
		return pane, 0, false
	}
	entryRow := contentRow - 1

	var scroll int
	var entries []sftppkg.Entry
	if pane == PaneLocal {
		scroll = m.LocalScroll
		entries = m.visible(PaneLocal)
	} else {
		scroll = m.RemoteScroll
		entries = m.visible(PaneRemote)
	}
	idx := entryRow + scroll
	if idx < 0 || idx >= len(entries) {
		return pane, idx, false
	}
	return pane, idx, true
}

func (m SftpModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.ShowHelp {
		ms := m.helpMaxScroll()
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.HelpScroll > 0 {
				m.HelpScroll--
			}
		case tea.MouseButtonWheelDown:
			if m.HelpScroll < ms {
				m.HelpScroll++
			}
		}
		return m, nil
	}
	if m.ShowInfo || m.PromptAction != "" || m.ConfirmAction != "" || len(m.BookmarkList) > 0 {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.Active == PaneLocal && m.LocalCursor > 0 {
			m.LocalCursor--
		} else if m.Active == PaneRemote && m.RemoteCursor > 0 {
			m.RemoteCursor--
		}
		m.clampScroll()
		return m, nil
	case tea.MouseButtonWheelDown:
		vis := m.visible(m.Active)
		if m.Active == PaneLocal && m.LocalCursor < len(vis)-1 {
			m.LocalCursor++
		} else if m.Active == PaneRemote && m.RemoteCursor < len(vis)-1 {
			m.RemoteCursor++
		}
		m.clampScroll()
		return m, nil
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		pane, idx, ok := m.hitTestEntry(msg.X, msg.Y)
		if !ok {
			// click on pane border/title: just switch focus
			paneW := m.paneLayout()
			if msg.X < paneW+2 {
				m.Active = PaneLocal
			} else {
				m.Active = PaneRemote
			}
			return m, nil
		}
		m.Active = pane
		if pane == PaneLocal {
			m.LocalCursor = idx
		} else {
			m.RemoteCursor = idx
		}
		m.clampScroll()
		if m.lastClickPane == pane && m.lastClickIdx == idx && time.Since(m.lastClickTime) < 500*time.Millisecond {
			m.descend()
			m.clampScroll()
			m.lastClickTime = time.Time{}
			return m, nil
		}
		m.lastClickPane = pane
		m.lastClickIdx = idx
		m.lastClickTime = time.Now()
		return m, nil
	case tea.MouseButtonRight:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		pane, idx, ok := m.hitTestEntry(msg.X, msg.Y)
		if !ok {
			return m, nil
		}
		m.Active = pane
		if pane == PaneLocal {
			m.LocalCursor = idx
		} else {
			m.RemoteCursor = idx
		}
		sel := m.selectedAt(pane)
		vis := m.visible(pane)
		if idx < len(vis) && !vis[idx].IsDir {
			name := vis[idx].Name
			if sel[name] {
				delete(sel, name)
			} else {
				sel[name] = true
			}
		}
		m.clampScroll()
		return m, nil
	}
	return m, nil
}

type keyHint struct{ Key, Label string }

var shortKeys = []keyHint{
	{"Tab", "switch"}, {"→", "open"}, {"←", "back"},
	{"Space", "select"}, {"c", "copy"}, {"d", "del"},
	{"/", "find"}, {"h", "elp"}, {"q", "back"},
}

func colorHints(keys []keyHint) string {
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(StyleKeyBracket.Render("["))
		b.WriteString(StyleKey.Render(k.Key))
		b.WriteString(StyleKeyBracket.Render("]"))
		b.WriteString(StyleKeyLabel.Render(k.Label))
	}
	return b.String()
}

// centerTitle renders a title bar centered over body, sized to body's max line width.
// Returns title line + "\n" + body.
func centerTitle(body, title string) string {
	maxW := 0
	for _, line := range strings.Split(body, "\n") {
		if w := lipgloss.Width(line); w > maxW {
			maxW = w
		}
	}
	if maxW < lipgloss.Width(title) {
		maxW = lipgloss.Width(title)
	}
	titleLine := lipgloss.NewStyle().
		Width(maxW).
		Align(lipgloss.Center).
		Bold(true).
		Foreground(lipgloss.Color(gbYellow)).
		Render(title)
	return titleLine + "\n" + body
}

var helpGroups = []struct {
	title string
	keys  []keyHint
}{
	{"Navigation", []keyHint{
		{"Tab", "switch active pane"},
		{"↑/↓", "move cursor"},
		{"→/Enter", "open folder"},
		{"←", "back to previous folder"},
		{"Bksp", "parent folder"},
		{"PgUp/PgDn", "page up/down"},
		{"Home/End", "first/last item"},
		{"g", "goto path (prompt)"},
		{"'", "open bookmark list"},
	}},
	{"Selection", []keyHint{
		{"Space", "toggle select on file"},
		{"a", "select all files in pane"},
		{"A", "clear selection"},
	}},
	{"File operations", []keyHint{
		{"c", "copy (file or directory, recursive)"},
		{"d", "delete (with confirm)"},
		{"R", "rename cursor item"},
		{"m", "make new directory"},
		{"e", "edit in $EDITOR (auto upload if remote)"},
		{"x", "chmod (octal mode, e.g. 644)"},
		{"b", "bookmark current dir"},
	}},
	{"View", []keyHint{
		{"s", "cycle sort: name/size/mtime"},
		{"S", "toggle sort direction"},
		{"/", "search/filter pane"},
		{".", "toggle hidden files"},
		{"i", "show file info"},
		{"r", "refresh both panes"},
	}},
	{"Mouse", []keyHint{
		{"left-click", "focus pane + move cursor"},
		{"double-click", "open folder under cursor"},
		{"right-click", "toggle select on file"},
		{"wheel up/dn", "scroll cursor in active pane"},
	}},
	{"System", []keyHint{
		{"h or ?", "this help"},
		{"q", "back to connection list"},
	}},
}

// helpTotalLines counts rendered lines including section headers and inter-group blanks.
func helpTotalLines() int {
	n := 0
	for gi, g := range helpGroups {
		if gi > 0 {
			n++ // blank separator
		}
		n++ // section title
		n += len(g.keys)
	}
	return n
}

func (m SftpModel) helpMaxScroll() int {
	maxRows := m.Height - 6
	if maxRows < 10 {
		maxRows = 10
	}
	visible := maxRows - 2
	if visible < 5 {
		visible = 5
	}
	ms := helpTotalLines() - visible
	if ms < 0 {
		ms = 0
	}
	return ms
}

func renderHelpBox(width, maxRows int, scroll *int) string {
	groups := helpGroups

	var lines []string
	for gi, g := range groups {
		if gi > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, StyleSection.Render(g.title))
		for _, k := range g.keys {
			line := StyleKeyBracket.Render("[") +
				StyleKey.Render(k.Key) +
				StyleKeyBracket.Render("] ") +
				StyleKeyLabel.Render(k.Label)
			lines = append(lines, line)
		}
	}

	total := len(lines)
	// reserve 2 lines: scroll indicator + close hint
	visible := maxRows - 2
	if visible < 5 {
		visible = 5
	}

	if *scroll < 0 {
		*scroll = 0
	}
	maxScroll := total - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if *scroll > maxScroll {
		*scroll = maxScroll
	}

	end := *scroll + visible
	if end > total {
		end = total
	}
	chunk := lines[*scroll:end]

	body := strings.Join(chunk, "\n")
	if total > visible {
		body += "\n" + StyleHelp.Render(fmt.Sprintf("  [%d-%d/%d]  ↑/↓ scroll  PgUp/PgDn page", *scroll+1, end, total))
	}
	body += "\n" + StyleHelp.Render("press any other key to close")
	_ = width
	return StyleBorder.Render(centerTitle(body, "Keyboard shortcuts"))
}

func renderConfirmBox(action string, targets []sftppkg.Entry, maxWidth int) string {
	var b strings.Builder
	title := action
	if len(title) > 0 {
		title = strings.ToUpper(title[:1]) + title[1:]
	}
	b.WriteString(StyleNormal.Render(fmt.Sprintf("%s %d item(s)?", action, len(targets))))
	b.WriteString("\n\n")
	maxList := 6
	for i, e := range targets {
		if i >= maxList {
			b.WriteString(StyleHelp.Render(fmt.Sprintf("… +%d more", len(targets)-maxList)))
			b.WriteString("\n")
			break
		}
		name := e.Name
		if e.IsDir {
			name += "/"
		}
		b.WriteString(StyleNormal.Render("- " + truncate(name, maxWidth-6)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(StyleKeyBracket.Render("["))
	b.WriteString(StyleError.Render("y"))
	b.WriteString(StyleKeyBracket.Render("] "))
	b.WriteString(StyleKeyLabel.Render("yes  "))
	b.WriteString(StyleKeyBracket.Render("["))
	b.WriteString(StyleKey.Render("N"))
	b.WriteString(StyleKeyBracket.Render("/"))
	b.WriteString(StyleKey.Render("Esc"))
	b.WriteString(StyleKeyBracket.Render("] "))
	b.WriteString(StyleKeyLabel.Render("cancel"))
	return StyleConfirm.Render(centerTitle(b.String(), title+" confirmation"))
}

func overlayCenter(base, box string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	boxLines := strings.Split(box, "\n")
	for len(baseLines) < height {
		baseLines = append(baseLines, "")
	}
	bl := len(baseLines)
	btl := len(boxLines)
	if btl > bl {
		btl = bl
		boxLines = boxLines[:bl]
	}
	startRow := (bl - btl) / 2
	if startRow < 0 {
		startRow = 0
	}
	for i := 0; i < btl && startRow+i < bl; i++ {
		boxLine := boxLines[i]
		bw := lipgloss.Width(boxLine)
		col := (width - bw) / 2
		if col < 0 {
			col = 0
		}
		baseLine := baseLines[startRow+i]
		baseW := lipgloss.Width(baseLine)
		if baseW < width {
			baseLine += strings.Repeat(" ", width-baseW)
			baseW = width
		}
		leftPart := ansi.Cut(baseLine, 0, col)
		rightPart := ansi.Cut(baseLine, col+bw, baseW)
		baseLines[startRow+i] = leftPart + boxLine + rightPart
	}
	return strings.Join(baseLines, "\n")
}

func overlayBottom(base, box string) string {
	baseLines := strings.Split(base, "\n")
	boxLines := strings.Split(box, "\n")
	bl := len(baseLines)
	btl := len(boxLines)
	start := bl - btl
	if start < 0 {
		start = 0
	}
	for i := 0; i < btl && start+i < bl; i++ {
		baseLines[start+i] = boxLines[i]
	}
	return strings.Join(baseLines, "\n")
}

func renderInfoBox(e sftppkg.Entry, pane Pane, localDir, remoteDir string, maxWidth int) string {
	kind := "file"
	if e.IsDir {
		kind = "directory"
	}
	mt := "-"
	if !e.ModTime.IsZero() {
		mt = e.ModTime.Local().Format("2006-01-02 15:04:05")
	}
	dir := localDir
	if pane == PaneRemote {
		dir = remoteDir
	}
	full := dir + "/" + e.Name
	body := StyleNormal.Render(fmt.Sprintf("name : %s", e.Name)) + "\n" +
		StyleNormal.Render(fmt.Sprintf("kind : %s", kind)) + "\n" +
		StyleNormal.Render(fmt.Sprintf("size : %s (%d bytes)", humanSize(e.Size), e.Size)) + "\n" +
		StyleNormal.Render(fmt.Sprintf("mtime: %s", mt)) + "\n" +
		StyleNormal.Render(fmt.Sprintf("path : %s", truncate(full, maxWidth-12))) + "\n" +
		StyleNormal.Render(fmt.Sprintf("pane : %s", paneLabel(pane))) + "\n" +
		"\n" + StyleHelp.Render("[any key] close")
	return StyleBorder.Render(centerTitle(body, "File info"))
}

func renderBookmarkBox(list []string, cursor, maxWidth int) string {
	var b strings.Builder
	if len(list) == 0 {
		b.WriteString(StyleHelp.Render("(empty — press [b] in pane to add)"))
		b.WriteString("\n")
	}
	max := 6
	start := 0
	if cursor >= max {
		start = cursor - max + 1
	}
	end := start + max
	if end > len(list) {
		end = len(list)
	}
	for i := start; i < end; i++ {
		name := truncate(list[i], maxWidth-4)
		if i == cursor {
			b.WriteString(StyleSelected.Render("▸ " + name))
		} else {
			b.WriteString(StyleNormal.Render("  " + name))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(StyleHelp.Render("[↑↓] move  [Enter] jump  [d] del  [Esc] close"))
	title := fmt.Sprintf("Bookmarks (%d)", len(list))
	return StyleBorder.Render(centerTitle(b.String(), title))
}

func renderPromptBox(action, input string, _ int) string {
	body := StyleSelected.Render(input+"_") + "\n\n" +
		StyleHelp.Render("[Enter] confirm  [Esc] cancel")
	return StyleBorder.Render(centerTitle(body, action))
}

func transferBar(done, total int64, width int) string {
	if width <= 0 {
		return ""
	}
	if total <= 0 {
		return "[" + strings.Repeat("-", width) + "]"
	}
	filled := int(done * int64(width) / total)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return "[" + strings.Repeat("=", filled) + strings.Repeat("-", width-filled) + "]"
}

func paneLabel(p Pane) string {
	if p == PaneLocal {
		return "local"
	}
	return "remote"
}

func (m SftpModel) renderPane(title string, entries []sftppkg.Entry, cursor, scroll int, selected map[string]bool, filter string, active bool, width, height int) string {
	innerH := height - 2
	body := innerH - 3
	if body < 1 {
		body = 1
	}

	titleBg := gbBorder
	titleFg := gbFgMute
	if active {
		titleBg = gbGreen
		titleFg = "#1d2021"
	}
	titleStyle := lipgloss.NewStyle().
		Width(width-2).
		Align(lipgloss.Center).
		Background(lipgloss.Color(titleBg)).
		Foreground(lipgloss.Color(titleFg)).
		Bold(true)

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	nameW := width - 8
	if nameW < 8 {
		nameW = 8
	}

	if len(entries) == 0 {
		b.WriteString(StyleHelp.Render("  (empty)"))
		b.WriteString("\n")
	} else {
		end := scroll + body
		if end > len(entries) {
			end = len(entries)
		}
		for i := scroll; i < end; i++ {
			e := entries[i]
			name := e.Name
			if e.IsDir {
				name += "/"
			}
			name = truncateMiddle(name, nameW)
			marker := " "
			if selected[e.Name] {
				marker = "*"
			}
			line := fmt.Sprintf(" %s %s", marker, name)
			if i == cursor {
				line = StyleSelected.Render("▸" + marker + " " + name)
			} else if selected[e.Name] {
				line = StyleError.Render(line)
			} else {
				line = StyleNormal.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	footer := ""
	if filter != "" {
		footer = fmt.Sprintf("filter: %s", filter)
	}
	if len(entries) > 0 {
		count := fmt.Sprintf("[%d/%d]", cursor+1, len(entries))
		if len(selected) > 0 {
			count += fmt.Sprintf(" sel:%d", len(selected))
		}
		if footer != "" {
			footer = footer + "  " + count
		} else {
			footer = count
		}
	}
	if footer != "" {
		b.WriteString(StyleHelp.Render(truncate(footer, width-4)))
	}

	rendered := b.String()
	currentLines := strings.Count(rendered, "\n") + 1
	for currentLines < innerH {
		rendered += "\n"
		currentLines++
	}

	style := StylePaneInactive
	if active {
		style = StylePaneActive
	}
	return style.Width(width).Height(innerH).Render(rendered)
}

func (m SftpModel) handleBookmarkKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.BookmarkList = nil
		return m, nil
	case tea.KeyUp:
		if m.BookmarkCursor > 0 {
			m.BookmarkCursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.BookmarkCursor < len(m.BookmarkList)-1 {
			m.BookmarkCursor++
		}
		return m, nil
	case tea.KeyEnter:
		if m.BookmarkCursor < len(m.BookmarkList) {
			dir := m.BookmarkList[m.BookmarkCursor]
			if m.Active == PaneLocal {
				m.LocalHistory = append(m.LocalHistory, m.LocalDir)
				m.LocalDir = dir
				m.LocalFilter = ""
				m.refreshLocal()
			} else {
				m.RemoteHistory = append(m.RemoteHistory, m.RemoteDir)
				m.RemoteDir = dir
				m.RemoteFilter = ""
				m.refreshRemote()
			}
		}
		m.BookmarkList = nil
		return m, nil
	case tea.KeyRunes:
		if string(msg.Runes) == "d" && m.BookmarkCursor < len(m.BookmarkList) {
			dir := m.BookmarkList[m.BookmarkCursor]
			if m.Bookmark != nil {
				m.Bookmark.Remove(m.currentScope(), dir)
			}
			m.BookmarkList = m.Bookmark.List(m.currentScope())
			if m.BookmarkCursor >= len(m.BookmarkList) && m.BookmarkCursor > 0 {
				m.BookmarkCursor--
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *SftpModel) startPrompt(action, initial, _ string) {
	m.PromptAction = action
	m.PromptInput = initial
	m.PromptInitial = initial
}

func (m *SftpModel) startPromptRename() {
	vis := m.visible(m.Active)
	cur := m.LocalCursor
	if m.Active == PaneRemote {
		cur = m.RemoteCursor
	}
	if cur < 0 || cur >= len(vis) {
		return
	}
	m.startPrompt("rename", vis[cur].Name, "rename: ")
}

func (m SftpModel) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.PromptAction = ""
		m.PromptInput = ""
		m.PromptInitial = ""
		m.ConfirmTargets = nil
		return m, nil
	case tea.KeyEnter:
		action := m.PromptAction
		input := strings.TrimSpace(m.PromptInput)
		m.PromptAction = ""
		m.PromptInput = ""
		initial := m.PromptInitial
		m.PromptInitial = ""
		if input == "" {
			return m, nil
		}
		switch action {
		case "rename":
			m.execRename(initial, input)
		case "mkdir":
			m.execMkdir(input)
		case "goto":
			m.execGoto(input)
		case "chmod":
			m.execChmod(input)
		}
		return m, nil
	case tea.KeyBackspace:
		if len(m.PromptInput) > 0 {
			m.PromptInput = m.PromptInput[:len(m.PromptInput)-1]
		}
		return m, nil
	case tea.KeySpace:
		m.PromptInput += " "
		return m, nil
	case tea.KeyRunes:
		m.PromptInput += string(msg.Runes)
		return m, nil
	}
	return m, nil
}

func (m *SftpModel) execRename(oldName, newName string) {
	if oldName == newName {
		return
	}
	if m.Active == PaneLocal {
		oldP := filepath.Join(m.LocalDir, oldName)
		newP := filepath.Join(m.LocalDir, newName)
		if err := os.Rename(oldP, newP); err != nil {
			m.Err = err.Error()
			return
		}
		m.refreshLocal()
	} else {
		if m.Client == nil {
			return
		}
		oldP := sftppkg.Join(m.RemoteDir, oldName)
		newP := sftppkg.Join(m.RemoteDir, newName)
		if err := m.Client.Rename(oldP, newP); err != nil {
			m.Err = err.Error()
			return
		}
		m.refreshRemote()
	}
	m.Info = "renamed"
}

func (m *SftpModel) execMkdir(name string) {
	if m.Active == PaneLocal {
		path := filepath.Join(m.LocalDir, name)
		if err := os.Mkdir(path, 0o755); err != nil {
			m.Err = err.Error()
			return
		}
		m.refreshLocal()
	} else {
		if m.Client == nil {
			return
		}
		path := sftppkg.Join(m.RemoteDir, name)
		if err := m.Client.Mkdir(path); err != nil {
			m.Err = err.Error()
			return
		}
		m.refreshRemote()
	}
	m.Info = "created: " + name
}

func (m *SftpModel) execChmod(modeStr string) {
	mode64, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		m.Err = "invalid mode: " + modeStr
		return
	}
	mode := os.FileMode(mode64)
	targets := m.ConfirmTargets
	m.ConfirmTargets = nil
	if len(targets) == 0 {
		return
	}
	ok := 0
	for _, e := range targets {
		if m.Active == PaneLocal {
			p := filepath.Join(m.LocalDir, e.Name)
			if err := os.Chmod(p, mode); err != nil {
				m.Err = err.Error()
				return
			}
		} else {
			if m.Client == nil {
				return
			}
			p := sftppkg.Join(m.RemoteDir, e.Name)
			if err := m.Client.Chmod(p, mode); err != nil {
				m.Err = err.Error()
				return
			}
		}
		ok++
	}
	m.Info = fmt.Sprintf("chmod %o on %d item(s)", mode, ok)
	if m.Active == PaneLocal {
		m.refreshLocal()
	} else {
		m.refreshRemote()
	}
}

func (m *SftpModel) editCurrent() tea.Cmd {
	vis := m.visible(m.Active)
	var cur int
	if m.Active == PaneLocal {
		cur = m.LocalCursor
	} else {
		cur = m.RemoteCursor
	}
	if cur < 0 || cur >= len(vis) {
		return nil
	}
	e := vis[cur]
	if e.IsDir {
		m.Err = "can't edit directory"
		return nil
	}

	candidates := []string{}
	if v := os.Getenv("VISUAL"); v != "" {
		candidates = append(candidates, v)
	}
	if e := os.Getenv("EDITOR"); e != "" {
		candidates = append(candidates, e)
	}
	candidates = append(candidates, "nvim", "vim", "nano", "vi")

	editor := ""
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			editor = c
			break
		}
	}
	if editor == "" {
		m.Err = "no editor found (set $EDITOR or install nvim/nano)"
		return nil
	}

	pane := m.Active
	var localPath, remotePath, cleanup string

	if pane == PaneLocal {
		localPath = filepath.Join(m.LocalDir, e.Name)
	} else {
		if m.Client == nil {
			return nil
		}
		tmpDir, err := os.MkdirTemp("", "tamagosh-edit-*")
		if err != nil {
			m.Err = err.Error()
			return nil
		}
		localPath = filepath.Join(tmpDir, e.Name)
		remotePath = sftppkg.Join(m.RemoteDir, e.Name)
		if err := m.Client.Download(remotePath, localPath); err != nil {
			os.RemoveAll(tmpDir)
			m.Err = err.Error()
			return nil
		}
		cleanup = tmpDir
	}

	client := m.Client
	// snapshot file state before editor so we can detect changes
	var beforeMTime time.Time
	var beforeSize int64
	if info, ierr := os.Stat(localPath); ierr == nil {
		beforeMTime = info.ModTime()
		beforeSize = info.Size()
	}
	// Bridge the alt-screen exit: dump the current SFTP frame to main buffer
	// so user sees the pane briefly instead of scrollback during the transition.
	frame := m.View()
	if cleanup == "" {
		// local file: no tmp dir created above; make one just for the frame snapshot
		td, terr := os.MkdirTemp("", "tamagosh-frame-*")
		if terr == nil {
			cleanup = td
		}
	}
	framePath := ""
	if cleanup != "" {
		framePath = filepath.Join(cleanup, "frame.txt")
		_ = os.WriteFile(framePath, []byte(frame), 0o600)
	}
	// 3J clears scrollback, 2J clears visible, H homes cursor → blank main buffer.
	// Then cat the snapshot so the SFTP pane appears, brief sleep, clear again,
	// then exec the editor.
	wrapper := `printf '\033[3J\033[2J\033[H'; [ -n "$2" ] && cat "$2"; sleep 0.18; printf '\033[3J\033[2J\033[H'; exec "$0" "$1"`
	cmd := exec.Command("sh", "-c", wrapper, editor, localPath, framePath)
	savedPath := localPath
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return editorDoneMsg{err: err, pane: pane, cleanup: cleanup}
		}
		if pane == PaneRemote {
			// skip upload if file unchanged (no save in editor)
			changed := true
			if info, ierr := os.Stat(localPath); ierr == nil {
				if info.ModTime().Equal(beforeMTime) && info.Size() == beforeSize {
					changed = false
				}
			}
			if !changed {
				return editorDoneMsg{pane: pane, cleanup: cleanup}
			}
			if uerr := client.Upload(localPath, remotePath); uerr != nil {
				return editorDoneMsg{
					err:     fmt.Errorf("upload failed (edits kept at %s): %w", savedPath, uerr),
					pane:    pane,
					cleanup: "",
				}
			}
		}
		return editorDoneMsg{pane: pane, cleanup: cleanup}
	})
}

func (m *SftpModel) execGoto(path string) {
	if m.Active == PaneLocal {
		if _, err := os.Stat(path); err != nil {
			m.Err = err.Error()
			return
		}
		m.LocalHistory = append(m.LocalHistory, m.LocalDir)
		m.LocalDir = path
		m.LocalFilter = ""
		m.refreshLocal()
	} else {
		if m.Client == nil {
			return
		}
		if _, err := m.Client.Stat(path); err != nil {
			m.Err = err.Error()
			return
		}
		m.RemoteHistory = append(m.RemoteHistory, m.RemoteDir)
		m.RemoteDir = path
		m.RemoteFilter = ""
		m.refreshRemote()
	}
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return "..." + s[len(s)-(n-3):]
}

func truncateMiddle(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	left := (n - 3) / 2
	right := n - 3 - left
	return s[:left] + "..." + s[len(s)-right:]
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	suffixes := "KMGTPE"
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit && exp < len(suffixes)-1; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), suffixes[exp])
}

func entryDetail(e sftppkg.Entry) string {
	kind := "file"
	if e.IsDir {
		kind = "dir "
	}
	mt := "-"
	if !e.ModTime.IsZero() {
		mt = e.ModTime.Local().Format("2006-01-02 15:04")
	}
	if e.IsDir {
		return fmt.Sprintf("%s  %s  %s", kind, mt, e.Name)
	}
	return fmt.Sprintf("%s  %s  %s  %s", kind, humanSize(e.Size), mt, e.Name)
}
