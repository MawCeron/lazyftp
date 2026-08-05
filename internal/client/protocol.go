package client

import "io"

// Protocol is chosen by the user rather than inferred from the port, so that a
// server on a non-standard port is still reachable.
type Protocol int

const (
	FTP Protocol = iota
	FTPS
	SFTP
)

var protocolNames = [...]string{FTP: "FTP", FTPS: "FTPS", SFTP: "SFTP"}

func (p Protocol) String() string {
	if p < 0 || int(p) >= len(protocolNames) {
		return "FTP"
	}
	return protocolNames[p]
}

// DefaultPort is used when the user leaves the port blank.
func (p Protocol) DefaultPort() int {
	if p == SFTP {
		return 22
	}
	return 21
}

// Next and Prev cycle through the protocols, wrapping at both ends.
func (p Protocol) Next() Protocol {
	return Protocol((int(p) + 1) % len(protocolNames))
}

func (p Protocol) Prev() Protocol {
	return Protocol((int(p) + len(protocolNames) - 1) % len(protocolNames))
}

// New returns a client speaking the given protocol. A non-nil logger records
// the FTP control dialogue; SFTP has no equivalent and ignores it.
func New(p Protocol, logger io.Writer) Client {
	switch p {
	case SFTP:
		return NewSFTPClient()
	case FTPS:
		return NewFTPSClient(logger)
	default:
		return NewFTPClient(logger)
	}
}
