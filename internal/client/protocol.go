package client

import "io"

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

// DefaultPort applies when the port field is left blank.
func (p Protocol) DefaultPort() int {
	if p == SFTP {
		return 22
	}
	return 21
}

func (p Protocol) Next() Protocol {
	return Protocol((int(p) + 1) % len(protocolNames))
}

func (p Protocol) Prev() Protocol {
	return Protocol((int(p) + len(protocolNames) - 1) % len(protocolNames))
}

// SFTP has no control dialogue and ignores logger.
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
