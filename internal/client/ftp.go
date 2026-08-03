package client

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
	goftp "github.com/secsy/goftp"
)

type FTPClient struct {
	conn   *goftp.Client
	host   string
	logger io.Writer
}

// NewFTPClient returns a client that records its control dialogue to logger.
// A nil logger disables logging.
func NewFTPClient(logger io.Writer) *FTPClient {
	return &FTPClient{logger: logger}
}

func (c *FTPClient) Connect(host, user, pass string, port int) error {
	c.host = fmt.Sprintf("%s:%d", host, port)

	config := goftp.Config{
		User:     user,
		Password: pass,
		Timeout:  dialTimeout,
		Logger:   c.logger,
	}

	conn, err := goftp.DialConfig(config, c.host)
	if err != nil {
		return fmt.Errorf("unable to connect to %s: %w", c.host, err)
	}

	c.conn = conn
	return nil
}

func (c *FTPClient) Disconnect() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *FTPClient) List(path string) ([]model.FileInfo, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("no active connection")
	}

	entries, err := c.conn.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("error listing %s: %w", path, err)
	}

	var files []model.FileInfo
	for _, e := range entries {
		if e.Name() == "." || e.Name() == ".." {
			continue
		}

		fileType := model.FileTypeFile
		if e.IsDir() {
			fileType = model.FileTypeDir
		} else if e.Mode()&os.ModeSymlink != 0 {
			fileType = model.FileTypeSymlink
		}

		files = append(files, model.FileInfo{
			Name:    e.Name(),
			Size:    e.Size(),
			ModTime: e.ModTime(),
			Type:    fileType,
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
	if err := c.conn.Store(remotePath, reader); err != nil {
		return fmt.Errorf("error uploading file: %w", err)
	}

	return nil
}

func (c *FTPClient) Download(remotePath, localPath string, progress func(int64)) error {
	if c.conn == nil {
		return fmt.Errorf("no active connection")
	}

	// Obtener tamaño antes de descargar para el progress
	entries, err := c.conn.ReadDir(filepath.Dir(remotePath))
	size := int64(0)
	if err == nil {
		for _, e := range entries {
			if e.Name() == filepath.Base(remotePath) {
				size = e.Size()
				break
			}
		}
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

	if err := c.conn.Retrieve(remotePath, writer); err != nil {
		return fmt.Errorf("error downloading file: %w", err)
	}

	return nil
}

func (c *FTPClient) Mkdir(path string) error {
	if c.conn == nil {
		return fmt.Errorf("no active connection")
	}
	_, err := c.conn.Mkdir(path)
	return err
}
