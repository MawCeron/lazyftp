package client

import (
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPClient struct {
	sshConn *ssh.Client
	client  *sftp.Client
}

func NewSFTPClient() *SFTPClient {
	return &SFTPClient{}
}

func (c *SFTPClient) Connect(host, user, pass string, port int) error {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		// TODO: verificar host key en versiones futuras
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         dialTimeout,
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))

	// ssh.Dial bounds the TCP dial only. A host that accepts without speaking
	// SSH leaves the handshake waiting with nothing to end it.
	tcpConn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return fmt.Errorf("unable to connect to %s: %w", addr, err)
	}
	tcpConn.SetDeadline(time.Now().Add(dialTimeout))

	conn, chans, reqs, err := ssh.NewClientConn(tcpConn, addr, config)
	if err != nil {
		tcpConn.Close()
		return fmt.Errorf("unable to connect to %s: %w", addr, err)
	}

	sshConn := ssh.NewClient(conn, chans, reqs)

	// The deadline set above still applies here: a server that accepts the
	// SSH handshake but never answers the SFTP subsystem request would
	// otherwise hang Connect forever.
	client, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return fmt.Errorf("error starting SFTP session: %w", err)
	}

	// Left in place the deadline would expire mid-transfer.
	tcpConn.SetDeadline(time.Time{})

	c.sshConn = sshConn
	c.client = client
	return nil
}

func (c *SFTPClient) Disconnect() error {
	if c.client != nil {
		c.client.Close()
	}
	if c.sshConn != nil {
		c.sshConn.Close()
	}
	return nil
}

func (c *SFTPClient) List(path string) ([]model.FileInfo, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active connection")
	}

	entries, err := c.client.ReadDir(path)
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

func (c *SFTPClient) Upload(localPath, remotePath string, progress func(int64)) error {
	if c.client == nil {
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

	remotePath = path.Join(remotePath, filepath.Base(localPath))
	dst, err := c.client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("error creating remote file: %w", err)
	}
	defer dst.Close()

	reader := &shared.ProgressReader{
		Reader:   f,
		Total:    info.Size(),
		Callback: progress,
	}

	if _, err := io.Copy(dst, reader); err != nil {
		return fmt.Errorf("error uploading file: %w", err)
	}

	return nil
}

func (c *SFTPClient) Download(remotePath, localPath string, progress func(int64)) error {
	if c.client == nil {
		return fmt.Errorf("no active connection")
	}

	src, err := c.client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("error opening remote file: %w", err)
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("error reading remote file: %w", err)
	}

	destPath := filepath.Join(localPath, filepath.Base(remotePath))
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creating local file: %w", err)
	}
	defer f.Close()

	writer := &shared.ProgressWriter{
		Writer:   f,
		Total:    info.Size(),
		Callback: progress,
	}

	if _, err := io.Copy(writer, src); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}

func (c *SFTPClient) Mkdir(path string) error {
	if c.client == nil {
		return fmt.Errorf("no active connection")
	}
	return c.client.MkdirAll(path)
}
