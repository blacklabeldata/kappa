package commands

import (
	"fmt"
	"io"

	log "github.com/mgutz/logxi/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// KappaCmd is the root root command.
var KappaCmd = &cobra.Command{
	Use:   "kappa",
	Short: "Kappa is a NoSQL database centered around replicated logs and views.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cmd.Help())
	},
}

// Pointer to KappaCmd used in initialization
var kappaCmd *cobra.Command

// Execute is the main entry point into the server.
func Execute() {
	AddCommands()
	KappaCmd.Execute()
}

// AddCommands add all of the subcommands to the main entry point.
func AddCommands() {
	KappaCmd.AddCommand(ServerCmd)
	KappaCmd.AddCommand(InitCACmd)
	KappaCmd.AddCommand(NewCertCmd)
	KappaCmd.AddCommand(ClientCmd)
}

// Command line args
var (
	ConfigPath string
)

func init() {
	KappaCmd.PersistentFlags().StringVarP(&ConfigPath, "config", "c", "./", "Configuration file")
	kappaCmd = KappaCmd

	// for Bash auto-complete
	validConfigFilenames := []string{"json", "js", "yaml", "yml", "toml", "tml"}
	annotation := make(map[string][]string)
	annotation[cobra.BashCompFilenameExt] = validConfigFilenames
	KappaCmd.PersistentFlags().Lookup("config").Annotations = annotation
}

// InitializeMainConfig sets up the config options for the kappa command
func InitializeMainConfig(logger log.Logger) error {
	if kappaCmd.PersistentFlags().Lookup("config").Changed {
		logger.Info("", "ConfigPath", ConfigPath)
		viper.Set("ConfigPath", ConfigPath)
	}

	// Setup config file support
	viper.SetConfigName("config")
	viper.AddConfigPath(ConfigPath)

	// Read configuration file
	logger.Info("Reading configuration file")
	err := viper.ReadInConfig()
	if err != nil {
		logger.Warn("Unable to locate configuration file.", err)
	}

	return nil
}

// InitializeConfig reads the configuration file and sets up the application settings via Viper.
func InitializeConfig(writer io.Writer) error {
	logger := log.NewLogger(writer, "config")

	if err := InitializeMainConfig(logger); err != nil {
		logger.Warn("Failed to initialize kappa command line flags")
		return err
	}

	if err := InitializeServerConfig(logger); err != nil {
		logger.Warn("Failed to initialize server command line flags")
		return err
	}

	if err := InitializeCertAuthConfig(logger); err != nil {
		logger.Warn("Failed to initialize init-ca command line flags")
		return err
	}

	if err := InitializeNewCertConfig(logger); err != nil {
		logger.Warn("Failed to initialize new-cert command line flags")
		return err
	}

	return nil
}
