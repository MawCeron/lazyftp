// sftpd is a throwaway SFTP server for recording assets/demo.tape -- not
// part of the lazyftp product. It serves one directory (the first argument)
// as the SFTP root, over password auth on 127.0.0.1:2022, using only
// dependencies lazyftp's own SFTP client already brings in
// (golang.org/x/crypto/ssh, github.com/pkg/sftp).
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	listenAddr = "127.0.0.1:2022"
	demoUser   = "demo"
	demoPass   = "demo"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: sftpd <root-dir>")
		os.Exit(1)
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "sftpd:", err)
		os.Exit(1)
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sftpd:", err)
		os.Exit(1)
	}
	signer, err := ssh.NewSignerFromSigner(priv)
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
	config.AddHostKey(signer)

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sftpd:", err)
		os.Exit(1)
	}
	defer ln.Close()
	fmt.Println("sftpd: serving", root, "on", listenAddr)

	for {
		nConn, err := ln.Accept()
		if err != nil {
			return
		}
		go serveConn(nConn, config, root)
	}
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
