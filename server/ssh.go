package server

import (
	"errors"
	"fmt"

	"github.com/blacklabeldata/kappa/datamodel"

	"golang.org/x/crypto/ssh"
)

// PublicKeyCallback returns a function to validate public keys for user login.
func PublicKeyCallback(sys datamodel.System) (func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error), error) {
	if sys == nil {
		return nil, errors.New("ssh server: System cannot be nil")
	}

	// Get user store
	users, err := sys.Users()
	if err != nil {
		return nil, fmt.Errorf("ssh server: user store: %s", err)
	}

	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (perm *ssh.Permissions, err error) {
		// fmt.Println(string(key.Marshal()))

		// Get user if exists, otherwise return error
		user, err := users.Get(conn.User())
		if err != nil {
			return
		}

		// Check keyring for public key
		if keyring := user.KeyRing(); !keyring.Contains(key.Marshal()) {
			err = fmt.Errorf("invalid public key")
			return
		}

		// Add pubkey and username to permissions
		perm = &ssh.Permissions{
			Extensions: map[string]string{
				"pubkey":   string(key.Marshal()),
				"username": conn.User(),
			},
		}
		return
	}, nil
}
