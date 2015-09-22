package datamodel

import (
    "crypto/sha256"
    "crypto/x509"
    "encoding/pem"
    "fmt"
    "strings"

    "github.com/blacklabeldata/kappa/auth"
    "github.com/boltdb/bolt"
    "github.com/eliquious/leaf"
    "golang.org/x/crypto/ssh"
)

var (

    // ErrUserDoesNotExist signifies that a user does not exist
    ErrUserDoesNotExist = fmt.Errorf("user does not exist")

    // ErrInvalidCertificate is returned when the certificate can't be decoded
    ErrInvalidCertificate = fmt.Errorf("unable to load certificate")

    // ErrFailedKeyConvertion means that the public key could not be converted to an SSH key
    ErrFailedKeyConvertion = fmt.Errorf("error converting public key to SSH key format")
)

// PublicKey wraps an ssh.PublicKey byte array and simply provides methods for validation.
type PublicKey struct {
    fingerprint []byte
    sshKey      []byte
}

// Fingerprint provides a string hash representing a PublicKey
func (p *PublicKey) Fingerprint() string {
    return string(p.fingerprint)
}

// Equals determines the equivalence of two PublicKeys
func (p *PublicKey) Equals(key []byte) bool {
    return SecureCompare(p.sshKey, key)
}

// PublicKeyRing provides an interface for interacting with a user's public keys
type PublicKeyRing interface {

    // AddPublicKey simply adds a public key to the user's key ring
    AddPublicKey(pemBytes []byte) (string, error)

    // RemovePublicKey will remove a public key from a user's key ring
    RemovePublicKey(fingerprint string) error

    // ListPublicKey returns all of a user's public keys
    ListPublicKeys() []PublicKey

    // Contains determines if a key exists in the ring. The provided bytes should be the output of ssh.PublicKey.Marshal.
    Contains(key []byte) bool
}

