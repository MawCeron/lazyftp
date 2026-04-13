package model

import (
	"io/fs"
	"time"
)

type FileType int

const (
	FileTypeFile FileType = iota
	FileTypeDir
	FileTypeSymLink
)

type FileInfo struct {
	Name     string
	Size     int64
	ModTime  time.Time
	Type     FileType
	Mode     fs.FileMode
	IsHidden bool
}

func (f FileInfo) IsDir() bool {
	return f.Type == FileTypeDir
}

func (f FileInfo) IsSymLink() bool {
	return f.Type == FileTypeSymLink
}
