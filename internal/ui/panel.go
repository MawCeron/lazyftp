package ui

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MawCeron/lazyftp/internal/model"
)

type fileItem struct {
	file model.FileInfo
}

func (f fileItem) Title() string {
	if f.file.IsDir() {
		return f.file.Name + "/"
	}
	return f.file.Name
}
func (f fileItem) Description() string { return "" }
func (f fileItem) FilterValue() string { return f.file.Name }

type fileDelegate struct {
	marked map[int]bool
}

func (d fileDelegate) Height() int                             { return 1 }
func (d fileDelegate) Spacing() int                            { return 0 }
func (d fileDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d fileDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	fi, ok := item.(fileItem)
	if !ok {
		return
	}

	cursorStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	markedStyle := lipgloss.NewStyle().Foreground(colorMarked).Bold(true)
	dirStyle := lipgloss.NewStyle().Foreground(colorDirectory)
	normalStyle := lipgloss.NewStyle().Foreground(colorPrimary)

	isSelected := index == m.Index()
	isMarked := d.marked[index]

	name := fi.file.Name
	if fi.file.IsDir() {
		name = name + "/"
	}

	// Cursor and mark each own a column so neither displaces the other:
	// a marked, selected file shows both the cursor and the mark at once.
	cursorChar := " "
	if isSelected {
		cursorChar = cursorStyle.Render(">")
	}
	markChar := " "
	if isMarked {
		markChar = markedStyle.Render(iconMark())
	}
	prefix := cursorChar + markChar + " "

	var nameRendered string
	switch {
	case isMarked:
		nameRendered = lipgloss.NewStyle().Foreground(colorMarked).Bold(true).Reverse(true).Render(name)
	case isSelected && fi.file.IsDir():
		nameRendered = lipgloss.NewStyle().Foreground(colorDirectory).Reverse(true).Render(name)
	case isSelected:
		nameRendered = lipgloss.NewStyle().Reverse(true).Render(name)
	case fi.file.IsDir():
		nameRendered = dirStyle.Render(name)
	default:
		nameRendered = normalStyle.Render(name)
	}

	fmt.Fprint(w, prefix+nameRendered)
}

type Panel struct {
	title  string
	path   string
	local  bool
	list   list.Model
	marked map[int]bool
	files  []model.FileInfo
}

// Local paths follow the host's rules; remote paths are always POSIX.

func (p Panel) cleanPath(s string) string {
	if p.local {
		return filepath.Clean(s)
	}
	return path.Clean(s)
}

func (p Panel) childPath(name string) string {
	if p.local {
		return filepath.Join(p.path, name)
	}
	return path.Join(p.path, name)
}

func (p Panel) parentPath() string {
	if p.local {
		return filepath.Dir(p.path)
	}
	return path.Dir(p.path)
}

func NewPanel(title string, local bool) Panel {
	delegate := fileDelegate{marked: make(map[int]bool)}
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(colorMuted).PaddingLeft(2)

	return Panel{
		title:  title,
		path:   "/",
		local:  local,
		list:   l,
		marked: make(map[int]bool),
		files:  []model.FileInfo{},
	}
}

func (p Panel) WithFiles(files []model.FileInfo, dir string) Panel {
	dir = p.cleanPath(dir)

	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir() != files[j].IsDir() {
			return files[i].IsDir()
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = fileItem{file: f}
	}

	p.files = files
	p.path = dir
	p.marked = make(map[int]bool)
	p.list.SetItems(items)
	p.list.Select(0)
	p.list.SetDelegate(fileDelegate{marked: p.marked})
	return p
}

func (p Panel) SetSize(width, height int) Panel {
	listWidth := width - 4
	if listWidth < 1 {
		listWidth = 1
	}
	listHeight := height - 6
	if listHeight < 1 {
		listHeight = 1
	}
	p.list.SetSize(listWidth, listHeight)
	return p
}

func (p Panel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {

		case "enter", "l":
			item, ok := p.list.SelectedItem().(fileItem)
			if ok && item.file.IsDir() {
				panel, child := p.title, p.childPath(item.file.Name)
				return p, func() tea.Msg {
					return NavigateMsg{Panel: panel, Path: child}
				}
			}
			// Swallowed even on a non-directory: "l" must never fall through
			// to the list, which binds it to pagination.
			return p, nil

		case "-", "backspace", "h":
			panel, parent := p.title, p.parentPath()
			return p, func() tea.Msg {
				return NavigateMsg{Panel: panel, Path: parent}
			}

		case "r":
			panel, path := p.title, p.path
			return p, func() tea.Msg {
				return NavigateMsg{Panel: panel, Path: path}
			}

		case "space":
			idx := p.list.Index()
			p.marked[idx] = !p.marked[idx]
			if !p.marked[idx] {
				delete(p.marked, idx)
			}
			p.list.SetDelegate(fileDelegate{marked: p.marked})
			return p, nil

		case "t":
			files := p.selectedFiles()
			if len(files) == 0 {
				return p, nil
			}
			panel := p.title
			return p, func() tea.Msg {
				return TransferMsg{SourcePanel: panel, Files: files}
			}
		}
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

func (p Panel) View(width, height int, active bool) string {
	borderColor := colorMuted
	if active {
		borderColor = colorAccent
	}

	pathStyle := lipgloss.NewStyle().Foreground(colorMuted)
	innerWidth := width - 4
	if innerWidth < 1 {
		innerWidth = 1
	}

	path := truncateHead(p.path, innerWidth)

	header := pathStyle.Render(path) + "\n" + strings.Repeat("─", innerWidth)
	body := header + "\n" + p.list.View()

	return borderWithTitle(body, p.title, width, height, borderColor)
}

// selectedFiles is the marked files, or the file under the cursor if none
// are marked.
func (p Panel) selectedFiles() []model.FileInfo {
	if marked := p.markedFiles(); len(marked) > 0 {
		return marked
	}
	item, ok := p.list.SelectedItem().(fileItem)
	if ok {
		return []model.FileInfo{item.file}
	}
	return nil
}

func (p Panel) markedFiles() []model.FileInfo {
	var marked []model.FileInfo
	for i := range p.marked {
		if i < len(p.files) {
			marked = append(marked, p.files[i])
		}
	}
	return marked
}

type NavigateMsg struct {
	Panel string
	Path  string
}

type TransferMsg struct {
	SourcePanel string
	Files       []model.FileInfo
}
