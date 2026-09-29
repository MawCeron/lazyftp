package client

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// HostKeyPrompt asks whether to trust a host seen for the first time. It
// blocks until the user answers.
type HostKeyPrompt func(host, keyType, fingerprint string) bool

func knownHostsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}

// hostKeyCallback checks keys against the user's own known_hosts, the file ssh
// and scp share. With no prompt an unknown host is refused, never trusted.
// The handshake deadline is lifted while the user reads the fingerprint.
func hostKeyCallback(path string, prompt HostKeyPrompt, conn interface{ SetDeadline(time.Time) error }) (ssh.HostKeyCallback, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	f.Close()

	check, err := knownhosts.New(path)
	if err != nil {
		return nil, err
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := check(hostname, remote, key)
		var ke *knownhosts.KeyError
		if !errors.As(err, &ke) {
			return err
		}
		if len(ke.Want) > 0 {
			return fmt.Errorf("host key for %s has changed (%s:%d): expected %s, got %s. "+
				"This may be a machine-in-the-middle; if the server was reinstalled, remove that line and reconnect",
				hostname, ke.Want[0].Filename, ke.Want[0].Line,
				ssh.FingerprintSHA256(ke.Want[0].Key), ssh.FingerprintSHA256(key))
		}
		if prompt == nil {
			return fmt.Errorf("host %s is not in known_hosts and cannot be confirmed", hostname)
		}

		conn.SetDeadline(time.Time{})
		trusted := prompt(hostname, key.Type(), ssh.FingerprintSHA256(key))
		conn.SetDeadline(time.Now().Add(dialTimeout))
		if !trusted {
			return fmt.Errorf("host key for %s rejected", hostname)
		}

		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = fmt.Fprintln(f, knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key))
		return err
	}, nil
}
