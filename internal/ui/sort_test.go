package ui

import (
	"testing"
	"time"

	"github.com/MawCeron/lazyftp/internal/model"
)

func names(files []model.FileInfo) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Name
	}
	return out
}

func equalNames(got []model.FileInfo, want []string) bool {
	g := names(got)
	if len(g) != len(want) {
		return false
	}
	for i := range g {
		if g[i] != want[i] {
			return false
		}
	}
	return true
}

func TestSortFilesDirectoriesAlwaysFirst(t *testing.T) {
	files := []model.FileInfo{
		{Name: "small.txt", Size: 1},
		{Name: "zdir", Type: model.FileTypeDir, Size: 999},
		{Name: "big.txt", Size: 1000},
	}

	// Sorting by size, descending, would put zdir last on size alone --
	// directories must still come first regardless.
	sortFiles(files, sortBySize, true)
	if !equalNames(files, []string{"zdir", "big.txt", "small.txt"}) {
		t.Errorf("got %v, want zdir first", names(files))
	}
}

func TestSortFilesByColumn(t *testing.T) {
	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	base := func() []model.FileInfo {
		return []model.FileInfo{
			{Name: "b.txt", Size: 200, ModTime: newer},
			{Name: "a.txt", Size: 300, ModTime: older},
			{Name: "c.txt", Size: 100, ModTime: older.Add(3 * time.Hour)},
		}
	}

	cases := []struct {
		name   string
		column sortColumn
		desc   bool
		want   []string
	}{
		{"name asc", sortByName, false, []string{"a.txt", "b.txt", "c.txt"}},
		{"name desc", sortByName, true, []string{"c.txt", "b.txt", "a.txt"}},
		{"size asc", sortBySize, false, []string{"c.txt", "b.txt", "a.txt"}},
		{"size desc", sortBySize, true, []string{"a.txt", "b.txt", "c.txt"}},
		{"date asc", sortByDate, false, []string{"a.txt", "c.txt", "b.txt"}},
		{"date desc", sortByDate, true, []string{"b.txt", "c.txt", "a.txt"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := base()
			sortFiles(files, c.column, c.desc)
			if !equalNames(files, c.want) {
				t.Errorf("sortFiles(%v, desc=%v) = %v, want %v", c.column, c.desc, names(files), c.want)
			}
		})
	}
}

func TestSortFilesTiesBreakByName(t *testing.T) {
	files := []model.FileInfo{
		{Name: "z.txt", Size: 100},
		{Name: "a.txt", Size: 100},
	}
	sortFiles(files, sortBySize, false)
	if !equalNames(files, []string{"a.txt", "z.txt"}) {
		t.Errorf("equal sizes: got %v, want tie broken by name", names(files))
	}
}

func TestSortColumnNextCycles(t *testing.T) {
	if sortByName.next() != sortBySize {
		t.Error("Name should cycle to Size")
	}
	if sortBySize.next() != sortByDate {
		t.Error("Size should cycle to Date")
	}
	if sortByDate.next() != sortByName {
		t.Error("Date should cycle back to Name")
	}
}