// User represents a database user
type User interface {

    // Username returns the user alias
    Username() string

    // IsAdmin returns whether the user has admin priviliges
    IsAdmin() bool

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

    // AddRole appends a role to namespace
    AddRole(namespace, role string) error

    // RemoveRole removed a role for a namespace
    RemoveRole(namespace, role string) error
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

// Get returns a User, returning an error if it doesn't exist
func (b boltUserStore) Get(name string) (u User, err error) {
    b.ks.ReadTx(func(bkt *bolt.Bucket) {

        // Get user bucket
        userBucket := bkt.Bucket([]byte(name))
        if userBucket == nil {
            err = ErrUserDoesNotExist
            return
        }
        u = boltUser{[]byte(name), b.ks}
        return
    })
    return
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
    name  []byte
    users leaf.Keyspace
}

// IsAdmin returns whether the user is an admin
func (b boltUser) IsAdmin() bool {
    return string(b.name) == "admin"
}

// Username returns the user alias
func (b boltUser) Username() string {
    return string(b.name)
}

// ValidatePassword determines the validity of a password.
func (b boltUser) ValidatePassword(password string) (match bool) {
    b.users.ReadTx(func(bkt *bolt.Bucket) {

        // Get user bucket
        user := bkt.Bucket(b.name)

        // If user is nil, the user does not exist
        if user == nil {
            return
        }

        // Get salt, if salt is nil return false.
        // If the salt is nil, a user password has not been set
        salt := user.Get([]byte("salt"))
        if salt == nil {
            return
        }

        // Get the salted password
        saltedpw := user.Get([]byte("salted_password"))
        if saltedpw == nil {
            return
        }

        // Salt password
        hash := sha256.New()
        hash.Write(salt)
        hash.Write([]byte(password))

        // compare byte strings
        match = SecureCompare(hash.Sum(nil), saltedpw)
        return
    })
    return
}

// UpdatePassword updates a user's password. This password is only used to log into the web ui.
func (b boltUser) UpdatePassword(password string) (err error) {
    b.users.WriteTx(func(bkt *bolt.Bucket) {

        // Get user bucket
        user := bkt.Bucket(b.name)

        // If user is nil, the user does not exist
        if user == nil {
            err = ErrUserDoesNotExist
            return
        }

        // Generate salt and salted password
        salt, saltedpw, err := GenerateSalt([]byte(password))
        if err != nil {
            return
        }

        // Save salt
        if err = user.Put([]byte("salt"), salt); err != nil {
            return
        }

        // Save salted password
        if err = user.Put([]byte("salted_password"), saltedpw); err != nil {
            return
        }
        return
    })
    return
}

// KeyRing returns a PublicKeyRing containing all of a user's public keys
func (b boltUser) KeyRing() PublicKeyRing {
    return &boltKeyRing{b.name, b.users}
}

// Namespaces returns a list of namespaces for which the user has access
func (b boltUser) Namespaces() (ns []string) {
    b.users.WriteTx(func(bkt *bolt.Bucket) {

        // Get user bucket
        user := bkt.Bucket(b.name)

        // If user is nil, the user does not exist
        if user == nil {
            return
        }

        // Get namespace sub-bucket
        namespaces, err := user.CreateBucketIfNotExists([]byte("namespaces"))
        if err != nil {
            return
        }

        // Iterate and append namespace name
        namespaces.ForEach(func(k []byte, _ []byte) error {
            ns = append(ns, string(k))
            return nil
        })
        return
    })
    return
}

// Roles returns the user's roles for the given namespace
func (b boltUser) Roles(namespace string) (roles []string) {
    b.users.WriteTx(func(bkt *bolt.Bucket) {

        // Get user bucket
        user := bkt.Bucket(b.name)

        // If user is nil, the user does not exist
        if user == nil {
            return
        }

        // Get namespace sub-bucket
        namespaces, err := user.CreateBucketIfNotExists([]byte("namespaces"))
        if err != nil {
            return
        }

        // Get roles for the given namespace
        namespaceRoles := namespaces.Get([]byte(namespace))
        if len(namespaceRoles) > 0 {
            roles = strings.Split(string(namespaceRoles), ",")
        }
        return
    })
    return
}

// AddRole appends a role to the given namespace
func (b boltUser) AddRole(namespace, role string) (err error) {
    b.users.WriteTx(func(bkt *bolt.Bucket) {

        // Get user bucket
        ns := bkt.Bucket(b.name)
        if ns == nil {
            err = ErrUserDoesNotExist
            return
        }

        // Get namespaces bucket
        namespaces, err := ns.CreateBucketIfNotExists([]byte("namespaces"))
        if err != nil {
            return
        }

        // Get existing roles and add new
        roles := namespaces.Get([]byte(namespace))
        if len(roles) > 0 {

            list := []string{string(roles), role}
            namespaces.Put([]byte(namespace), []byte(strings.Join(list, ",")))
        } else {
            namespaces.Put([]byte(namespace), []byte(role))
        }
        return
    })
    return
}

// RemoveRole removes the role from the given namespace
func (b boltUser) RemoveRole(namespace, role string) (err error) {
    b.users.WriteTx(func(bkt *bolt.Bucket) {

        // Get namespace bucket
        ns := bkt.Bucket(b.name)
        if ns == nil {
            err = ErrUserDoesNotExist
            return
        }

        // Get namespaces bucket
        namespaces, err := ns.CreateBucketIfNotExists([]byte("namespaces"))
        if err != nil {
            return
        }

        // Get existing roles and add new
        roles := namespaces.Get([]byte(namespace))
        if len(roles) > 0 {

            var list []string
            for _, r := range strings.Split(string(roles), ",") {
                if r != role {
                    list = append(list, r)
                }
            }

            // Save roles
            err = namespaces.Put([]byte(namespace), []byte(strings.Join(list, ",")))
        }
        return
    })
    return
}

type boltKeyRing struct {
    username []byte
    users    leaf.Keyspace
}

// AddPublicKey simply adds a public key to the user's key ring
func (b *boltKeyRing) AddPublicKey(pemBytes []byte) (fingerprint string, e error) {
    b.users.WriteTx(func(bkt *bolt.Bucket) {
        if len(pemBytes) == 0 {
            e = ErrInvalidCertificate
            return
        }

        // Get user bucket
        user := bkt.Bucket(b.username)

        // If user is nil, the user does not exist
        if user == nil {
            e = ErrUserDoesNotExist
            return
        }

        // Get keys sub-bucket
        keys, err := user.CreateBucketIfNotExists([]byte("keys"))
        if err != nil {
            return
        }

        // Decode PEM bytes
        block, _ := pem.Decode(pemBytes)
        if block == nil {
            e = ErrInvalidCertificate
            return
        }

        pub, err := x509.ParseCertificate(block.Bytes)
        if err != nil {
            e = ErrInvalidCertificate
            return
        }

        // Convert Public Key to SSH format
        sshKey, err := ssh.NewPublicKey(pub.PublicKey)
        if err != nil {
            e = ErrFailedKeyConvertion
            return
        }

        // Convert key to bytes
        key := sshKey.Marshal()
        fingerprint = auth.CreateFingerprint(key)

        // Write key to keys bucket
        e = keys.Put([]byte(fingerprint), key)
        return
    })
    return
}

// RemovePublicKey will remove a public key from a user's key ring
func (b *boltKeyRing) RemovePublicKey(fingerprint string) (err error) {
    b.users.WriteTx(func(bkt *bolt.Bucket) {

        // Get user bucket
        user := bkt.Bucket(b.username)

        // If user is nil, the user does not exist
        if user == nil {
            err = ErrUserDoesNotExist
            return
        }

        // Get keys sub-bucket
        keys, err := user.CreateBucketIfNotExists([]byte("keys"))
        if err != nil {
            return
        }

        // Delete finger print
        err = keys.Delete([]byte(fingerprint))
        return
    })
    return
}

// ListPublicKey returns all of a user's public keys
func (b *boltKeyRing) ListPublicKeys() (publicKeys []PublicKey) {
    b.users.WriteTx(func(bkt *bolt.Bucket) {

        // Get user bucket
        user := bkt.Bucket(b.username)

        // If user is nil, the user does not exist
        if user == nil {
            return
        }

        // Get keys sub-bucket
        keys, err := user.CreateBucketIfNotExists([]byte("keys"))
        if err != nil {
            return
        }

        // Public keys are stored as fingerprint : key
        keys.ForEach(func(k []byte, v []byte) error {
            publicKeys = append(publicKeys, PublicKey{k, v})
            return nil
        })
        return
    })
    return
}

// Contains determines if a key exists in the ring. The provided bytes should be the output of ssh.PublicKey.Marshal.
func (b *boltKeyRing) Contains(key []byte) (exists bool) {
    b.users.WriteTx(func(bkt *bolt.Bucket) {

        // Get user bucket
        user := bkt.Bucket(b.username)

        // If user is nil, the user does not exist
        if user == nil {
            return
        }

        // Get keys sub-bucket
        keys, err := user.CreateBucketIfNotExists([]byte("keys"))
        if err != nil {
            return
        }

        // Create Fingerprint
        fingerprint := auth.CreateFingerprint(key)

        // Get fingerprint
        exists = keys.Get([]byte(fingerprint)) != nil
        return
    })
    return
}
