package ui

import (
	"os"

	"github.com/MawCeron/lazyftp/internal/model"
)

func listLocalDir(path string) ([]model.FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []model.FileInfo
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}

		fileType := model.FileTypeFile
		if e.IsDir() {
			fileType = model.FileTypeDir
		} else if info.Mode()&os.ModeSymlink != 0 {
			fileType = model.FileTypeSymlink
		}

		files = append(files, model.FileInfo{
			Name:     e.Name(),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			Type:     fileType,
			Mode:     info.Mode(),
			IsHidden: len(e.Name()) > 0 && e.Name()[0] == '.',
		})
	}

	return files, nil
}
