package model

import (
	"time"
)

type FileType int

const (
	FileTypeFile FileType = iota
	FileTypeDir
	FileTypeSymlink
)

type FileInfo struct {
	Name     string
	Size     int64
	ModTime  time.Time
	Type     FileType
	IsHidden bool
}

func (f FileInfo) IsDir() bool {
	return f.Type == FileTypeDir
}

func (f FileInfo) IsSymlink() bool {
	return f.Type == FileTypeSymlink
}
