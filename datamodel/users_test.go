package datamodel

import (
    "bytes"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "crypto/x509"
    "encoding/pem"
    "io/ioutil"
    "math/big"
    "net"
    "os"
    "path"
    "sort"
    "time"

    "testing"

    "github.com/boltdb/bolt"
    "github.com/eliquious/leaf"
    log "github.com/mgutz/logxi/v1"
    "github.com/stretchr/testify/suite"

    "github.com/blacklabeldata/kappa/auth"
    "golang.org/x/crypto/ssh"
)

// TestUserTestSuite runs the UserTestSuite
func TestUserTestSuite(t *testing.T) {
    suite.Run(t, new(UserTestSuite))
}

// UserTestSuite tests all the auth routes
type UserTestSuite struct {
    suite.Suite
    Dir string
    DB  leaf.KeyValueDatabase
    US  UserStore
    KS  leaf.Keyspace
}

// SetupSuite prepares the suite before any tests are ran
func (suite *UserTestSuite) SetupSuite() {

    // Create temp directory
    suite.Dir, _ = ioutil.TempDir("", "datamodel.test")

    // Connect to database
    db, err := leaf.NewLeaf(path.Join(suite.Dir, "test.db"))
    if err != nil {
        suite.T().Log("Error creating database")
        suite.T().FailNow()
    }
    suite.DB = db

    // Create keyspace
    ks, err := db.GetOrCreateKeyspace(Users)
    suite.Nil(err)
    suite.KS = ks

    // Create user store
    suite.US = NewBoltUserStore(ks)
}

// TearDownSuite cleans up suite state after all the tests have completed
func (suite *UserTestSuite) TearDownSuite() {
    os.RemoveAll(suite.Dir)

    // Close database
    suite.DB.Close()
}

// SetupTest prepares each test before execution
func (suite *UserTestSuite) SetupTest() {
}

// TearDownTest cleans up after each test
func (suite *UserTestSuite) TearDownTest() {
}

// TestCreateUserError ensures a bolt db error bubbles up. Such as an empty user name;
func (suite *UserTestSuite) TestCreateUserError() {
    ns, err := suite.US.Create("")
    suite.Nil(ns)
    suite.NotNil(err)
}

// TestCreateUser ensures a user can be created
func (suite *UserTestSuite) TestCreateUser() {
    ns, err := suite.US.Create("acme")
    suite.Nil(err)
    suite.NotNil(ns)

    // Test that the user was created
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {

        b := bkt.Bucket([]byte("acme"))
        suite.NotNil(b)
    })
}

// TestGetUser ensures a user can be created
func (suite *UserTestSuite) TestGetUser() {

    // Test that the user does not exist prior
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {

        b := bkt.Bucket([]byte("acme.none"))
        suite.Nil(b)
    })

    // Get the user
    ns, err := suite.US.Get("acme.none")
    suite.NotNil(err)
    suite.Nil(ns)

    // Test that the user was not created
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {

        b := bkt.Bucket([]byte("acme.none"))
        suite.Nil(b)
    })
}

// TestDeleteUser ensures a user can be deleted
func (suite *UserTestSuite) TestDeleteUser() {
    ns, err := suite.US.Create("acme.delete")
    suite.Nil(err)
    suite.NotNil(ns)

    // Test that the user was created
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {

        b := bkt.Bucket([]byte("acme.delete"))
        suite.NotNil(b)
    })

    // Delete the user
    err = suite.US.Delete("acme.delete")
    suite.Nil(err)

    // Test that the user was deleted
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {

        b := bkt.Bucket([]byte("acme.delete"))
        suite.Nil(b)
    })
}

func (suite *UserTestSuite) verifyUserExists(name string) (exists bool) {

    // Test that the user was created
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {
        b := bkt.Bucket([]byte(name))
        exists = b != nil
    })
    return
}

func (suite *UserTestSuite) createUser(name string) (User, error) {
    ns, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(ns)
    return ns, err
}

func (suite *UserTestSuite) deleteUser(name string) {
    err := suite.US.Delete(name)
    suite.Nil(err)
}

