package client

import (
	"crypto/tls"
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
	path   string
	logger io.Writer
	tls    bool
}

// NewFTPClient returns a client that records its control dialogue to logger.
// A nil logger disables logging.
func NewFTPClient(logger io.Writer) *FTPClient {
	return &FTPClient{path: "/", logger: logger}
}

// NewFTPSClient is NewFTPClient over TLS, negotiated with AUTH TLS once the
// control connection is open. Implicit FTPS, which expects TLS from the first
// byte, is not offered.
func NewFTPSClient(logger io.Writer) *FTPClient {
	return &FTPClient{path: "/", logger: logger, tls: true}
}

func (c *FTPClient) Connect(host, user, pass string, port int) error {
	c.host = fmt.Sprintf("%s:%d", host, port)

	config := goftp.Config{
		User:     user,
		Password: pass,
		Timeout:  dialTimeout,
		Logger:   c.logger,
	}

	if c.tls {
		// Certificates are verified. A server with a self-signed certificate
		// fails here, which is the honest outcome rather than a silent bypass.
		config.TLSConfig = &tls.Config{ServerName: host}
	}

	conn, err := goftp.DialConfig(config, c.host)
	if err != nil {
		return fmt.Errorf("unable to connect to %s: %w", c.host, err)
	}

	// DialConfig only builds a connection pool; nothing has reached the server
	// yet. One round trip is what turns an unreachable host, a refused login or
	// a server without TLS into an error here, instead of a connection that
	// reports success and then shows an empty panel.
	if _, err := conn.Getwd(); err != nil {
		conn.Close()
		if c.tls {
			// A server without TLS refuses AUTH TLS with a 530, the same reply
			// it gives to a bad login, so which of the two happened cannot be
			// told apart here. Naming both beats reading "please login with
			// USER and PASS" and going to check a password that was fine.
			return fmt.Errorf("unable to connect to %s over FTPS; the server may not offer TLS, or the credentials may be wrong: %w", c.host, err)
		}
		return fmt.Errorf("unable to connect to %s: %w", c.host, err)
	}

	c.conn = conn
	c.path = "/"
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
			Name:     e.Name(),
			Size:     e.Size(),
			ModTime:  e.ModTime(),
			Type:     fileType,
			IsHidden: len(e.Name()) > 0 && e.Name()[0] == '.',
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

func (c *FTPClient) CurrentPath() string {
	return c.path
}

func (c *FTPClient) ChangePath(path string) error {
	if c.conn == nil {
		return fmt.Errorf("no active connection")
	}
	// Verificar que existe y es directorio
	info, err := c.conn.Stat(path)
	if err != nil {
		return fmt.Errorf("dir not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	c.path = path
	return nil
}
