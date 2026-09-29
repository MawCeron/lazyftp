package client

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var defaultKeys = []string{"id_ed25519", "id_ecdsa", "id_rsa"}

// sshAuth offers the agent, the default keys and then the password, and
// remembers what it offered: ssh reports which method failed but never which
// one succeeded.
type sshAuth struct {
	pass        string
	identity    string
	identityErr error
	sources     []string
	locked      []string
	agent       net.Conn
	method      string
}

func (a *sshAuth) methods() []ssh.AuthMethod {
	// ssh tries one publickey method per connection, so agent and files share it.
	m := []ssh.AuthMethod{ssh.PublicKeysCallback(a.signers)}
	if a.pass != "" {
		m = append(m, ssh.PasswordCallback(func() (string, error) {
			a.method = "password"
			return a.pass, nil
		}))
	}
	return m
}

func (a *sshAuth) signers() ([]ssh.Signer, error) {
	var out []ssh.Signer

	// A key the user named goes first, and failing to use it is reported: it
	// was a choice, unlike a default key that simply is not there.
	if a.identity != "" {
		path := expandHome(a.identity)
		s, err := a.loadKey(path)
		switch {
		case err == nil:
			out = append(out, s)
			a.sources = append(a.sources, filepath.Base(path))
		case errors.Is(err, errPassphrase):
			a.identityErr = fmt.Errorf("identity file %s is encrypted: put its passphrase in the password field", a.identity)
		default:
			a.identityErr = fmt.Errorf("identity file %s: %w", a.identity, err)
		}
	}

	// Windows agents speak a named pipe, not a unix socket, and are not reached.
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if conn, err := net.Dial("unix", sock); err == nil {
			a.agent = conn
			if s, err := agent.NewClient(conn).Signers(); err == nil && len(s) > 0 {
				out = append(out, s...)
				a.sources = append(a.sources, "ssh-agent")
			}
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		for _, name := range defaultKeys {
			s, err := a.loadKey(filepath.Join(home, ".ssh", name))
			if err == nil {
				out = append(out, s)
				a.sources = append(a.sources, name)
			} else if errors.Is(err, errPassphrase) {
				a.locked = append(a.locked, name)
			}
		}
	}

	a.method = "public key (" + strings.Join(a.sources, ", ") + ")"
	return out, nil
}

var errPassphrase = errors.New("passphrase missing or wrong")

// The password field doubles as the passphrase: the connection form has no
// second secret to ask for.
func (a *sshAuth) loadKey(path string) (ssh.Signer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s, err := ssh.ParsePrivateKey(b)
	var missing *ssh.PassphraseMissingError
	if !errors.As(err, &missing) {
		return s, err
	}
	if a.pass != "" {
		if s, err := ssh.ParsePrivateKeyWithPassphrase(b, []byte(a.pass)); err == nil {
			return s, nil
		}
	}
	return nil, errPassphrase
}

func (a *sshAuth) close() {
	if a.agent != nil {
		a.agent.Close()
	}
}

func expandHome(p string) string {
	if p != "~" && !strings.HasPrefix(p, "~/") && !strings.HasPrefix(p, `~\`) {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, p[1:])
}

// explain adds what the ssh error cannot know: keys that were skipped.
func (a *sshAuth) explain(err error) error {
	msg := err.Error()
	if a.identityErr != nil {
		msg += "; " + a.identityErr.Error()
	}
	if len(a.locked) > 0 {
		msg += "; encrypted key " + strings.Join(a.locked, ", ") + " needs its passphrase in the password field"
	}
	return errors.New(msg)
}
