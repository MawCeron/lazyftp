package ui

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
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
	// Keyed by name, not list index -- an index only means "this file"
	// until the list is next reordered or filtered (sorting, #30; fuzzy
	// filtering, #31), at which point the same index would point at a
	// different file.
	marked map[string]bool

	// uniqueOnly names this panel's entries missing from the other panel;
	// sizeDiffers names entries present on both sides whose size doesn't
	// match (--highlight-diff). The two are mutually exclusive by
	// construction -- sizeDiffers only ever considers names that exist on
	// both sides -- so they share one indicator column instead of each
	// costing their own. nil means the flag is off, distinct from a
	// non-nil empty map (flag on, nothing differs) -- nil also drops the
	// column entirely rather than reserving a permanently blank one nobody
	// asked for.
	uniqueOnly  map[string]bool
	sizeDiffers map[string]bool
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
	uniqueStyle := lipgloss.NewStyle().Foreground(colorDiffOnly).Bold(true)
	sizeDiffersStyle := lipgloss.NewStyle().Foreground(colorSizeDiffers).Bold(true)
	dirStyle := lipgloss.NewStyle().Foreground(colorDirectory)
	normalStyle := lipgloss.NewStyle().Foreground(colorPrimary)
	sizeMetaStyle := lipgloss.NewStyle().Foreground(colorPrimary)
	dateMetaStyle := lipgloss.NewStyle().Foreground(colorMuted)

	isSelected := index == m.Index()
	isMarked := d.marked[fi.file.Name]
	showDiffCol := d.uniqueOnly != nil
	isUnique := d.uniqueOnly[fi.file.Name]
	sizeDiffers := d.sizeDiffers[fi.file.Name]

	name := fi.file.Name
	if fi.file.IsDir() {
		name = name + "/"
	}

	// Cursor, mark and (when --highlight-diff is on) the diff indicator
	// each own a column so none displaces another: a marked, selected,
	// unique file shows all three markers at once.
	cursorChar := " "
	if isSelected {
		cursorChar = cursorStyle.Render(">")
	}
	markChar := " "
	if isMarked {
		markChar = markedStyle.Render(iconMark())
	}
	prefix := cursorChar + markChar
	if showDiffCol {
		diffChar := " "
		switch {
		case isUnique:
			diffChar = uniqueStyle.Render(iconUnique())
		case sizeDiffers:
			diffChar = sizeDiffersStyle.Render(iconSizeDiffers())
		}
		prefix += diffChar
	}
	prefix += " "

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
	case isUnique:
		nameStyle = uniqueStyle
		sizeStyle, dateStyle = sizeMetaStyle, dateMetaStyle
	case sizeDiffers:
		nameStyle = sizeDiffersStyle
		sizeStyle, dateStyle = sizeDiffersStyle, dateMetaStyle
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

	// prefix is 3 display cells (cursor + mark + gap), 4 with the diff
	// column; widthSlack is an empirical safety margin. A real terminal
	// wrapped rows at the reported width with all three columns shown,
	// meaning m.Width() reads wider than what's actually safe to fill.
	// ponytail: widthSlack papers over an unpinned discrepancy between
	// list.Model.Width() and the real usable width instead of tracking it
	// down live; raise it further (or find the exact cause) if wrapping
	// recurs at a specific width/terminal.
	const widthSlack = 3
	prefixWidth := 3
	if showDiffCol {
		prefixWidth = 4
	}
	avail := m.Width() - prefixWidth - widthSlack
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
	title    string
	path     string
	local    bool
	list     list.Model
	marked   map[string]bool
	files    []model.FileInfo
	sortBy   sortColumn
	sortDesc bool

	jumping   bool
	jumpInput textinput.Model
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

// resolveJumpPath treats input as absolute if it's rooted by the panel's own
// convention (a leading "/" for remote's always-POSIX paths, filepath.IsAbs
// for local's host rules), otherwise resolves it against the current
// directory -- the same way a relative path works in a shell.
func (p Panel) resolveJumpPath(input string) string {
	isAbs := strings.HasPrefix(input, "/")
	if p.local {
		isAbs = filepath.IsAbs(input)
	}
	if isAbs {
		return p.cleanPath(input)
	}
	return p.childPath(input)
}

func NewPanel(title string, local bool) Panel {
	delegate := fileDelegate{marked: make(map[string]bool)}
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	// Filtering delegates to bubbles' own fuzzy-match implementation
	// (sahilm/fuzzy is already in the dependency graph) and its built-in
	// "/", Esc, Enter keymap -- see the guard in Update below that lets
	// those keys reach the list uninterrupted while a query is being typed.
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(colorMuted).PaddingLeft(2)

	jump := textinput.New()
	jump.Prompt = ""

	return Panel{
		title:     title,
		path:      "/",
		local:     local,
		list:      l,
		marked:    make(map[string]bool),
		files:     []model.FileInfo{},
		jumpInput: jump,
	}
}

