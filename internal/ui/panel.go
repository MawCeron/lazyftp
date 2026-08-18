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
	"github.com/mattn/go-runewidth"
)

// Column widths for the size/date fields; degradation order (drop date,
// then size, keeping only the name) lives in Render.
const (
	sizeColWidth = 8  // "999.9 GB" and under
	dateColWidth = 16 // "2006-01-02 15:04"
	minNameWidth = 15
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
	sizeMetaStyle := lipgloss.NewStyle().Foreground(colorPrimary)
	dateMetaStyle := lipgloss.NewStyle().Foreground(colorMuted)

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

	// nameStyle carries the row's state; sizeStyle/dateStyle match it when
	// the row is marked or selected, so the whole row highlights as one
	// block instead of just the name. Otherwise size and date are each
	// their own shade of metadata, dim enough to read as secondary but
	// distinguishable from one another.
	var nameStyle, sizeStyle, dateStyle lipgloss.Style
	switch {
	case isMarked:
		nameStyle = lipgloss.NewStyle().Foreground(colorMarked).Bold(true).Reverse(true)
		sizeStyle, dateStyle = nameStyle, nameStyle
	case isSelected && fi.file.IsDir():
		nameStyle = lipgloss.NewStyle().Foreground(colorDirectory).Reverse(true)
		sizeStyle, dateStyle = nameStyle, nameStyle
	case isSelected:
		nameStyle = lipgloss.NewStyle().Reverse(true)
		sizeStyle, dateStyle = nameStyle, nameStyle
	case fi.file.IsDir():
		nameStyle = dirStyle
		sizeStyle, dateStyle = sizeMetaStyle, dateMetaStyle
	default:
		nameStyle = normalStyle
		sizeStyle, dateStyle = sizeMetaStyle, dateMetaStyle
	}

	sizeStr := "-"
	if !fi.file.IsDir() {
		sizeStr = formatSize(fi.file.Size)
	}
	dateStr := fi.file.ModTime.Format("2006-01-02 15:04")

	// prefix is 3 display cells (cursor + mark + gap); widthSlack is an
	// empirical safety margin. A real terminal wrapped rows at the reported
	// width with all three columns shown, meaning m.Width() reads wider
	// than what's actually safe to fill.
	// ponytail: widthSlack papers over an unpinned discrepancy between
	// list.Model.Width() and the real usable width instead of tracking it
	// down live; raise it further (or find the exact cause) if wrapping
	// recurs at a specific width/terminal.
	const widthSlack = 3
	avail := m.Width() - 3 - widthSlack
	showDate := avail >= minNameWidth+1+sizeColWidth+1+dateColWidth
	showSize := showDate || avail >= minNameWidth+1+sizeColWidth

	nameW := avail
	if showSize {
		nameW -= 1 + sizeColWidth
	}
	if showDate {
		nameW -= 1 + dateColWidth
	}
	if nameW < 1 {
		nameW = 1
	}

	name = runewidth.Truncate(name, nameW, "...")
	name = runewidth.FillRight(name, nameW)

	row := nameStyle.Render(name)
	if showSize {
		row += " " + sizeStyle.Render(runewidth.FillLeft(sizeStr, sizeColWidth))
	}
	if showDate {
		row += " " + dateStyle.Render(dateStr)
	}

	fmt.Fprint(w, prefix+row)
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
	listHeight := height - 5
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

	body := pathStyle.Render(path) + "\n" + p.list.View()

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