// TestValidatePasswordNoUser
func (suite *UserTestSuite) TestValidatePasswordNoUser() {
    // user, err := suite.US.Create("acme.validate.password")
    // suite.Nil(err)
    // suite.NotNil(us)

    user := boltUser{[]byte("blahblahblah"), suite.KS}

    match := user.ValidatePassword("password")
    suite.False(match)
}

// TestValidatePasswordNoSalt
func (suite *UserTestSuite) TestValidatePasswordNoSalt() {
    user, err := suite.US.Create("acme.validate.password.no.salt")
    suite.Nil(err)
    suite.NotNil(user)

    match := user.ValidatePassword("password")
    suite.False(match)
}

// TestValidatePasswordNoSaltedPassword
func (suite *UserTestSuite) TestValidatePasswordNoSaltedPassword() {
    name := "acme.validate.password.no.saltedpw"
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Write salt
    suite.KS.WriteTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        // Generate salt
        salt, _, _ := GenerateSalt([]byte("password"))
        userBucket.Put([]byte("salt"), salt)
    })

    match := user.ValidatePassword("password")
    suite.False(match)
}

// TestValidatePasswordInvalid
func (suite *UserTestSuite) TestValidatePasswordInvalid() {
    name := "acme.validate.password.invalid"
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Write salt
    suite.KS.WriteTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        // Generate salt
        salt, saltedpw, _ := GenerateSalt([]byte("password"))
        userBucket.Put([]byte("salt"), salt)
        userBucket.Put([]byte("salted_password"), saltedpw)
    })

    // Test match
    match := user.ValidatePassword("shaken, not stirred")
    suite.False(match)
}

// TestValidatePassword
func (suite *UserTestSuite) TestValidatePassword() {
    name := "acme.validate.password"
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Write salt
    suite.KS.WriteTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        // Generate salt
        salt, saltedpw, _ := GenerateSalt([]byte("password"))
        userBucket.Put([]byte("salt"), salt)
        userBucket.Put([]byte("salted_password"), saltedpw)
    })

    // Test match
    match := user.ValidatePassword("password")
    suite.True(match)
}

func (suite *UserTestSuite) TestUpdatePasswordInvalidUser() {
    user := boltUser{[]byte("blahblahblah"), suite.KS}

    err := user.UpdatePassword("password")
    suite.NotNil(err)
}

func (suite *UserTestSuite) TestUpdatePassword() {
    name := "acme.update.password"

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Update password
    err = user.UpdatePassword("password")
    suite.Nil(err)

    // Validate salt + salted_password
    suite.KS.WriteTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        salt := userBucket.Get([]byte("salt"))
        suite.NotNil(salt)

        saltedPassword := userBucket.Get([]byte("salted_password"))
        suite.NotNil(saltedPassword)

        // Salt password
        hash := sha256.New()
        hash.Write(salt)
        hash.Write([]byte("password"))

        match := SecureCompare(hash.Sum(nil), saltedPassword)
        suite.True(match)
    })
}

func (suite *UserTestSuite) TestUserNamespacesInvalidUser() {
    user := boltUser{[]byte("blahblahblah"), suite.KS}

    // Get namespaces
    nss := user.Namespaces()
    suite.Nil(nss)
}

func (suite *UserTestSuite) TestUserNamespaces() {
    name := "acme.user.namespaces"

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Validate salt + salted_password
    suite.KS.WriteTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        ns, err := userBucket.CreateBucketIfNotExists([]byte("namespaces"))
        suite.Nil(err)

        ns.Put([]byte("acme.users"), []byte("guest"))
        ns.Put([]byte("acme.trending"), []byte("guest"))
    })

    // Get namespaces
    nss := user.Namespaces()
    suite.NotNil(nss)
    suite.Equal(len(nss), 2)
    suite.Equal(nss[0], "acme.trending")
    suite.Equal(nss[1], "acme.users")
}

func (suite *UserTestSuite) TestUserRolesInvalidUser() {
    user := boltUser{[]byte("blahblahblah"), suite.KS}

    // Get roles
    nss := user.Roles("acme.namespace")
    suite.Nil(nss)
}

