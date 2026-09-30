package client

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// The keepalive is what notices a session that died while nobody was asking
// anything of it, and it doubles as the traffic that keeps idle timeouts away.
const (
	keepaliveInterval = 30 * time.Second
	keepaliveTimeout  = 10 * time.Second
)

var errNoConnection = errors.New("no active connection")

// SFTPClient holds one ssh connection and one sftp session, unlike FTP's pool
// that replaces broken connections by itself. When the session dies it opens a
// new one from the same host, user and credentials, so a drop costs a retry
// and not a retyped password. Remote paths are absolute, so the caller's
// current directory survives the swap untouched.
type SFTPClient struct {
	mu      sync.RWMutex
	sshConn *ssh.Client
	client  *sftp.Client
	closed  bool

	// recMu lets one goroutine reconnect while the rest, who saw the same
	// failure, wait and then find the new session already in place.
	recMu sync.Mutex

	host, user, pass string
	port             int
	auth             string
	prompt           HostKeyPrompt
	identity         string
	notify           func(error)
	logger           io.Writer

	// Zero means the constants above; tests shorten them.
	kaEvery, kaWait time.Duration
}

// SetIdentityFile names a private key to try before the agent and the defaults.
func (c *SFTPClient) SetIdentityFile(path string) { c.identity = path }

func (c *SFTPClient) SetHostKeyPrompt(p HostKeyPrompt) { c.prompt = p }

// SetSessionNotifier is told, from another goroutine, every time a lost session
// was reopened (nil) or could not be (the error).
func (c *SFTPClient) SetSessionNotifier(f func(error)) { c.notify = f }

// AuthMethod names how the last successful Connect authenticated.
func (c *SFTPClient) AuthMethod() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.auth
}

// A nil logger disables logging.
func NewSFTPClient(logger io.Writer) *SFTPClient {
	return &SFTPClient{logger: logger}
}

func (c *SFTPClient) logf(format string, args ...any) {
	if c.logger != nil {
		fmt.Fprintf(c.logger, "SFTP "+format+"\n", args...)
	}
}

func (c *SFTPClient) Connect(host, user, pass string, port int) error {
	c.host, c.user, c.pass, c.port = host, user, pass, port

	sshConn, client, auth, err := c.dial()
	if err != nil {
		c.logf("connect failed: %v", err)
		return err
	}
	c.install(sshConn, client, auth)
	return nil
}

func (c *SFTPClient) dial() (*ssh.Client, *sftp.Client, string, error) {
	addr := net.JoinHostPort(c.host, strconv.Itoa(c.port))
	c.logf("> connect %s as %s", addr, c.user)

	// ssh.Dial bounds the TCP dial only. A host that accepts without speaking
	// SSH leaves the handshake waiting with nothing to end it.
	tcpConn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return nil, nil, "", fmt.Errorf("unable to connect to %s: %w", addr, err)
	}
	tcpConn.SetDeadline(time.Now().Add(dialTimeout))

	path, err := knownHostsPath()
	if err != nil {
		tcpConn.Close()
		return nil, nil, "", err
	}
	checkHostKey, err := hostKeyCallback(path, c.prompt, tcpConn)
	if err != nil {
		tcpConn.Close()
		return nil, nil, "", fmt.Errorf("unable to read known_hosts: %w", err)
	}
	hostKeys := func(host string, remote net.Addr, key ssh.PublicKey) error {
		err := checkHostKey(host, remote, key)
		if err == nil {
			c.logf("< host key %s %s accepted", key.Type(), ssh.FingerprintSHA256(key))
		}
		return err
	}

	auth := &sshAuth{pass: c.pass, identity: c.identity}
	defer auth.close()

	config := &ssh.ClientConfig{
		User:            c.user,
		Auth:            auth.methods(),
		HostKeyCallback: hostKeys,
		Timeout:         dialTimeout,
	}

	conn, chans, reqs, err := ssh.NewClientConn(tcpConn, addr, config)
	if err != nil {
		tcpConn.Close()
		return nil, nil, "", fmt.Errorf("unable to connect to %s: %w", addr, auth.explain(err))
	}

	sshConn := ssh.NewClient(conn, chans, reqs)

	// The deadline set above still applies here: a server that accepts the
	// SSH handshake but never answers the SFTP subsystem request would
	// otherwise hang Connect forever.
	client, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return nil, nil, "", fmt.Errorf("error starting SFTP session: %w", err)
	}

	// Left in place the deadline would expire mid-transfer.
	tcpConn.SetDeadline(time.Time{})

	c.logf("< authenticated with %s, sftp subsystem open", auth.method)
	return sshConn, client, auth.method, nil
}

// install makes a fresh session the current one. Callers hold no lock.
func (c *SFTPClient) install(sshConn *ssh.Client, client *sftp.Client, auth string) {
	c.mu.Lock()
	c.sshConn, c.client, c.auth, c.closed = sshConn, client, auth, false
	c.mu.Unlock()
	go c.keepalive(sshConn)
}

func (c *SFTPClient) current() (*ssh.Client, *sftp.Client) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sshConn, c.client
}

