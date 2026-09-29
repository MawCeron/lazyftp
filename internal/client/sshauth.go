package client

import (
	"errors"
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
	pass    string
	sources []string
	locked  []string
	agent   net.Conn
	method  string
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

// explain adds what the ssh error cannot know: keys that were skipped.
func (a *sshAuth) explain(err error) error {
	if len(a.locked) == 0 {
		return err
	}
	return errors.New(err.Error() + "; encrypted key " + strings.Join(a.locked, ", ") +
		" needs its passphrase in the password field")
}
