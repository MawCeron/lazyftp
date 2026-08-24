package ui

import (
	"cmp"
	"sort"
	"strings"

	"github.com/MawCeron/lazyftp/internal/model"
)

type sortColumn int

const (
	sortByName sortColumn = iota
	sortBySize
	sortByDate
)

func (s sortColumn) String() string {
	switch s {
	case sortBySize:
		return "Size"
	case sortByDate:
		return "Date"
	default:
		return "Name"
	}
}

// next cycles Name -> Size -> Date -> Name.
func (s sortColumn) next() sortColumn {
	return (s + 1) % 3
}

// sortFiles sorts files by column and direction, in place. Directories
// always group first regardless of column or direction -- mixing files and
// directories by size or date is rarely what anyone wants. Ties within a
// column break by name, ascending.
func sortFiles(files []model.FileInfo, column sortColumn, desc bool) {
	sort.SliceStable(files, func(i, j int) bool {
		a, b := files[i], files[j]
		if a.IsDir() != b.IsDir() {
			return a.IsDir()
		}

		c := compareColumn(column, a, b)
		if c == 0 {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		if desc {
			c = -c
		}
		return c < 0
	})
}

func compareColumn(column sortColumn, a, b model.FileInfo) int {
	switch column {
	case sortBySize:
		return cmp.Compare(a.Size, b.Size)
	case sortByDate:
		return a.ModTime.Compare(b.ModTime)
	default:
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	}
}