func (c *SFTPClient) keepalive(conn *ssh.Client) {
	every, wait := c.kaEvery, c.kaWait
	if every == 0 {
		every = keepaliveInterval
	}
	if wait == 0 {
		wait = keepaliveTimeout
	}
	for {
		time.Sleep(every)
		c.mu.RLock()
		stop := c.closed || c.sshConn != conn
		c.mu.RUnlock()
		if stop {
			return
		}
		if !alive(conn, wait) {
			c.logf("keepalive unanswered")
			c.reconnect(conn)
			return
		}
	}
}

// alive asks the server something, since a connection that went quiet looks
// exactly like an idle one until a request goes unanswered.
func alive(conn *ssh.Client, wait time.Duration) bool {
	done := make(chan error, 1)
	go func() {
		_, _, err := conn.SendRequest("keepalive@openssh.com", true, nil)
		done <- err
	}()
	select {
	case err := <-done:
		return err == nil
	case <-time.After(wait):
		return false
	}
}

// reconnect replaces the session that failed. Several goroutines usually see
// the same failure; only the first dials, the rest find a new session waiting.
func (c *SFTPClient) reconnect(failed *ssh.Client) error {
	c.recMu.Lock()
	defer c.recMu.Unlock()

	c.mu.RLock()
	closed, stale := c.closed, c.sshConn != failed
	c.mu.RUnlock()
	if closed {
		return errNoConnection
	}
	if stale {
		return nil
	}

	c.logf("session lost, reopening")
	sshConn, client, auth, err := c.dial()
	if err != nil {
		c.logf("reopen failed: %v", err)
		c.tell(err)
		return err
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		client.Close()
		sshConn.Close()
		return errNoConnection
	}
	oldClient, oldConn := c.client, c.sshConn
	c.sshConn, c.client, c.auth = sshConn, client, auth
	c.mu.Unlock()

	oldClient.Close()
	oldConn.Close()
	go c.keepalive(sshConn)
	c.logf("session reopened")
	c.tell(nil)
	return nil
}

func (c *SFTPClient) tell(err error) {
	if c.notify != nil {
		c.notify(err)
	}
}

// connectionLost is true for what a dead session produces: the sftp layer's own
// error, a closed or reset socket, or EOF.
func connectionLost(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sftp.ErrSSHFxConnectionLost) || errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	msg := err.Error()
	for _, s := range []string{"connection lost", "broken pipe", "connection reset", "use of closed network connection"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// do runs fn on the current session. If the session turns out to be dead it is
// reopened; an operation that is safe to repeat (a listing, a transfer that
// starts over) then runs again, and one that is not (a rename or a delete,
// which the server may have finished before the drop) reports the loss instead.
func (c *SFTPClient) do(desc string, retry bool, fn func(*sftp.Client) error) (err error) {
	conn, cl := c.current()
	if cl == nil {
		return errNoConnection
	}

	c.logf("> %s", desc)
	defer func() {
		if err != nil {
			c.logf("< %s failed: %v", desc, err)
		} else {
			c.logf("< %s ok", desc)
		}
	}()

	err = fn(cl)
	if !connectionLost(err) {
		return err
	}
	if rerr := c.reconnect(conn); rerr != nil {
		return fmt.Errorf("%w (reopening the session failed: %v)", err, rerr)
	}
	if !retry {
		return fmt.Errorf("the connection was lost and has been reopened, try again: %w", err)
	}
	_, cl = c.current()
	return fn(cl)
}

func (c *SFTPClient) Disconnect() error {
	c.logf("> disconnect")
	c.mu.Lock()
	c.closed = true
	client, conn := c.client, c.sshConn
	c.mu.Unlock()

	var err error
	if client != nil {
		err = client.Close()
	}
	if conn != nil {
		if e := conn.Close(); err == nil {
			err = e
		}
	}
	// A session the server already dropped answers "closed" or EOF, which is
	// not news to someone who asked to disconnect.
	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func (c *SFTPClient) List(path string) ([]model.FileInfo, error) {
	var entries []os.FileInfo
	err := c.do("READDIR "+path, true, func(cl *sftp.Client) (err error) {
		entries, err = cl.ReadDir(path)
		return err
	})
	if errors.Is(err, errNoConnection) {
		return nil, err
	}
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
	return c.do("PUT "+localPath+" to "+remotePath, true, func(cl *sftp.Client) error {
		f, err := os.Open(localPath)
		if err != nil {
			return fmt.Errorf("error opening local file: %w", err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return fmt.Errorf("error reading local file: %w", err)
		}

		dstPath := path.Join(remotePath, filepath.Base(localPath))
		dst, err := cl.Create(dstPath)
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
	})
}

func (c *SFTPClient) Download(remotePath, localPath string, progress func(int64)) error {
	return c.do("GET "+remotePath+" to "+localPath, true, func(cl *sftp.Client) error {
		src, err := cl.Open(remotePath)
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
	})
}

func (c *SFTPClient) Mkdir(path string) error {
	return c.do("MKDIR "+path, true, func(cl *sftp.Client) error { return cl.MkdirAll(path) })
}

func (c *SFTPClient) Rename(oldPath, newPath string) error {
	return c.do("RENAME "+oldPath+" to "+newPath, false, func(cl *sftp.Client) error { return cl.Rename(oldPath, newPath) })
}

func (c *SFTPClient) Delete(path string, isDir bool) error {
	return c.do("REMOVE "+path, false, func(cl *sftp.Client) error {
		if isDir {
			return cl.RemoveAll(path)
		}
		return cl.Remove(path)
	})
}
