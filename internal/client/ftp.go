package client

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
	"github.com/jlaffaye/ftp"
)

type FTPClient struct {
	conn *ftp.ServerConn
	path string
}

func NewFTPClient() *FTPClient {
	return &FTPClient{
		path: "/",
	}
}

func (c *FTPClient) Connect(host, user, pass string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)

	conn, err := ftp.Dial(addr, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return fmt.Errorf("unable to connect to %s: %w", addr, err)
	}

	if err := conn.Login(user, pass); err != nil {
		return fmt.Errorf("authentication error: %w", err)
	}

	c.conn = conn
	c.path = "/"
	return nil
}

func (c *FTPClient) Disconnect() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Quit()
}

func (c *FTPClient) List(path string) ([]model.FileInfo, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("no active connection")
	}

	entries, err := c.conn.List(path)
	if err != nil {
		return nil, fmt.Errorf("error listing %s: %w", path, err)
	}

	var files []model.FileInfo
	for _, e := range entries {
		if e.Name == "." || e.Name == ".." {
			continue
		}

		fileType := model.FileTypeFile
		switch e.Type {
		case ftp.EntryTypeFolder:
			fileType = model.FileTypeDir
		case ftp.EntryTypeLink:
			fileType = model.FileTypeSymlink
		}

		files = append(files, model.FileInfo{
			Name:     e.Name,
			Size:     int64(e.Size),
			ModTime:  e.Time,
			Type:     fileType,
			IsHidden: len(e.Name) > 0 && e.Name[0] == '.',
		})
	}

	return files, nil
}

func (c *FTPClient) Upload(localPath, remotePath string, progress func(int64)) error {
	if c.conn == nil {
		return fmt.Errorf("no active connection")
	}

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("error opening local file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("error reading local file: %w", err)
	}

	reader := &shared.ProgressReader{
		Reader:   f,
		Total:    info.Size(),
		Callback: progress,
	}

	remotePath = filepath.Join(remotePath, filepath.Base(localPath))
	if err := c.conn.Stor(remotePath, reader); err != nil {
		return fmt.Errorf("error uploading file: %w", err)
	}

	return nil
}

func (c *FTPClient) Download(remotePath, localPath string, progress func(int64)) error {
	if c.conn == nil {
		return fmt.Errorf("no active connection")
	}

	resp, err := c.conn.Retr(remotePath)
	if err != nil {
		return fmt.Errorf("error donwloading file: %w", err)
	}
	defer resp.Close()

	size, err := c.conn.FileSize(remotePath)
	if err != nil {
		size = 0
	}

	destPath := filepath.Join(localPath, filepath.Base(remotePath))
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creating local file: %w", err)
	}
	defer f.Close()

	writer := &shared.ProgressWriter{
		Writer:   f,
		Total:    size,
		Callback: progress,
	}

	if _, err := io.Copy(writer, resp); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}

func (c *FTPClient) Mkdir(path string) error {
	if c.conn == nil {
		return fmt.Errorf("no active connection")
	}
	return c.conn.MakeDir(path)
}

func (c *FTPClient) CurrentPath() string {
	return c.path
}

func (c *FTPClient) ChangePath(path string) error {
	if c.conn == nil {
		return fmt.Errorf("no active connection")
	}
	if err := c.conn.ChangeDir(path); err != nil {
		return fmt.Errorf("error changing directories: %w", err)
	}
	c.path = path
	return nil
}
