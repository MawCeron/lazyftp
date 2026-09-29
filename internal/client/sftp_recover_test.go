package client

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// testServer is an SFTP server on a loopback port whose connections can be cut
// on demand, the way an idle timeout or a firewall cuts them.
type testServer struct {
	ln    net.Listener
	fs    sftp.Handlers // one filesystem for every session, as a real server has
	mu    sync.Mutex
	conns []net.Conn
	seen  int
}

func startServer(t *testing.T) *testServer {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromSigner(priv)
	config := &ssh.ServerConfig{
		PasswordCallback: func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) { return nil, nil },
	}
	config.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &testServer{ln: ln, fs: sftp.InMemHandler()}
	t.Cleanup(func() { ln.Close(); s.dropAll() })

	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				return
			}
			s.mu.Lock()
			s.conns = append(s.conns, nc)
			s.seen++
			s.mu.Unlock()
			go serve(nc, config, s.fs)
		}
	}()
	return s
}

func serve(nc net.Conn, config *ssh.ServerConfig, fs sftp.Handlers) {
	defer nc.Close()
	_, chans, reqs, err := ssh.NewServerConn(nc, config)
	if err != nil {
		return
	}
	go func() {
		for r := range reqs {
			r.Reply(false, nil)
		}
	}()
	for ch := range chans {
		channel, requests, err := ch.Accept()
		if err != nil {
			continue
		}
		go func() {
			for r := range requests {
				ok := r.Type == "subsystem"
				r.Reply(ok, nil)
				if ok {
					sftp.NewRequestServer(channel, fs).Serve()
					channel.Close()
					return
				}
			}
		}()
	}
}

func (s *testServer) dropAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.conns {
		c.Close()
	}
	s.conns = nil
}

func (s *testServer) sessions() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seen
}

// connect returns a client on the server plus a channel of what its notifier
// reports; HOME points at a temp dir so known_hosts is the test's own.
func (s *testServer) connect(t *testing.T, tune func(*SFTPClient)) (*SFTPClient, <-chan error) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SSH_AUTH_SOCK", "")

	events := make(chan error, 8)
	c := NewSFTPClient()
	c.SetHostKeyPrompt(func(string, string, string) bool { return true })
	c.SetSessionNotifier(func(err error) { events <- err })
	if tune != nil {
		tune(c)
	}
	host, port, _ := net.SplitHostPort(s.ln.Addr().String())
	p, _ := strconv.Atoi(port)
	if err := c.Connect(host, "u", "p", p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Disconnect() })
	return c, events
}

func wait(t *testing.T, events <-chan error) error {
	t.Helper()
	select {
	case err := <-events:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("no session event")
		return nil
	}
}

func TestAListingSurvivesADroppedSession(t *testing.T) {
	s := startServer(t)
	c, events := s.connect(t, nil)

	if err := c.Mkdir("/work"); err != nil {
		t.Fatal(err)
	}
	s.dropAll()

	files, err := c.List("/")
	if err != nil || len(files) != 1 || files[0].Name != "work" {
		t.Fatalf("list after the drop: %v, %v", files, err)
	}
	if err := wait(t, events); err != nil {
		t.Fatalf("reopened, but the notifier said %v", err)
	}
	if s.sessions() != 2 {
		t.Errorf("server saw %d sessions, want the original and one replacement", s.sessions())
	}
}

func TestAnOperationThatIsNotSafeToRepeatReportsTheLossButReopens(t *testing.T) {
	s := startServer(t)
	c, _ := s.connect(t, nil)
	c.Mkdir("/a")
	s.dropAll()

	if err := c.Rename("/a", "/b"); err == nil {
		t.Fatal("a rename over a dead session cannot have worked")
	}
	if err := c.Rename("/a", "/b"); err != nil {
		t.Fatalf("the session should be back for the next attempt: %v", err)
	}
}

func TestKeepaliveNoticesADeadSessionWithoutAnyOperation(t *testing.T) {
	s := startServer(t)
	_, events := s.connect(t, func(c *SFTPClient) { c.kaEvery, c.kaWait = 30*time.Millisecond, time.Second })

	s.dropAll()
	if err := wait(t, events); err != nil {
		t.Fatalf("keepalive found the drop but reopening failed: %v", err)
	}
}

func TestAServerThatIsGoneIsReportedNotRetriedForever(t *testing.T) {
	s := startServer(t)
	c, events := s.connect(t, nil)

	s.ln.Close()
	s.dropAll()
	if _, err := c.List("/"); err == nil {
		t.Fatal("listed on a server that is gone")
	}
	if err := wait(t, events); err == nil {
		t.Fatal("the failure to reopen was not reported")
	}
}

func TestNothingIsReopenedAfterDisconnect(t *testing.T) {
	s := startServer(t)
	c, _ := s.connect(t, func(c *SFTPClient) { c.kaEvery, c.kaWait = 20*time.Millisecond, time.Second })
	c.Disconnect()
	s.dropAll()

	time.Sleep(200 * time.Millisecond)
	if s.sessions() != 1 {
		t.Fatalf("a client that was disconnected dialled again: %d sessions", s.sessions())
	}
	if _, err := c.List("/"); err == nil {
		t.Fatal("listed after Disconnect")
	}
}

func TestATransferRestartsOnTheReopenedSession(t *testing.T) {
	s := startServer(t)
	c, _ := s.connect(t, nil)

	dir := t.TempDir()
	src := filepath.Join(dir, "data.bin")
	os.WriteFile(src, []byte("payload"), 0o644)

	s.dropAll()
	var last int64
	if err := c.Upload(src, "/", func(n int64) { last = n }); err != nil {
		t.Fatalf("upload over a dropped session: %v", err)
	}
	if last != 7 {
		t.Errorf("progress ended at %d, want 7", last)
	}

	out := t.TempDir()
	s.dropAll()
	if err := c.Download("/data.bin", out, func(int64) {}); err != nil {
		t.Fatalf("download over a dropped session: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(out, "data.bin")); string(got) != "payload" {
		t.Errorf("downloaded %q", got)
	}
}
