// sftpd is a throwaway SFTP server for recording the GIFs under assets/ -- not
// part of the lazyftp product. It serves one directory (the last argument)
// as the SFTP root, on 127.0.0.1:2022, over password auth, using only
// dependencies lazyftp's own SFTP client already brings in
// (golang.org/x/crypto/ssh, github.com/pkg/sftp).
//
//	sftpd [-hostkey file] [-authorized file] [-drop-first duration] <root-dir>
//
// -hostkey keeps the server's host key in a file, created on first use, so a
// restarted server is still the same server to a client that already trusts it.
// Without it a new key is made on every start. -authorized is an
// authorized_keys file whose keys may log in as the demo user, besides the
// password. -drop-first cuts the first connection after that long, the way an
// idle timeout would, so a recording can show a dropped session being reopened.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	listenAddr = "127.0.0.1:2022"
	demoUser   = "demo"
	demoPass   = "demo"
)

func main() {
	hostKeyFile := flag.String("hostkey", "", "keep the host key in this `file`, creating it if missing")
	dropFirst := flag.Duration("drop-first", 0, "cut the first connection after this `duration`, like an idle timeout")
	authorizedFile := flag.String("authorized", "", "authorized_keys `file`: its keys may log in as the demo user")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: sftpd [-hostkey file] [-authorized file] [-drop-first duration] <root-dir>")
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	root, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "sftpd:", err)
		os.Exit(1)
	}

	signer, err := loadOrCreateHostKey(*hostKeyFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sftpd:", err)
		os.Exit(1)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == demoUser && string(pass) == demoPass {
				return nil, nil
			}
			return nil, fmt.Errorf("denied")
		},
	}
	if *authorizedFile != "" {
		keys, err := loadAuthorizedKeys(*authorizedFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "sftpd:", err)
			os.Exit(1)
		}
		config.PublicKeyCallback = func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if c.User() == demoUser && keys[string(key.Marshal())] {
				return nil, nil
			}
			return nil, fmt.Errorf("denied")
		}
	}
	config.AddHostKey(signer)

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sftpd:", err)
		os.Exit(1)
	}
	defer ln.Close()
	fmt.Println("sftpd: serving", root, "on", listenAddr)

	for first := true; ; first = false {
		nConn, err := ln.Accept()
		if err != nil {
			return
		}
		if first && *dropFirst > 0 {
			dropAfter(nConn, *dropFirst)
		}
		go serveConn(nConn, config, root)
	}
}

// loadOrCreateHostKey returns the key stored in path, writing a new one there
// first if the file is missing. An empty path is a key that lives and dies with
// the process.
func loadOrCreateHostKey(path string) (ssh.Signer, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			return ssh.ParsePrivateKey(data)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	if path != "" {
		block, err := ssh.MarshalPrivateKey(priv, "")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
			return nil, err
		}
	}
	return ssh.NewSignerFromSigner(priv)
}

// loadAuthorizedKeys reads an authorized_keys file into a set keyed by the
// wire form of each public key.
func loadAuthorizedKeys(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	keys := map[string]bool{}
	for len(data) > 0 {
		key, _, _, rest, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			break
		}
		keys[string(key.Marshal())] = true
		data = rest
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("%s holds no public keys", path)
	}
	return keys, nil
}

func dropAfter(c net.Conn, d time.Duration) *time.Timer {
	return time.AfterFunc(d, func() { c.Close() })
}

func serveConn(nConn net.Conn, config *ssh.ServerConfig, root string) {
	defer nConn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "only sessions are supported")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go serveChannel(channel, requests, root)
	}
}

// serveChannel waits for the "sftp" subsystem request the client sends right
// after opening the session, then hands the channel to an SFTP request
// server for the rest of its life -- the same subsystem handshake a real
// sshd performs for `sftp-server`.
func serveChannel(channel ssh.Channel, requests <-chan *ssh.Request, root string) {
	defer channel.Close()

	for req := range requests {
		isSFTP := req.Type == "subsystem" && len(req.Payload) >= 4 && string(req.Payload[4:]) == "sftp"
		req.Reply(isSFTP, nil)
		if !isSFTP {
			continue
		}

		server := sftp.NewRequestServer(channel, sftp.Handlers{
			FileGet:  rootedFS{root},
			FilePut:  rootedFS{root},
			FileCmd:  rootedFS{root},
			FileList: rootedFS{root},
		})
		server.Serve()
		server.Close()
		return
	}
}

// rootedFS answers SFTP requests against root, translating every absolute
// SFTP path (the client always deals in "/", never root's real filesystem
// path) onto a real path under it -- a chroot done in Go, since the chroot
// syscall itself needs privileges this demo has no reason to ask for.
type rootedFS struct{ root string }

func (fs rootedFS) real(p string) string {
	return filepath.Join(fs.root, filepath.FromSlash(path.Clean("/"+p)))
}

func (fs rootedFS) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	return os.Open(fs.real(r.Filepath))
}

func (fs rootedFS) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	return os.OpenFile(fs.real(r.Filepath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
}

func (fs rootedFS) Filecmd(r *sftp.Request) error {
	switch r.Method {
	case "Mkdir":
		return os.MkdirAll(fs.real(r.Filepath), 0o755)
	case "Remove":
		return os.Remove(fs.real(r.Filepath))
	case "Rmdir":
		return os.RemoveAll(fs.real(r.Filepath))
	case "Rename":
		return os.Rename(fs.real(r.Filepath), fs.real(r.Target))
	default:
		return nil
	}
}

type fileInfoList []os.FileInfo

func (l fileInfoList) ListAt(dst []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n := copy(dst, l[offset:])
	if n < len(dst) {
		return n, io.EOF
	}
	return n, nil
}

func (fs rootedFS) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	real := fs.real(r.Filepath)
	switch r.Method {
	case "List":
		entries, err := os.ReadDir(real)
		if err != nil {
			return nil, err
		}
		infos := make(fileInfoList, 0, len(entries))
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			infos = append(infos, info)
		}
		return infos, nil
	case "Stat":
		info, err := os.Stat(real)
		if err != nil {
			return nil, err
		}
		return fileInfoList{info}, nil
	default:
		return nil, fmt.Errorf("unsupported list method %q", r.Method)
	}
}
