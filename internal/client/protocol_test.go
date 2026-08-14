package client

import "testing"

func TestProtocolDefaultPort(t *testing.T) {
	for p, want := range map[Protocol]int{FTP: 21, FTPS: 21, SFTP: 22} {
		if got := p.DefaultPort(); got != want {
			t.Errorf("%s default port = %d, want %d", p, got, want)
		}
	}
}

func TestProtocolCyclesBothWaysAndWraps(t *testing.T) {
	if got := SFTP.Next(); got != FTP {
		t.Errorf("SFTP.Next() = %s, want FTP", got)
	}
	if got := FTP.Prev(); got != SFTP {
		t.Errorf("FTP.Prev() = %s, want SFTP", got)
	}

	// Walking the whole cycle returns to the start, in either direction.
	p := FTP
	for range protocolNames {
		p = p.Next()
	}
	if p != FTP {
		t.Errorf("a full cycle of Next ended on %s, want FTP", p)
	}
	for range protocolNames {
		p = p.Prev()
	}
	if p != FTP {
		t.Errorf("a full cycle of Prev ended on %s, want FTP", p)
	}
}

func TestNewReturnsTheRightClient(t *testing.T) {
	if _, ok := New(SFTP, nil).(*SFTPClient); !ok {
		t.Error("New(SFTP) did not return an SFTP client")
	}

	ftp, ok := New(FTP, nil).(*FTPClient)
	if !ok {
		t.Fatal("New(FTP) did not return an FTP client")
	}
	if ftp.tls {
		t.Error("New(FTP) returned a client using TLS")
	}

	ftps, ok := New(FTPS, nil).(*FTPClient)
	if !ok {
		t.Fatal("New(FTPS) did not return an FTP client")
	}
	if !ftps.tls {
		t.Error("New(FTPS) returned a client not using TLS")
	}
}
