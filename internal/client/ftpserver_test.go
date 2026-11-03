package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeFTP is a minimal FTP server that records every command line it receives,
// so a test can say which commands did and did not reach the wire. It supports
// just enough of the protocol for goftp to log in and open a data connection.
type fakeFTP struct {
	ln net.Listener

	// listing is what a LIST sends over the data connection.
	listing string

	mu    sync.Mutex
	lines []string
}

func startFakeFTP(t *testing.T) *fakeFTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeFTP{ln: ln}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serve(c)
		}
	}()
	return s
}

func (s *fakeFTP) port() int { return s.ln.Addr().(*net.TCPAddr).Port }

func (s *fakeFTP) record(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lines = append(s.lines, line)
}

// sent returns the command lines received so far.
func (s *fakeFTP) sent() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.lines...)
}

// tainted reports whether any received line carries a byte that FTP uses to end
// or cut a command: a line break inside a name would have been split into two
// commands, and a NUL truncates one on some servers.
func (s *fakeFTP) tainted() bool {
	for _, l := range s.sent() {
		if strings.ContainsAny(l, "\r\x00") {
			return true
		}
	}
	return false
}

// reached reports whether any received line starts with the command verb.
func (s *fakeFTP) reached(verb string) bool {
	for _, l := range s.sent() {
		if strings.HasPrefix(strings.ToUpper(l), verb+" ") || strings.EqualFold(l, verb) {
			return true
		}
	}
	return false
}

func (s *fakeFTP) serve(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	reply := func(format string, args ...any) { fmt.Fprintf(c, format+"\r\n", args...) }
	reply("220 ready")

	var data net.Listener
	defer func() {
		if data != nil {
			data.Close()
		}
	}()
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		s.record(line)
		verb, _, _ := strings.Cut(line, " ")
		switch strings.ToUpper(verb) {
		case "USER":
			reply("331 password please")
		case "PASS":
			reply("230 logged in")
		case "PWD":
			reply(`257 "/" is the current directory`)
		case "TYPE", "OPTS", "NOOP":
			reply("200 ok")
		case "MKD":
			reply(`257 "created"`)
		case "DELE", "RMD", "RNTO":
			reply("250 done")
		case "RNFR":
			reply("350 ready for RNTO")
		case "EPSV":
			if data != nil {
				data.Close()
			}
			data, err = net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				reply("425 no data connection")
				continue
			}
			reply("229 Entering Extended Passive Mode (|||%s|)", strconv.Itoa(data.Addr().(*net.TCPAddr).Port))
		case "STOR", "RETR", "LIST":
			if data == nil {
				reply("425 use EPSV first")
				continue
			}
			reply("150 opening data connection")
			dc, err := data.Accept()
			if err != nil {
				reply("425 no data connection")
				continue
			}
			switch strings.ToUpper(verb) {
			case "STOR":
				io.Copy(io.Discard, dc)
			case "LIST":
				io.WriteString(dc, s.listing)
			}
			dc.Close()
			reply("226 transfer complete")
		case "QUIT":
			reply("221 bye")
			return
		default:
			reply("502 not implemented")
		}
	}
}
