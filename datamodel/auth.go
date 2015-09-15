package datamodel

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"io"
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
