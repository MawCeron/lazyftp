package client

import (
	"net"
	"testing"
)

// A PASV reply names the address the data connection is dialled to. A server
// behind NAT names a private one and a hostile one can name any host, so the
// client dials the host it logged in to.
func TestThePASVAddressInTheServersReplyIsNotDialled(t *testing.T) {
	if l, err := net.Listen("tcp", "127.0.0.2:0"); err != nil {
		t.Skipf("cannot listen on 127.0.0.2: %v", err)
	} else {
		l.Close()
	}
	s := startFakeFTP(t)
	s.announce = "127.0.0.2"
	s.listing = "-rw-r--r--   1 u g 5 Jan 01  2026 file.txt\r\n"
	c := connectFakeFTP(t, s)

	files, err := c.List("/")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(files) != 1 || files[0].Name != "file.txt" {
		t.Errorf("listed %v, want file.txt", files)
	}
	if got := s.usedHost(); got != "127.0.0.1" {
		t.Errorf("the data connection went to %s, want the host of the control connection, 127.0.0.1", got)
	}
}
