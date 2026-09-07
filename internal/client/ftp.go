package client

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
	goftp "github.com/secsy/goftp"
)

type FTPClient struct {
	conn   *goftp.Client
	host   string
	logger io.Writer
	tls    bool
}

// A nil logger disables logging.
func NewFTPClient(logger io.Writer) *FTPClient {
	return &FTPClient{logger: logger}
}

// Explicit TLS only; implicit FTPS is not offered.
func NewFTPSClient(logger io.Writer) *FTPClient {
	return &FTPClient{logger: logger, tls: true}
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
		// Certificates are verified; a self-signed one fails here.
		config.TLSConfig = &tls.Config{ServerName: host}
	}

	conn, err := goftp.DialConfig(config, c.host)
	if err != nil {
		return fmt.Errorf("unable to connect to %s: %w", c.host, err)
	}

	// DialConfig only builds a pool; nothing has reached the server yet.
	if _, err := conn.Getwd(); err != nil {
		conn.Close()
		if c.tls {
			// A server without TLS refuses AUTH TLS with the same 530 it gives
			// a bad login, so the two cannot be told apart here.
			return fmt.Errorf("unable to connect to %s over FTPS; the server may not offer TLS, or the credentials may be wrong: %w", c.host, err)
		}
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

	entries, err := c.readDir(path)
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

	remotePath = path.Join(remotePath, filepath.Base(localPath))
	if err := c.conn.Store(remotePath, reader); err != nil {
		return fmt.Errorf("error uploading file: %w", err)
	}

	return nil
}

func (c *FTPClient) Download(remotePath, localPath string, progress func(int64)) error {
	if c.conn == nil {
		return fmt.Errorf("no active connection")
	}

	entries, err := c.readDir(path.Dir(remotePath))
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

// readDir lists a directory, falling back to a DOS/IIS-style LIST parser
// when goftp's own Unix-only parser can't read the server's output (#86).
func (c *FTPClient) readDir(path string) ([]os.FileInfo, error) {
	entries, err := c.conn.ReadDir(path)
	if err != nil && strings.Contains(err.Error(), "failed parsing LIST entry:") {
		return c.readDirDOS(path)
	}
	return entries, err
}

// readDirDOS re-issues LIST over its own raw connection (goftp's ReadDir
// gives no way to swap in a different parser) and reads entries in the
// MS-DOS format Windows/IIS FTP servers use, e.g.:
//
//	07-20-26  09:52AM       <DIR>          Fuentes
//	07-20-26  10:15AM             123456 report.pdf
func (c *FTPClient) readDirDOS(path string) ([]os.FileInfo, error) {
	raw, err := c.conn.OpenRawConn()
	if err != nil {
		return nil, err
	}
	defer raw.Close()

	getConn, err := raw.PrepareDataConn()
	if err != nil {
		return nil, err
	}

	code, msg, err := raw.SendCommand("LIST %s", path)
	if err != nil {
		return nil, err
	}
	if code/100 != 1 {
		return nil, fmt.Errorf("unexpected response to LIST %s: %d-%s", path, code, msg)
	}

	dc, err := getConn()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(dc)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	scanErr := scanner.Err()
	dc.Close()
	if scanErr != nil {
		return nil, scanErr
	}

	if code, msg, err := raw.ReadResponse(); err != nil {
		return nil, err
	} else if code/100 != 2 {
		return nil, fmt.Errorf("unexpected response after LIST %s: %d-%s", path, code, msg)
	}

	var entries []os.FileInfo
	for _, line := range lines {
		if info, ok := parseDOSListEntry(line); ok {
			entries = append(entries, info)
		}
	}
	return entries, nil
}

var dosListRegex = regexp.MustCompile(`^(\d{2}-\d{2}-\d{2})\s+(\d{2}:\d{2}(?:AM|PM))\s+(<DIR>|\d+)\s+(.+)$`)

func parseDOSListEntry(line string) (os.FileInfo, bool) {
	m := dosListRegex.FindStringSubmatch(line)
	if m == nil {
		return nil, false
	}

	name := m[4]
	if name == "." || name == ".." {
		return nil, false
	}

	mtime, err := time.Parse("01-02-06 03:04PM", m[1]+" "+m[2])
	if err != nil {
		return nil, false
	}

	isDir := m[3] == "<DIR>"
	var size int64
	if !isDir {
		size, err = strconv.ParseInt(m[3], 10, 64)
		if err != nil {
			return nil, false
		}
	}

	return &dosFileInfo{name: name, size: size, isDir: isDir, mtime: mtime}, true
}

type dosFileInfo struct {
	name  string
	size  int64
	isDir bool
	mtime time.Time
}

func (f *dosFileInfo) Name() string       { return f.name }
func (f *dosFileInfo) Size() int64        { return f.size }
func (f *dosFileInfo) ModTime() time.Time { return f.mtime }
func (f *dosFileInfo) IsDir() bool        { return f.isDir }
func (f *dosFileInfo) Sys() any           { return nil }

func (f *dosFileInfo) Mode() os.FileMode {
	if f.isDir {
		return os.ModeDir | 0o755
	}
	return 0o644
}