func (suite *UserTestSuite) TestUserRoles() {
    name := "acme.user.roles"

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Validate salt + salted_password
    suite.KS.WriteTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        ns, err := userBucket.CreateBucketIfNotExists([]byte("namespaces"))
        suite.Nil(err)

        ns.Put([]byte("acme.users"), []byte("guest,admin"))
        ns.Put([]byte("acme.trending"), []byte("admin"))
    })

    // Get Roles for invalid namespace
    roles := user.Roles("invalid.namespace")
    suite.Equal(len(roles), 0)
    suite.Nil(roles)

    // Get roles for acme.users (multiple roles)
    roles = user.Roles("acme.users")
    suite.Equal(len(roles), 2)
    suite.NotNil(roles)
    suite.Equal("guest", roles[0])
    suite.Equal("admin", roles[1])

    // Get roles for acme.trending (single role)
    roles = user.Roles("acme.trending")
    suite.Equal(len(roles), 1)
    suite.NotNil(roles)
    suite.Equal(roles[0], "admin")
}

func (suite *UserTestSuite) TestAddRoleInvalidUser() {
    user := boltUser{[]byte("blahblahblah"), suite.KS}

    // Add role
    err := user.AddRole("acme", "acme.add.user")
    suite.NotNil(err)
}

func (suite *UserTestSuite) TestAddRole() {
    name := "acme.user.roles"

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Add role
    err = user.AddRole("acme.namespace", "create.log")
    suite.Nil(err)

    // Validate roles
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        ns := userBucket.Bucket([]byte("namespaces"))
        suite.NotNil(ns)

        roles := ns.Get([]byte("acme.namespace"))
        suite.NotNil(roles)
        suite.Equal(roles, []byte("create.log"))
    })

    // Add second role
    err = user.AddRole("acme.namespace", "create.view")
    suite.Nil(err)

    // Validate roles
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        ns := userBucket.Bucket([]byte("namespaces"))
        suite.NotNil(ns)

        roles := ns.Get([]byte("acme.namespace"))
        suite.NotNil(roles)
        suite.Equal(roles, []byte("create.log,create.view"))
    })
}

func (suite *UserTestSuite) TestRemoveRoleInvalidUser() {
    user := boltUser{[]byte("blahblahblah"), suite.KS}

    // Remove role
    err := user.RemoveRole("acme", "acme.remove.user")
    suite.NotNil(err)
}

func (suite *UserTestSuite) TestRemoveRole() {
    name := "acme.user.remove.role"

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Add role
    err = user.AddRole("acme.namespace", "create.log")
    suite.Nil(err)

    // Validate roles
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        ns := userBucket.Bucket([]byte("namespaces"))
        suite.NotNil(ns)

        roles := ns.Get([]byte("acme.namespace"))
        suite.NotNil(roles)
        suite.Equal(roles, []byte("create.log"))
    })

    // Remove role
    err = user.RemoveRole("acme.namespace", "create.log")
    suite.Nil(err)

    // Validate roles
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        ns := userBucket.Bucket([]byte("namespaces"))
        suite.NotNil(ns)

        roles := ns.Get([]byte("acme.namespace"))
        suite.NotNil(roles)
        suite.Equal(roles, []byte(""))
    })
}

func (suite *UserTestSuite) TestRemoveRoleMultiple() {
    name := "acme.user.remove.role"

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Add role
    err = user.AddRole("acme.namespace", "create.log")
    suite.Nil(err)

    // Add role
    err = user.AddRole("acme.namespace", "create.view")
    suite.Nil(err)

    // Validate roles
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        ns := userBucket.Bucket([]byte("namespaces"))
        suite.NotNil(ns)

        roles := ns.Get([]byte("acme.namespace"))
        suite.NotNil(roles)
        suite.Equal(roles, []byte("create.log,create.view"))
    })

    // Remove role
    err = user.RemoveRole("acme.namespace", "create.log")
    suite.Nil(err)

    // Validate roles
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        ns := userBucket.Bucket([]byte("namespaces"))
        suite.NotNil(ns)

        roles := ns.Get([]byte("acme.namespace"))
        suite.NotNil(roles)
        suite.Equal(roles, []byte("create.view"))
    })
}