// WithFiles returns the command applySort's SetItems produces, which callers
// must run: with filtering enabled (#31), the list needs it to rebuild the
// filtered view against the new items, or a panel reloaded while a filter is
// active (via refresh, navigation, or a completed transfer) would show no
// files at all.
func (p Panel) WithFiles(files []model.FileInfo, dir string) (Panel, tea.Cmd) {
	p.files = files
	p.path = p.cleanPath(dir)
	p.marked = make(map[string]bool)
	p, cmd := p.applySort()
	p.list.Select(0)
	p.list.SetDelegate(fileDelegate{marked: p.marked})
	return p, cmd
}

// applySort re-sorts p.files by the panel's current sort settings and
// rebuilds the list's items to match, in place. It does not touch the
// cursor -- callers decide what that means for them (WithFiles resets it
// to the top for a new directory; resort below keeps it on the same file).
//
// The returned command must be run: with filtering enabled (#31), the list
// needs it to rebuild the filtered view against the resorted items, or
// resorting while a filter is active would show no files at all -- the same
// failure WithFiles guards against.
func (p Panel) applySort() (Panel, tea.Cmd) {
	sortFiles(p.files, p.sortBy, p.sortDesc)
	items := make([]list.Item, len(p.files))
	for i, f := range p.files {
		items[i] = fileItem{file: f}
	}
	cmd := p.list.SetItems(items)
	return p, cmd
}

// resort re-applies the current sort after sortBy/sortDesc changed, keeping
// the cursor on the same file instead of snapping back to the top.
func (p Panel) resort() (Panel, tea.Cmd) {
	item, hadSelection := p.list.SelectedItem().(fileItem)
	p, cmd := p.applySort()
	if hadSelection {
		for i, f := range p.files {
			if f.Name == item.file.Name {
				p.list.Select(i)
				break
			}
		}
	}
	return p, cmd
}

func (p Panel) SetSize(width, height int) Panel {
	listWidth := borderInteriorWidth(width)
	if listWidth < 1 {
		listWidth = 1
	}
	listHeight := height - 3
	if listHeight < 1 {
		listHeight = 1
	}
	p.list.SetSize(listWidth, listHeight)
	// listWidth minus 2 for the ": " prompt drawn beside it in View.
	p.jumpInput.SetWidth(max(listWidth-2, 1))
	return p
}

// Filtering reports whether the list is actively capturing filter query
// input (the user is typing after "/"). While true, callers must let keys
// reach the list unintercepted -- otherwise a query character that
// happens to match a bound key (e.g. "q" while quit is bound) fires that
// action instead of being typed into the filter.
func (p Panel) Filtering() bool {
	return p.list.SettingFilter()
}

// HasFilter reports whether a filter is active at all, whether still being
// typed or already applied. Esc must reach the list in both states so
// bubbles' own filter keymap can cancel (while typing) or clear (once
// applied) it -- see acceptance criteria on #31.
func (p Panel) HasFilter() bool {
	return p.list.FilterState() != list.Unfiltered
}

