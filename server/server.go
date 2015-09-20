package server

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/blacklabeldata/kappa/auth"
	"github.com/blacklabeldata/kappa/datamodel"
	"github.com/blacklabeldata/kappa/pkg/uuid"
	"github.com/blacklabeldata/sshh"
	"github.com/hashicorp/serf/serf"
	log "github.com/mgutz/logxi/v1"
	"golang.org/x/crypto/ssh"
	tomb "gopkg.in/tomb.v2"
)

const serfSnapshot = "serf/local.snapshot"

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
		Deadline:          c.SSHConnectionDeadline,
		Logger:            sshLogger,
		Bind:              c.SSHBindAddress,
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

	// Create database server
	s := &Server{
		config:       c,
		logger:       logger,
		sshServer:    &sshServer,
		serfEventCh:  make(chan serf.Event, 256),
		kappaEventCh: make(chan serf.UserEvent, 256),
		reconcileCh:  make(chan serf.Member, 32),
	}

	// Create serf server
	s.serf, err = s.setupSerf()
	if err != nil {
		err = fmt.Errorf("Failed to start serf: %v", err)
		return
	}

	return s, nil
}

type Server struct {
	config    *DatabaseConfig
	logger    log.Logger
	sshServer *sshh.SSHServer

	// localKappas is used to track the known kappas
	// in the cluster. Used to do leader forwarding.
	localKappas map[string]*NodeDetails
	localLock   sync.RWMutex

	serf         *serf.Serf
	serfEventCh  chan serf.Event
	kappaEventCh chan serf.UserEvent
	reconcileCh  chan serf.Member
	t            tomb.Tomb
}

func (s *Server) Start() {
	s.sshServer.Start()

	// Start serf handler
	s.t.Go(s.serfEventHandler)
}

func (s *Server) Stop() {
	s.sshServer.Stop()

	// Kill Serf handler
	s.t.Kill(nil)
	s.logger.Info("Shutting down Serf server...")
	s.t.Wait()
}

func (s *Server) setupSerf() (*serf.Serf, error) {
	conf := serf.DefaultConfig()

	// Generate NodeName if missing
	id, err := uuid.UUID4()
	if err != nil {
		return nil, err
	}

	// Get SSH server port
	port := s.sshServer.Addr.Port

	// Initialize serf
	conf.Init()

	conf.NodeName = s.config.NodeName
	conf.MemberlistConfig.BindAddr = s.config.GossipBindAddr
	conf.MemberlistConfig.BindPort = s.config.GossipBindPort
	conf.MemberlistConfig.AdvertiseAddr = s.config.GossipAdvertiseAddr
	conf.MemberlistConfig.AdvertisePort = s.config.GossipAdvertisePort

	conf.Tags["id"] = id
	conf.Tags["role"] = "kappa"
	conf.Tags["cluster"] = s.config.ClusterName
	conf.Tags["build"] = s.config.Build
	conf.Tags["port"] = fmt.Sprintf("%d", port)
	if s.config.Bootstrap {
		conf.Tags["bootstrap"] = "1"
	}
	if s.config.BootstrapExpect != 0 {
		conf.Tags["expect"] = fmt.Sprintf("%d", s.config.BootstrapExpect)
	}

	conf.MemberlistConfig.LogOutput = s.config.LogOutput
	conf.LogOutput = s.config.LogOutput
	conf.EventCh = s.serfEventCh
	conf.SnapshotPath = filepath.Join(s.config.DataPath, serfSnapshot)
	conf.ProtocolVersion = conf.ProtocolVersion
	conf.RejoinAfterLeave = true
	conf.EnableNameConflictResolution = false

	conf.Merge = &mergeDelegate{name: s.config.ClusterName}
	if err := ensurePath(conf.SnapshotPath, false); err != nil {
		return nil, err
	}
	return serf.Create(conf)
}

// Join is used to have Kappa join the cluster.
// The target address should be another node inside the cluster
// listening on the Serf address
func (s *Server) Join(addrs []string) (int, error) {
	return s.serf.Join(addrs, true)
}

// LocalMember is used to return the local node
func (c *Server) LocalMember() serf.Member {
	return c.serf.LocalMember()
}

// Members is used to return the members of the cluster
func (s *Server) Members() []serf.Member {
	return s.serf.Members()
}

// RemoveFailedNode is used to remove a failed node from the cluster
func (s *Server) RemoveFailedNode(node string) error {
	if err := s.serf.RemoveFailedNode(node); err != nil {
		return err
	}
	return nil
}

// IsLeader checks if this server is the cluster leader
func (s *Server) IsLeader() bool {
	return true
	// return s.raft.State() == raft.Leader
}

// KeyManager returns the Serf keyring manager
func (s *Server) KeyManager() *serf.KeyManager {
	return s.serf.KeyManager()
}

// Encrypted determines if gossip is encrypted
func (s *Server) Encrypted() bool {
	return s.serf.EncryptionEnabled()
}
