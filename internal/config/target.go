package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

var schemes = map[string]string{"ftp": "FTP", "ftps": "FTPS", "sftp": "SFTP"}

// ParseTarget reads [scheme://][user@]host[:port]. The protocol is the scheme,
// else the one asked for by --protocol, else decided by the port: 22 is SFTP,
// 990 is FTPS, anything else FTP, which is also where the connection dialog
// starts. A password is refused: it would end up in the shell history and in
// the process list.
func ParseTarget(arg, protocol string) (Connection, error) {
	raw := arg
	if !strings.Contains(raw, "://") {
		raw = "//" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return Connection{}, fmt.Errorf("%q is not a valid destination", arg)
	}
	if u.User != nil && u.User.Username() == "" {
		return Connection{}, fmt.Errorf("%q has an empty user before the @", arg)
	}
	if _, has := u.User.Password(); has {
		return Connection{}, fmt.Errorf("a password cannot be given in the destination; lazyftp asks for it")
	}
	if p := strings.Trim(u.Path, "/"); p != "" || u.RawQuery != "" || u.Fragment != "" {
		return Connection{}, fmt.Errorf("%q: only user, host and port are accepted", arg)
	}

	c := Connection{Host: u.Hostname(), User: u.User.Username()}

	c.Protocol = protocol
	if u.Scheme != "" {
		proto, ok := schemes[strings.ToLower(u.Scheme)]
		if !ok {
			return Connection{}, fmt.Errorf("unknown protocol %q, expected ftp, ftps or sftp", u.Scheme)
		}
		if protocol != "" && protocol != proto {
			return Connection{}, fmt.Errorf("%q says %s but --protocol says %s", arg, proto, protocol)
		}
		c.Protocol = proto
	}

	if ps := u.Port(); ps != "" {
		port, err := strconv.Atoi(ps)
		if err != nil || port < 1 || port > 65535 {
			return Connection{}, fmt.Errorf("invalid port %q", ps)
		}
		c.Port = port
	}

	if c.Protocol == "" {
		c.Protocol = protocolForPort(c.Port)
	}
	if c.Port == 0 {
		c.Port = defaultPort(c.Protocol)
	}
	return c, nil
}

func protocolForPort(port int) string {
	switch port {
	case 22:
		return "SFTP"
	case 990:
		return "FTPS"
	}
	return "FTP"
}

func defaultPort(protocol string) int {
	if protocol == "SFTP" {
		return 22
	}
	return 21
}

// Resolve takes a saved name, from favorites or ssh_config, before treating
// the argument as a destination: a name the user chose beats a guess. A
// protocol asked for on the command line wins over a saved one.
func Resolve(arg, protocol string, saved ...[]Connection) (Connection, error) {
	for _, list := range saved {
		for _, c := range list {
			if c.Name == arg {
				if protocol != "" {
					c.Protocol = protocol
				}
				return c, nil
			}
		}
	}
	return ParseTarget(arg, protocol)
}

// ParseProtocolFlag validates a --protocol value; empty means "not given".
func ParseProtocolFlag(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if p, ok := schemes[strings.ToLower(s)]; ok {
		return p, nil
	}
	return "", fmt.Errorf("unknown protocol %q, expected ftp, ftps or sftp", s)
}
