package ui

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	markedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	isSelected := index == m.Index()
	isMarked := d.marked[index]

	name := fi.file.Name
	if fi.file.IsDir() {
		name = name + "/"
	}

	var prefix string
	var nameRendered string

	if isMarked {
		if isSelected {
			prefix = cursorStyle.Render("> ")
		} else {
			prefix = markedStyle.Render("✓ ")
		}
		nameRendered = lipgloss.NewStyle().
			Background(lipgloss.Color("52")).
			Foreground(lipgloss.Color("214")).
			Bold(true).
			Render(name)
	} else {
		if isSelected {
			prefix = cursorStyle.Render("> ")
			if fi.file.IsDir() {
				nameRendered = lipgloss.NewStyle().
					Background(lipgloss.Color("236")).
					Foreground(lipgloss.Color("75")).
					Render(name)
			} else {
				nameRendered = lipgloss.NewStyle().
					Background(lipgloss.Color("236")).
					Render(name)
			}
		} else {
			prefix = "  "
			if fi.file.IsDir() {
				nameRendered = dirStyle.Render(name)
			} else {
				nameRendered = normalStyle.Render(name)
			}
		}
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
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(2)

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
	p.list.SetSize(width-4, height-6)
	return p
}

func (p Panel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "enter", " ":
			item, ok := p.list.SelectedItem().(fileItem)
			if ok && item.file.IsDir() {
				panel, child := p.title, p.childPath(item.file.Name)
				return p, func() tea.Msg {
					return NavigateMsg{Panel: panel, Path: child}
				}
			}

		case "-", "backspace":
			panel, parent := p.title, p.parentPath()
			return p, func() tea.Msg {
				return NavigateMsg{Panel: panel, Path: parent}
			}

		case "x":
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
	borderColor := lipgloss.Color("240")
	if active {
		borderColor = lipgloss.Color("39")
	}

	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	innerWidth := width - 4

	path := p.path
	if len([]rune(path)) > innerWidth {
		path = "…" + string([]rune(path)[len([]rune(path))-innerWidth+1:])
	}

	header := pathStyle.Render(path) + "\n" + strings.Repeat("─", innerWidth)
	body := header + "\n" + p.list.View()

	return borderWithTitle(body, p.title, width, height, borderColor)
}

func (p Panel) selectedFiles() []model.FileInfo {
	if len(p.marked) == 0 {
		item, ok := p.list.SelectedItem().(fileItem)
		if ok {
			return []model.FileInfo{item.file}
		}
		return nil
	}
	var selected []model.FileInfo
	for i := range p.marked {
		if i < len(p.files) {
			selected = append(selected, p.files[i])
		}
	}
	return selected
}

type NavigateMsg struct {
	Panel string
	Path  string
}

type TransferMsg struct {
	SourcePanel string
	Files       []model.FileInfo
}

func formatSize(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%dB", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.1fK", float64(size)/1024)
	case size < 1024*1024*1024:
		return fmt.Sprintf("%.1fM", float64(size)/(1024*1024))
	default:
		return fmt.Sprintf("%.1fG", float64(size)/(1024*1024*1024))
	}
}
