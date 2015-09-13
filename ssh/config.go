package ssh

import (
	"sync"
	"time"

	"github.com/blacklabeldata/kappa/ssh/handlers"
	log "github.com/mgutz/logxi/v1"
	"golang.org/x/crypto/ssh"
)

// Config is used to setup the SSHServer.
type Config struct {
	sync.Mutex

	// Deadline is the maximum time the listener will block
	// between connections. As a consequence, this duration
	// also sets the max length of time the SSH server will
	// be unresponsive before shutting down.
	Deadline time.Duration

	// Handlers is an array of SSHHandlers which process incoming connections
	Handlers map[string]handlers.SSHHandler

	// Logger logs errors and debug output for the SSH server
	Logger log.Logger

	// Bind specifies the Bind address the SSH server will listen on
	Bind string

	// PrivateKey is added to the SSH config as a host key
	PrivateKey ssh.Signer

	// AuthLogCallback, if non-nil, is called to log all authentication
	// attempts.
	AuthLogCallback func(conn ssh.ConnMetadata, method string, err error)

	// PasswordCallback, if non-nil, is called when a user
	// attempts to authenticate using a password.
	PasswordCallback func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error)

	// System is the System datamodel
	PublicKeyCallback func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error)

	// sshConfig is used to verify incoming connections
	sshConfig *ssh.ServerConfig
}

func (c *Config) SSHConfig() (*ssh.ServerConfig, error) {

	// Set a default auth log function
	if c.AuthLogCallback == nil {
		c.AuthLogCallback = func(conn ssh.ConnMetadata, method string, err error) {
			if err == nil {
				c.Logger.Info("Successful login", "user", conn.User(), "method", method)
			}
		}
	}

	// Create server config
	sshConfig := &ssh.ServerConfig{
		NoClientAuth:      false,
		PasswordCallback:  c.PasswordCallback,
		PublicKeyCallback: c.PublicKeyCallback,
		AuthLogCallback:   c.AuthLogCallback,
	}
	sshConfig.AddHostKey(c.PrivateKey)
	return sshConfig, nil
}

func (c *Config) Handler(channel string) (handler handlers.SSHHandler, ok bool) {
	c.Lock()
	handler, ok = c.Handlers[channel]
	c.Unlock()
	return
}