func (suite *UserTestSuite) generateCertificate() []byte {

    // generate private key
    privatekey, err := rsa.GenerateKey(rand.Reader, 2048)
    suite.Nil(err)

    // Create Certificate request
    csr, _, err := auth.CreateCertificateRequest(log.NullLog, privatekey, "kappa", "kappa", "US", "127.0.0.1")
    suite.Nil(err)

    // Generate subject key id
    subjectKeyID, err := auth.GenerateSubjectKeyID(privatekey)
    suite.Nil(err)

    // Create serial number
    serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
    serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
    suite.Nil(err)

    // Create template
    template := &x509.Certificate{
        IsCA: false,
        BasicConstraintsValid: false,
        SubjectKeyId:          subjectKeyID,
        SerialNumber:          serialNumber,
        Subject:               csr.Subject,
        PublicKeyAlgorithm:    x509.RSA,
        SignatureAlgorithm:    x509.SHA512WithRSA,
        NotBefore:             time.Now().Add(-600).UTC(),
        NotAfter:              time.Now().AddDate(10, 0, 0).UTC(),
        IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},

        // see http://golang.org/pkg/crypto/x509/#KeyUsage
        ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
        KeyUsage:           x509.KeyUsageDigitalSignature,
        UnknownExtKeyUsage: nil,

        // Subject Alternative Name
        DNSNames: nil,

        PermittedDNSDomainsCritical: false,
        PermittedDNSDomains:         nil,
    }

    // Create cert
    crt, err := x509.CreateCertificate(rand.Reader, template, template, &privatekey.PublicKey, privatekey)
    suite.Nil(err)

    return crt
}

func (suite *UserTestSuite) TestAddPublicKey() {
    name := "acme.user.add.key"

    // Create Certificate
    crt := suite.generateCertificate()

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Get key ring
    keyRing := user.KeyRing()
    suite.NotNil(keyRing)

    // Encode cert
    pemFile := new(bytes.Buffer)
    pemkey := &pem.Block{
        Type:  "CERTIFICATE",
        Bytes: crt}
    pem.Encode(pemFile, pemkey)

    // Add key
    fp, err := keyRing.AddPublicKey(pemFile.Bytes())
    suite.Nil(err)

    // Validate key
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        keys := userBucket.Bucket([]byte("keys"))
        suite.NotNil(keys)

        // Pub key
        pub, err := x509.ParseCertificate(crt)
        suite.Nil(err)

        // Convert public key to SSH key format
        sshKey, err := ssh.NewPublicKey(pub.PublicKey)
        suite.Nil(err)

        // Convert key to bytes
        key := sshKey.Marshal()
        fingerprint := auth.CreateFingerprint(key)

        // Get key
        suite.Equal(key, keys.Get([]byte(fingerprint)))

        // Verify fingerprint
        suite.Equal(fingerprint, fp)
    })
}

func (suite *UserTestSuite) TestAddPublicKeyInvalidCertificate() {
    name := "acme.user.add.key"

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Get key ring
    keyRing := user.KeyRing()
    suite.NotNil(keyRing)

    // Add key
    bytes := []byte("")
    _, err = keyRing.AddPublicKey(bytes)
    suite.NotNil(err)

}

func (suite *UserTestSuite) TestAddPublicKeyInvalidUser() {
    user := boltUser{[]byte("blahblahblah"), suite.KS}

    // Get key ring
    keyRing := user.KeyRing()
    suite.NotNil(keyRing)

    // Add key
    bytes := []byte("blahblah")
    _, err := keyRing.AddPublicKey(bytes)
    suite.NotNil(err)
}

func (suite *UserTestSuite) TestAddPublicKeyInvalidKey() {
    name := "acme.user.add.invalid.key"

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Get key ring
    keyRing := user.KeyRing()
    suite.NotNil(keyRing)

    // Add key
    bytes := []byte("blahblah")
    _, err = keyRing.AddPublicKey(bytes)
    suite.NotNil(err)
}

func (suite *UserTestSuite) TestRemovePublicKeyInvalidUser() {
    user := boltUser{[]byte("blahblahblah"), suite.KS}

    // Get key ring
    keyRing := user.KeyRing()
    suite.NotNil(keyRing)

    // Remove key
    err := keyRing.RemovePublicKey("bytes")
    suite.NotNil(err)
}

