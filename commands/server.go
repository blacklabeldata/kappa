package commands

import (
	"os"
	"os/signal"
	"time"

	"github.com/blacklabeldata/kappa/server"
	log "github.com/mgutz/logxi/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

		// Initialize config
		err := InitializeConfig(writer)
		if err != nil {
			return
		}

		// Create server config
		cfg := server.DatabaseConfig{
			LogOutput:             writer,
			NodeName:              viper.GetString("NodeName"),
			ClusterName:           viper.GetString("ClusterName"),
			Bootstrap:             viper.GetBool("Bootstrap"),
			BootstrapExpect:       viper.GetInt("BootstrapExpect"),
			AdminCertificateFile:  viper.GetString("AdminCert"),
			CACertificateFile:     viper.GetString("CACert"),
			DataPath:              viper.GetString("DataPath"),
			SSHBindAddress:        viper.GetString("SSHListen"),
			SSHPrivateKeyFile:     viper.GetString("SSHKey"),
			SSHConnectionDeadline: time.Second,
		}

		// Create server
		svr, err := server.NewServer(&cfg)
		if err != nil {
			logger.Error("Failed to start server:", "error", err)
			return
		}

		// Start server
		svr.Start()

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
		// sshServer.Stop()
		svr.Stop()
	},
}

// Pointer to ServerCmd used in initialization
var serverCmd *cobra.Command

// Command line args
var (
	SSHKey          string
	AdminCert       string
	CACert          string
	TLSCert         string
	TLSKey          string
	DataPath        string
	SSHListen       string
	HTTPListen      string
	NodeName        string
	ClusterName     string
	Bootstrap       bool
	BootstrapExpect int
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
	ServerCmd.PersistentFlags().StringVarP(&NodeName, "node-name", "", "", "Node name")
	ServerCmd.PersistentFlags().StringVarP(&ClusterName, "cluster", "", "", "Cluster name")
	ServerCmd.PersistentFlags().BoolVarP(&Bootstrap, "bootstrap", "", false, "Bootstrap node")
	ServerCmd.PersistentFlags().IntVarP(&BootstrapExpect, "bootstrap-expect", "", 0, "Bootstrap node")
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
	viper.SetDefault("NodeName", "kappa-server")
	viper.SetDefault("ClusterName", "kappa")
	viper.SetDefault("Bootstrap", false)
	viper.SetDefault("BootstrapExpect", 0)

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
	if serverCmd.PersistentFlags().Lookup("node-name").Changed {
		logger.Info("", "NodeName", NodeName)
		viper.Set("NodeName", NodeName)
	}
	if serverCmd.PersistentFlags().Lookup("cluster").Changed {
		logger.Info("", "ClusterName", ClusterName)
		viper.Set("ClusterName", ClusterName)
	}
	if serverCmd.PersistentFlags().Lookup("bootstrap").Changed {
		logger.Info("", "Bootstrap", Bootstrap)
		viper.Set("Bootstrap", Bootstrap)
	}
	if serverCmd.PersistentFlags().Lookup("bootstrap-expect").Changed {
		logger.Info("", "BootstrapExpect", BootstrapExpect)
		viper.Set("BootstrapExpect", BootstrapExpect)
	}

	return nil
}
