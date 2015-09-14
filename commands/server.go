package commands

import (
	"crypto/x509"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/blacklabeldata/kappa/auth"
	"github.com/blacklabeldata/kappa/datamodel"
	"github.com/blacklabeldata/sshh"
	log "github.com/mgutz/logxi/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

// ServerCmd is the kappa root command.
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "server starts the database server",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		// Create logger
		writer := log.NewConcurrentWriter(os.Stdout)
		logger := log.NewLogger(writer, "kappa")

		err := InitializeConfig(writer)
		if err != nil {
			return
		}

		// Create data directory
		if err := os.MkdirAll(viper.GetString("DataPath"), os.ModeDir|0655); err != nil {
			logger.Warn("Could not create data directory", "err", err.Error())
			return
		}

		// Connect to database
		cwd, err := os.Getwd()
		if err != nil {
			logger.Error("Could not get working directory", "error", err.Error())
			return
		}

		file := path.Join(cwd, viper.GetString("DataPath"), "meta.db")
		logger.Info("Connecting to database", "file", file)
		system, err := datamodel.NewSystem(file)
		if err != nil {
			logger.Error("Could not connect to database", "error", err.Error())
			return
		}

		// Get SSH Key file
		sshKeyFile := viper.GetString("SSHKey")
		logger.Info("Reading private key", "file", sshKeyFile)

		privateKey, err := auth.ReadPrivateKey(logger, sshKeyFile)
		if err != nil {
			return
		}

		// Get admin certificate
		adminCertFile := viper.GetString("AdminCert")
		logger.Info("Reading admin public key", "file", adminCertFile)

		// Read admin certificate
		cert, err := ioutil.ReadFile(adminCertFile)
		if err != nil {
			logger.Error("admin certificate could not be read", "filename", viper.GetString("AdminCert"))
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
		rootPem, err := ioutil.ReadFile(viper.GetString("CACert"))
		if err != nil {
			logger.Error("root certificate could not be read", "filename", viper.GetString("CACert"))
			return
		}

		// Create certificate pool
		roots := x509.NewCertPool()
		if ok := roots.AppendCertsFromPEM(rootPem); !ok {
			logger.Error("failed to parse root certificate")
			return
		}

		// Setup SSH Server
		sshLogger := log.NewLogger(writer, "ssh")
		pubKeyCallback, err := datamodel.PublicKeyCallback(system)
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
					sshLogger.Info("login failure", "user", meta.User(), "err", err)
				}
			},
		}

		// Create SSH server
		sshServer, err := sshh.NewSSHServer(&config)
		if err != nil {
			logger.Error("SSH Server could not be configured", "error", err.Error())
			return
		}

		// Start servers
		sshServer.Start()

		// Handle signals
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, os.Kill)

		// Wait for signal
		logger.Info("Ready to serve requests")

		// Block until signal is received
		<-sig

		// Stop listening for signals and close channel
		signal.Stop(sig)
		close(sig)

		// Shut down SSH server
		logger.Info("Shutting down servers.")
		sshServer.Stop()
	},
}

// Pointer to ServerCmd used in initialization
var serverCmd *cobra.Command

// Command line args
var (
	SSHKey     string
	AdminCert  string
	CACert     string
	TLSCert    string
	TLSKey     string
	DataPath   string
	SSHListen  string
	HTTPListen string
)

func init() {

	ServerCmd.PersistentFlags().StringVarP(&SSHKey, "ssh-key", "", "", "Private key to identify server with")
	ServerCmd.PersistentFlags().StringVarP(&AdminCert, "admin-cert", "", "", "Public certificate for admin user")
	ServerCmd.PersistentFlags().StringVarP(&CACert, "ca-cert", "", "", "Root Certificate")
	ServerCmd.PersistentFlags().StringVarP(&TLSCert, "tls-cert", "", "", "TLS certificate file")
	ServerCmd.PersistentFlags().StringVarP(&TLSKey, "tls-key", "", "", "TLS private key file")
	ServerCmd.PersistentFlags().StringVarP(&DataPath, "data", "D", "", "Data directory")
	ServerCmd.PersistentFlags().StringVarP(&SSHListen, "ssh-listen", "S", "", "Host and port for SSH server to listen on")
	ServerCmd.PersistentFlags().StringVarP(&HTTPListen, "http-listen", "H", ":", "Host and port for HTTP server to listen on")
	serverCmd = ServerCmd
}

// InitializeServerConfig sets up the config options for the database servers.
func InitializeServerConfig(logger log.Logger) error {

	// Load default settings
	logger.Info("Loading default server settings")
	viper.SetDefault("CACert", "ca.crt")
	viper.SetDefault("AdminCert", "admin.crt")
	viper.SetDefault("SSHKey", "ssh-identity.key")
	viper.SetDefault("TLSCert", "tls-identity.crt")
	viper.SetDefault("TLSKey", "tls-identity.key")
	viper.SetDefault("DataPath", "./data")
	viper.SetDefault("SSHListen", ":9022")
	viper.SetDefault("HTTPListen", ":19022")

	if serverCmd.PersistentFlags().Lookup("ca-cert").Changed {
		logger.Info("", "CACert", CACert)
		viper.Set("CACert", CACert)
	}
	if serverCmd.PersistentFlags().Lookup("admin-cert").Changed {
		logger.Info("", "AdminCert", AdminCert)
		viper.Set("AdminCert", AdminCert)
	}
	if serverCmd.PersistentFlags().Lookup("ssh-key").Changed {
		logger.Info("", "SSHKey", SSHKey)
		viper.Set("SSHKey", SSHKey)
	}
	if serverCmd.PersistentFlags().Lookup("tls-cert").Changed {
		logger.Info("", "TLSCert", TLSCert)
		viper.Set("TLSCert", TLSCert)
	}
	if serverCmd.PersistentFlags().Lookup("tls-key").Changed {
		logger.Info("", "TLSKey", TLSKey)
		viper.Set("TLSKey", TLSKey)
	}
	if serverCmd.PersistentFlags().Lookup("ssh-listen").Changed {
		logger.Info("", "SSHListen", SSHListen)
		viper.Set("SSHListen", SSHListen)
	}
	if serverCmd.PersistentFlags().Lookup("http-listen").Changed {
		logger.Info("", "HTTPListen", HTTPListen)
		viper.Set("HTTPListen", HTTPListen)
	}
	if serverCmd.PersistentFlags().Lookup("data").Changed {
		logger.Info("", "DataPath", DataPath)
		viper.Set("DataPath", DataPath)
	}

	return nil
}