func (suite *UserTestSuite) TestRemovePublicKey() {
    name := "acme.user.remove.key"

    // Create Certificate
    crt := suite.generateCertificate()

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Get key ring
    keyRing := user.KeyRing()
    suite.NotNil(keyRing)

    // Encode cert
    pemFile := new(bytes.Buffer)
    pemkey := &pem.Block{
        Type:  "CERTIFICATE",
        Bytes: crt}
    pem.Encode(pemFile, pemkey)

    // Add key
    fp, err := keyRing.AddPublicKey(pemFile.Bytes())
    suite.Nil(err)

    // Remove Key
    err = keyRing.RemovePublicKey(fp)
    suite.Nil(err)

    // Validate key
    suite.KS.ReadTx(func(bkt *bolt.Bucket) {
        userBucket := bkt.Bucket([]byte(name))

        keys := userBucket.Bucket([]byte("keys"))
        suite.NotNil(keys)

        // Verify key removed
        suite.Nil(keys.Get([]byte(fp)))
    })
}

func (suite *UserTestSuite) TestListPublicKeys() {
    name := "acme.user.list.keys"

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Get key ring
    keyRing := user.KeyRing()
    suite.NotNil(keyRing)

    // Create Certificate # 1
    crt := suite.generateCertificate()

    // Encode cert
    pemFile := new(bytes.Buffer)
    pemkey := &pem.Block{
        Type:  "CERTIFICATE",
        Bytes: crt}
    pem.Encode(pemFile, pemkey)

    // Add key
    fp1, err := keyRing.AddPublicKey(pemFile.Bytes())
    suite.Nil(err)

    // Create Certificate # 2
    crt = suite.generateCertificate()

    // Encode cert
    pemFile = new(bytes.Buffer)
    pemkey = &pem.Block{
        Type:  "CERTIFICATE",
        Bytes: crt}
    pem.Encode(pemFile, pemkey)

    // Add key
    fp2, err := keyRing.AddPublicKey(pemFile.Bytes())
    suite.Nil(err)

    // Validate keys
    keys := keyRing.ListPublicKeys()
    suite.Equal(2, len(keys))

    // Fingerprints
    fingerprints := []string{fp1, fp2}
    sort.Strings(fingerprints)

    // Validate fingerprints
    suite.Equal(fingerprints[0], keys[0].Fingerprint())
    suite.Equal(fingerprints[1], keys[1].Fingerprint())
}

func (suite *UserTestSuite) TestListPublicKeysInvalidUser() {
    user := boltUser{[]byte("blahblahblah"), suite.KS}

    // Get key ring
    keyRing := user.KeyRing()
    suite.NotNil(keyRing)

    // List keys
    keys := keyRing.ListPublicKeys()
    suite.Equal(0, len(keys))
}

func (suite *UserTestSuite) TestContainsPublicKeys() {
    name := "acme.user.contains.key"

    // Create user
    user, err := suite.US.Create(name)
    suite.Nil(err)
    suite.NotNil(user)

    // Get key ring
    keyRing := user.KeyRing()
    suite.NotNil(keyRing)

    // Create Certificate # 1
    crt := suite.generateCertificate()

    // Encode cert
    pemFile := new(bytes.Buffer)
    pemkey := &pem.Block{
        Type:  "CERTIFICATE",
        Bytes: crt}
    pem.Encode(pemFile, pemkey)

    // Add key
    _, err = keyRing.AddPublicKey(pemFile.Bytes())
    suite.Nil(err)

    // Validate keys
    keys := keyRing.ListPublicKeys()
    suite.Equal(1, len(keys))

    // Validate fingerprints
    suite.True(keyRing.Contains(keys[0].sshKey))
}

func (suite *UserTestSuite) TestContainsPublicKeyInvalidUser() {
    user := boltUser{[]byte("blahblahblah"), suite.KS}

    // Get key ring
    keyRing := user.KeyRing()
    suite.NotNil(keyRing)

    // List keys
    var buf []byte
    exists := keyRing.Contains(buf)
    suite.False(exists)
}
