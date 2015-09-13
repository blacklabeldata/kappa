package datamodel

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

// SaltSize is the size of the salt for encrypting passwords
const SaltSize = 16

// GenerateSalt creates a new salt and encodes the given password.
// It returns the new salt, the ecrypted password and a possible error
func GenerateSalt(secret []byte) ([]byte, []byte, error) {
	buf := make([]byte, SaltSize, SaltSize+sha256.Size)
	_, err := io.ReadFull(rand.Reader, buf)

	if err != nil {
		return nil, nil, err
	}

	hash := sha256.New()
	hash.Write(buf)
	hash.Write(secret)
	return buf, hash.Sum(nil), nil
}

// SecureCompare compares salted passwords in constant time
// http://stackoverflow.com/questions/20663468/secure-compare-of-strings-in-go
func SecureCompare(given, actual []byte) bool {
	if subtle.ConstantTimeEq(int32(len(given)), int32(len(actual))) == 1 {
		return subtle.ConstantTimeCompare(given, actual) == 1
	}

	/* Securely compare actual to itself to keep constant time, but always return false */
	return subtle.ConstantTimeCompare(actual, actual) == 1 && false
}

// PublicKeyCallback returns a function to validate public keys for user login.
func PublicKeyCallback(sys System) (func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error), error) {
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
