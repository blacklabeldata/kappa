package datamodel

import (
    "github.com/boltdb/bolt"
    "github.com/eliquious/leaf"
)

// PublicKey wraps an ssh.PublicKey and simply provides methods for validation.
type PublicKey interface {

    // Fingerprint provides a string hash representing a PublicKey
    Fingerprint() string

    // Equals determines the equivalence of two PublicKeys
    Equals([]byte) bool
}

// PublicKeyRing provides an interface for interacting with a user's public keys
type PublicKeyRing interface {

    // AddPublicKey simply adds a public key to the user's key ring
    AddPublicKey(pemBytes []byte) error

    // RemovePublicKey will remove a public key from a user's key ring
    RemovePublicKey(fingerprint string) error

    // ListPublicKey returns all of a user's public keys
    ListPublicKeys() []PublicKey

    // Contains determines if a key exists in the ring. The provided bytes should be the output of ssh.PublicKey.Marshal.
    Contains(key []byte) bool
}

// User represents a database user
type User interface {

    // ValidatePassword determines the validity of a password.
    ValidatePassword(password string) bool

    // UpdatePassword updates a user's password. This password is only used to log into the web ui.
    UpdatePassword(password string) error

    // KeyRing returns a PublicKeyRing containing all of a user's public keys
    KeyRing() PublicKeyRing

    // Namespaces returns a list of namespaces for which the user has access
    Namespaces() []string

    // Roles returns the user's roles for the given namespace
    Roles(namespace string) []string
}

// UserStore stores all user information
type UserStore interface {

    // Get returns a User by username
    Get(username string) (User, error)

    // Create inserts a new user
    Create(username string) (User, error)

    // Delete removes a user account from a namespace
    Delete(username string) error
}

// NewBoltUserStore returns a UserStore backed by boltdb. If the user keyspace does not already exist, it will be created.
func NewBoltUserStore(ks leaf.Keyspace) UserStore {
    return &boltUserStore{ks}
}

// boltUserStore implements the UserStore interface
type boltUserStore struct {
    ks leaf.Keyspace
}

// Create adds a user to the database
func (b boltUserStore) Create(name string) (u User, err error) {
    b.ks.WriteTx(func(bkt *bolt.Bucket) {

        // Create bucket
        if _, err = bkt.CreateBucketIfNotExists([]byte(name)); err == nil {
            u = boltUser{[]byte(name), b.ks}
        }
        return
    })
    return
}

// Get returns a User, creating it if doesn't exist
func (b boltUserStore) Get(name string) (User, error) {
    return b.Create(name)
}

// Delete removes a user from the database
func (b boltUserStore) Delete(name string) (err error) {
    b.ks.WriteTx(func(bkt *bolt.Bucket) {

        // Delete bucket
        err = bkt.DeleteBucket([]byte(name))
        return
    })
    return
}

// boltUser implements the User interface on top of boltdb
type boltUser struct {
    name       []byte
    namespaces leaf.Keyspace
}

// ValidatePassword determines the validity of a password.
func (b boltUser) ValidatePassword(password string) bool {
    return false
}

// UpdatePassword updates a user's password. This password is only used to log into the web ui.
func (b boltUser) UpdatePassword(password string) error {
    return nil
}

// KeyRing returns a PublicKeyRing containing all of a user's public keys
func (b boltUser) KeyRing() PublicKeyRing {
    return nil
}

// Namespaces returns a list of namespaces for which the user has access
func (b boltUser) Namespaces() []string {
    return nil
}

// Roles returns the user's roles for the given namespace
func (b boltUser) Roles(namespace string) []string {
    return nil
}
