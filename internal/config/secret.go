package config

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const service = "lazyftp"

// ErrNoSecret means nothing is stored for the connection, or the system has no
// keyring; either way the caller asks for the password.
var ErrNoSecret = errors.New("no stored password")

func account(c Connection) string {
	return fmt.Sprintf("%s://%s@%s:%d", c.Protocol, c.User, c.Host, c.Port)
}

func Secret(c Connection) (string, error) {
	s, err := keyring.Get(service, account(c))
	if err != nil {
		return "", ErrNoSecret
	}
	return s, nil
}

func SetSecret(c Connection, password string) error {
	return keyring.Set(service, account(c), password)
}

func DeleteSecret(c Connection) error {
	err := keyring.Delete(service, account(c))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
