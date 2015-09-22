package commands

import (
	"os"
	"os/signal"
	"strings"
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
			ExistingNodes:         strings.Split(viper.GetString("ClusterNodes"), ","),
			Bootstrap:             viper.GetBool("Bootstrap"),
			BootstrapExpect:       viper.GetInt("BootstrapExpect"),
			AdminCertificateFile:  viper.GetString("AdminCert"),
			CACertificateFile:     viper.GetString("CACert"),
			DataPath:              viper.GetString("DataPath"),
			SSHBindAddress:        viper.GetString("SSHListen"),
			SSHPrivateKeyFile:     viper.GetString("SSHKey"),
			SSHConnectionDeadline: time.Second,
			GossipBindAddr:        viper.GetString("GossipBindAddr"),
			GossipBindPort:        viper.GetInt("GossipBindPort"),
			GossipAdvertiseAddr:   viper.GetString("GossipAdvertiseAddr"),
			GossipAdvertisePort:   viper.GetInt("GossipAdvertisePort"),
		}

		// Create server
		svr, err := server.NewServer(&cfg)
		if err != nil {
			logger.Error("Failed to start server:", "error", err)
			return
		}

		// Start server
		if err := svr.Start(); err != nil {
			svr.Stop()
			return
		}

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
	SSHKey              string
	AdminCert           string
	CACert              string
	TLSCert             string
	TLSKey              string
	DataPath            string
	SSHListen           string
	HTTPListen          string
	NodeName            string
	ClusterName         string
	ClusterNodes        string
	Bootstrap           bool
	BootstrapExpect     int
	GossipBindAddr      string
	GossipBindPort      int
	GossipAdvertiseAddr string
	GossipAdvertisePort int
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

	// Serf
	ServerCmd.PersistentFlags().StringVarP(&NodeName, "node-name", "", "", "Node name")
	ServerCmd.PersistentFlags().StringVarP(&ClusterName, "cluster", "", "", "Cluster name")
	ServerCmd.PersistentFlags().StringVarP(&ClusterNodes, "nodes", "", "", "Comma delimited list of IPs or domains")
	ServerCmd.PersistentFlags().BoolVarP(&Bootstrap, "bootstrap", "", false, "Bootstrap node")
	ServerCmd.PersistentFlags().IntVarP(&BootstrapExpect, "bootstrap-expect", "", 0, "Bootstrap node")

	// Memberlist
	ServerCmd.PersistentFlags().StringVarP(&GossipBindAddr, "gossip-bind-addr", "", "", "Address for gossip")
	ServerCmd.PersistentFlags().IntVarP(&GossipBindPort, "gossip-bind-port", "", 7946, "Port for gossip")
	ServerCmd.PersistentFlags().StringVarP(&GossipAdvertiseAddr, "gossip-advert-addr", "", "", "Address to advertise gossip")
	ServerCmd.PersistentFlags().IntVarP(&GossipAdvertisePort, "gossip-advert-port", "", 7946, "Port to advertise gossip")
	serverCmd = ServerCmd
}