func (p Panel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if p.jumping {
			switch {
			case key.Matches(msg, keyJumpGo):
				p.jumping = false
				target := strings.TrimSpace(p.jumpInput.Value())
				if target == "" {
					return p, nil
				}
				panel, dest := p.title, p.resolveJumpPath(target)
				return p, func() tea.Msg {
					return NavigateMsg{Panel: panel, Path: dest}
				}
			case key.Matches(msg, keyJumpCancel):
				p.jumping = false
				return p, nil
			}
			var cmd tea.Cmd
			p.jumpInput, cmd = p.jumpInput.Update(msg)
			return p, cmd
		}

		// While a filter query is being typed, every key belongs to the
		// list's own filter input -- none of lazyftp's bindings below
		// (which include letters like "l"/"h"/"t"/"r" and space, and ":" to
		// jump) may intercept it, or normal filter text couldn't be typed.
		if !p.list.SettingFilter() {
			switch {

			case key.Matches(msg, keyJump):
				p.jumping = true
				p.jumpInput.SetValue("")
				p.jumpInput.Focus()
				return p, nil

			case key.Matches(msg, keyOpen):
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

			case key.Matches(msg, keyUp):
				panel, parent := p.title, p.parentPath()
				return p, func() tea.Msg {
					return NavigateMsg{Panel: panel, Path: parent}
				}

			case key.Matches(msg, keyRefresh):
				panel, path := p.title, p.path
				return p, func() tea.Msg {
					return NavigateMsg{Panel: panel, Path: path}
				}

			case key.Matches(msg, keyMark):
				item, ok := p.list.SelectedItem().(fileItem)
				if !ok {
					return p, nil
				}
				name := item.file.Name
				p.marked[name] = !p.marked[name]
				if !p.marked[name] {
					delete(p.marked, name)
				}
				p.list.SetDelegate(fileDelegate{marked: p.marked})
				return p, nil

			case key.Matches(msg, keyTransfer):
				files := p.selectedFiles()
				if len(files) == 0 {
					return p, nil
				}
				panel := p.title
				return p, func() tea.Msg {
					return TransferMsg{SourcePanel: panel, Files: files}
				}

			case key.Matches(msg, keySortNext):
				p.sortBy = p.sortBy.next()
				p.sortDesc = false
				return p.resort()

			case key.Matches(msg, keySortFlip):
				p.sortDesc = !p.sortDesc
				return p.resort()
			}
		}
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// diffMarks bundles the two --highlight-diff comparisons a panel can show
// against the other side. Both fields nil means the flag is off, in which
// case View skips the indicator column entirely -- see fileDelegate.
type diffMarks struct {
	uniqueOnly  map[string]bool
	sizeDiffers map[string]bool
}

func (p Panel) View(width, height int, active bool, diff diffMarks) string {
	borderColor := colorMuted
	if active {
		borderColor = colorAccent
	}
	p.list.SetDelegate(fileDelegate{marked: p.marked, uniqueOnly: diff.uniqueOnly, sizeDiffers: diff.sizeDiffers})

	pathStyle := lipgloss.NewStyle().Foreground(colorMuted)
	sortStyle := lipgloss.NewStyle().Foreground(colorMuted)
	counterStyle := lipgloss.NewStyle().Foreground(colorMuted)
	innerWidth := borderInteriorWidth(width)
	if innerWidth < 1 {
		innerWidth = 1
	}

	var pathLine string
	switch {
	case p.jumping:
		prompt := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(":")
		pathLine = prompt + " " + p.jumpInput.View()

	case p.list.FilterState() != list.Unfiltered:
		// Match counter, styled "12/340": visible listed items over the
		// total unfiltered count, shown whenever a filter is typing or
		// applied.
		counter := fmt.Sprintf("%d/%d", len(p.list.VisibleItems()), len(p.files))
		avail := innerWidth - lipgloss.Width(counter) - 1
		if avail < 0 {
			avail = 0
		}
		trimmedPath := runewidth.FillRight(truncateHead(p.path, avail), avail)
		pathLine = pathStyle.Render(trimmedPath) + " " + counterStyle.Render(counter)

	default:
		// The active sort column and direction ride the path line instead of
		// a dedicated header row, which would cost every panel a line of
		// files to show one word and an arrow.
		arrow := "▲"
		if p.sortDesc {
			arrow = "▼"
		}
		indicator := p.sortBy.String() + " " + arrow
		indicatorWidth := runewidth.StringWidth(indicator)

		pathWidth := innerWidth - indicatorWidth - 1 // 1-cell gap before the indicator
		if pathWidth < 1 {
			pathWidth = 1
		}
		path := truncateHead(p.path, pathWidth)

		gap := innerWidth - runewidth.StringWidth(path) - indicatorWidth
		if gap < 1 {
			gap = 1
		}
		pathLine = pathStyle.Render(path) + strings.Repeat(" ", gap) + sortStyle.Render(indicator)
	}

	body := pathLine + "\n" + p.list.View()

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
	for _, f := range p.files {
		if p.marked[f.Name] {
			marked = append(marked, f)
		}
	}
	return marked
}

// uniqueNames returns the names in files that don't appear in other, by
// name only (#52). Directories and files are compared the same way, since
// FileInfo.Name doesn't carry the distinction.
func uniqueNames(files, other []model.FileInfo) map[string]bool {
	otherNames := make(map[string]bool, len(other))
	for _, f := range other {
		otherNames[f.Name] = true
	}

	unique := make(map[string]bool)
	for _, f := range files {
		if !otherNames[f.Name] {
			unique[f.Name] = true
		}
	}
	return unique
}

// sizeDiffers returns the names present in both files and other whose size
// doesn't match. Unlike a timestamp -- which depends on the server's
// timezone and is sometimes only minute-precise -- size is an exact byte
// count from both sides, so this doesn't carry the reliability problem that
// kept #52 to presence-by-name only. Directories are skipped: their
// reported size isn't meaningful to compare.
func sizeDiffers(files, other []model.FileInfo) map[string]bool {
	otherSizes := make(map[string]int64, len(other))
	for _, f := range other {
		if !f.IsDir() {
			otherSizes[f.Name] = f.Size
		}
	}

	differs := make(map[string]bool)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if size, ok := otherSizes[f.Name]; ok && size != f.Size {
			differs[f.Name] = true
		}
	}
	return differs
}

type NavigateMsg struct {
	Panel string
	Path  string
}

type TransferMsg struct {
	SourcePanel string
	Files       []model.FileInfo
}
