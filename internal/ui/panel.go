package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/MawCeron/lazyftp/internal/model"
)

type Panel struct {
	title  string
	path   string
	files  []model.FileInfo
	cursor int
	marked map[int]bool
	offset int
}

func NewPanel(title string) Panel {
	return Panel{
		title:  title,
		path:   "/",
		files:  []model.FileInfo{},
		marked: make(map[int]bool),
	}
}

func (p Panel) WithFiles(files []model.FileInfo, path string) Panel {
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir() != files[j].IsDir() {
			return files[i].IsDir()
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	p.files = files
	p.path = path
	p.cursor = 0
	p.offset = 0
	p.marked = make(map[int]bool)
	return p
}

func (p Panel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "j", "down":
			if p.cursor < len(p.files)-1 {
				p.cursor++
			}

		case "k", "up":
			if p.cursor > 0 {
				p.cursor--
			}

		case "enter", " ":
			if len(p.files) > 0 && p.files[p.cursor].IsDir() {
				name := p.files[p.cursor].Name
				newPath := p.path + "/" + name
				if p.path == "/" {
					newPath = "/" + name
				}
				panel := p.title
				return p, func() tea.Msg {
					return NavigateMsg{Panel: panel, Path: newPath}
				}
			}

		case "-", "backspace":
			panel := p.title
			parent := parentPath(p.path)
			return p, func() tea.Msg {
				return NavigateMsg{Panel: panel, Path: parent}
			}

		case "x":
			p.marked[p.cursor] = !p.marked[p.cursor]
			if !p.marked[p.cursor] {
				delete(p.marked, p.cursor)
			}

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

	return p, nil
}

func (p Panel) View(width, height int, active bool) string {
	borderColor := lipgloss.Color("240")
	if active {
		borderColor = lipgloss.Color("39")
	}

	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	markedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("75"))

	innerWidth := width - 4
	path := p.path
	if len([]rune(path)) > innerWidth {
		path = "…" + string([]rune(path)[len([]rune(path))-innerWidth+1:])
	}

	header := pathStyle.Render(path) + "\n" +
		strings.Repeat("─", innerWidth)

	visibleHeight := height - 5
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	p.adjustOffset(visibleHeight)

	var rows []string
	for i := p.offset; i < len(p.files) && i < p.offset+visibleHeight; i++ {
		f := p.files[i]
		isMarked := p.marked[i]

		var line string
		if isMarked {
			prefix := markedStyle.Render("✓ ")
			if i == p.cursor {
				prefix = cursorStyle.Render("> ")
			}
			name := markedStyle.Render(f.Name)
			if f.IsDir() {
				name = markedStyle.Render(f.Name + "/")
			}
			line = prefix + name
			if active {
				line = lipgloss.NewStyle().
					Background(lipgloss.Color("52")).
					Width(innerWidth).
					Render(line)
			}
		} else {
			prefix := "  "
			if i == p.cursor {
				prefix = cursorStyle.Render("> ")
			}
			name := f.Name
			if f.IsDir() {
				name = dirStyle.Render(f.Name + "/")
			}
			line = prefix + name
			if i == p.cursor && active {
				line = lipgloss.NewStyle().
					Background(lipgloss.Color("236")).
					Width(innerWidth).
					Render(line)
			}
		}

		rows = append(rows, line)
	}

	if len(p.files) == 0 {
		rows = append(rows, pathStyle.Render("  (vacío)"))
	}

	body := header + "\n" + strings.Join(rows, "\n")
	return borderWithTitle(body, p.title, width, height, borderColor)
}

func (p *Panel) adjustOffset(visible int) {
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+visible {
		p.offset = p.cursor - visible + 1
	}
}

func (p Panel) selectedFiles() []model.FileInfo {
	if len(p.marked) == 0 {
		if len(p.files) > 0 {
			return []model.FileInfo{p.files[p.cursor]}
		}
		return nil
	}
	var selected []model.FileInfo
	for i := range p.marked {
		selected = append(selected, p.files[i])
	}
	return selected
}

func parentPath(path string) string {
	if path == "/" {
		return "/"
	}
	idx := strings.LastIndex(path, "/")
	if idx == 0 {
		return "/"
	}
	return path[:idx]
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
