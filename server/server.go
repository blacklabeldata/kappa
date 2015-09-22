package server

import (
	"crypto/x509"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/blacklabeldata/kappa/auth"
	"github.com/blacklabeldata/kappa/datamodel"
	"github.com/blacklabeldata/sshh"
	log "github.com/mgutz/logxi/v1"
	"golang.org/x/crypto/ssh"
)

func NewServer(c *DatabaseConfig) (server *Server, err error) {

	// Create logger
	if c.LogOutput == nil {
		c.LogOutput = log.NewConcurrentWriter(os.Stdout)
	}
	logger := log.NewLogger(c.LogOutput, "kappa")

	// Create data directory
	if err = os.MkdirAll(c.DataPath, os.ModeDir|0655); err != nil {
		logger.Warn("Could not create data directory", "err", err.Error())
		return
	}

	// Connect to database
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("Could not get working directory", "error", err.Error())
		return
	}

	file := path.Join(cwd, c.DataPath, "meta.db")
	logger.Info("Connecting to database", "file", file)
	system, err := datamodel.NewSystem(file)
	if err != nil {
		logger.Error("Could not connect to database", "error", err.Error())
		return
	}

	// Get SSH Key file
	sshKeyFile := c.SSHPrivateKeyFile
	logger.Info("Reading private key", "file", sshKeyFile)

	privateKey, err := auth.ReadPrivateKey(logger, sshKeyFile)
	if err != nil {
		return
	}

	// Get admin certificate
	adminCertFile := c.AdminCertificateFile
	logger.Info("Reading admin public key", "file", adminCertFile)

	// Read admin certificate
	cert, err := ioutil.ReadFile(adminCertFile)
	if err != nil {
		logger.Error("admin certificate could not be read", "filename", c.AdminCertificateFile)
		return
	}

	// Add admin cert to key ring
	userStore, err := system.Users()
	if err != nil {
		logger.Error("could not get user store", "error", err.Error())
		return
	}

	// Create admin account
	admin, err := userStore.Create("admin")
	if err != nil {
		logger.Error("error creating admin account", "error", err.Error())
		return
	}

	// Add admin certificate
	keyRing := admin.KeyRing()
	fingerprint, err := keyRing.AddPublicKey(cert)
	if err != nil {
		logger.Error("admin certificate could not be added", "error", err.Error())
		return
	}
	logger.Info("Added admin certificate", "fingerprint", fingerprint)

	// Read root cert
	rootPem, err := ioutil.ReadFile(c.CACertificateFile)
	if err != nil {
		logger.Error("root certificate could not be read", "filename", c.CACertificateFile)
		return
	}

	// Create certificate pool
	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(rootPem); !ok {
		logger.Error("failed to parse root certificate")
		return
	}

	// Setup SSH Server
	sshLogger := log.NewLogger(c.LogOutput, "ssh")
	pubKeyCallback, err := PublicKeyCallback(system)
	if err != nil {
		logger.Error("failed to create PublicKeyCallback", err)
		return
	}

	// Setup server config
	config := sshh.Config{
		Deadline:          time.Second,
		Logger:            sshLogger,
		Bind:              ":9022",
		PrivateKey:        privateKey,
		PublicKeyCallback: pubKeyCallback,
		AuthLogCallback: func(meta ssh.ConnMetadata, method string, err error) {
			if err == nil {
				sshLogger.Info("login success", "user", meta.User())
			} else if err != nil && method == "publickey" {
				sshLogger.Info("login failure", "user", meta.User(), "err", err.Error())
			}
		},
		Handlers: map[string]sshh.SSHHandler{
			"kappa-client": &EchoHandler{},
		},
	}

	// Create SSH server
	sshServer, err := sshh.NewSSHServer(&config)
	if err != nil {
		logger.Error("SSH Server could not be configured", "error", err.Error())
		return
	}

	return &Server{
		sshServer: &sshServer,
	}, nil
}

type Server struct {
	sshServer *sshh.SSHServer
}

func (s *Server) Start() {
	s.sshServer.Start()
}

func (s *Server) Stop() {
	s.sshServer.Stop()
}