// InitializeServerConfig sets up the config options for the database servers.
func InitializeServerConfig(logger log.Logger) error {

	// Load default settings
	logger.Info("Loading default server settings")

	// CACert sets the certificate authority
	viper.SetDefault("CACert", "ca.crt")
	viper.BindEnv("CACert", "KAPPA_CA_CERT")

	// AdminCert sets the admin certificate
	viper.SetDefault("AdminCert", "admin.crt")
	viper.BindEnv("AdminCert", "KAPPA_ADMIN_CERT")

	// SSHKey sets the private key for the SSH server
	viper.SetDefault("SSHKey", "ssh-identity.key")
	viper.BindEnv("SSHKey", "KAPPA_SSH_KEY")

	// TLSCert sets the certificate for HTTPS
	viper.SetDefault("TLSCert", "tls-identity.crt")
	viper.BindEnv("TLSCert", "KAPPA_TLS_CERT")

	// TLSKey sets the private key for HTTPS
	viper.SetDefault("TLSKey", "tls-identity.key")
	viper.BindEnv("TLSKey", "KAPPA_TLS_KEY")

	// DataPath sets the directory for data storage
	viper.SetDefault("DataPath", "./data")
	viper.BindEnv("DataPath", "KAPPA_DATA_PATH")

	// SSHListen sets the address to listen for SSH traffic
	viper.SetDefault("SSHListen", ":9022")
	viper.BindEnv("SSHListen", "KAPPA_SSH_LISTEN")

	// HTTPListen sets the address to listen for HTTP traffic
	viper.SetDefault("HTTPListen", ":19022")
	viper.BindEnv("HTTPListen", "KAPPA_HTTP_LISTEN")

	// Serf config
	// ClusterNodes is a list of existing cluster nodes
	viper.SetDefault("ClusterNodes", "")
	viper.BindEnv("ClusterNodes", "KAPPA_CLUSTER_NODES")

	// NodeName sets the server's name
	viper.SetDefault("NodeName", "kappa-server")
	viper.BindEnv("NodeName", "KAPPA_NODE_NAME")

	// ClusterName sets the cluster name of this node.
	viper.SetDefault("ClusterName", "kappa")
	viper.BindEnv("ClusterName", "KAPPA_CLUSTER_NAME")

	// Bootstrap sets whether to bootstrap this node.
	viper.SetDefault("Bootstrap", false)
	viper.BindEnv("Bootstrap", "KAPPA_BOOTSTRAP")

	// BootstrapExpect is an argument used by Serf.
	viper.SetDefault("BootstrapExpect", 0)

	// Memberlist config
	// GossipBindAddr sets the Addr for cluster gossip.
	viper.SetDefault("GossipBindAddr", "0.0.0.0")
	viper.BindEnv("GossipBindAddr", "KAPPA_GOSSIP_BIND_ADDR")

	// GossipBindPort sets the port for cluster gossip. The port is used for both UDP and TCP gossip.
	viper.SetDefault("GossipBindPort", 7946)
	viper.BindEnv("GossipBindPort", "KAPPA_GOSSIP_BIND_PORT")

	// GossipAdvertiseAddr sets what address to advertise to other
	// cluster members. Used for nat traversal.
	viper.SetDefault("GossipAdvertiseAddr", "")
	viper.BindEnv("GossipAdvertiseAddr", "KAPPA_GOSSIP_ADVERTISE_ADDR")

	// GossipAdvertisePort sets the port for cluster gossip and can
	// be useful for NAT traversal.
	viper.SetDefault("GossipAdvertisePort", 7946)
	viper.BindEnv("GossipAdvertisePort", "KAPPA_GOSSIP_ADVERTISE_PORT")

	// Set viper flags
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

	// Serf config
	if serverCmd.PersistentFlags().Lookup("nodes").Changed {
		logger.Info("", "ClusterNodes", ClusterNodes)
		viper.Set("ClusterNodes", ClusterNodes)
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

	// Memberlist Config
	if serverCmd.PersistentFlags().Lookup("gossip-bind-addr").Changed {
		logger.Info("", "GossipBindAddr", GossipBindAddr)
		viper.Set("GossipBindAddr", GossipBindAddr)
	}
	if serverCmd.PersistentFlags().Lookup("gossip-bind-port").Changed {
		logger.Info("", "GossipBindPort", GossipBindPort)
		viper.Set("GossipBindPort", GossipBindPort)
	}
	if serverCmd.PersistentFlags().Lookup("gossip-advert-addr").Changed {
		logger.Info("", "GossipAdvertiseAddr", GossipAdvertiseAddr)
		viper.Set("GossipAdvertiseAddr", GossipAdvertiseAddr)
	}
	if serverCmd.PersistentFlags().Lookup("gossip-advert-port").Changed {
		logger.Info("", "GossipAdvertisePort", GossipAdvertisePort)
		viper.Set("GossipAdvertisePort", GossipAdvertisePort)
	}

	return nil
}
