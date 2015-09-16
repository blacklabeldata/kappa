package commands

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/blacklabeldata/kappa/auth"
	log "github.com/mgutz/logxi/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InitCACmd is the kappa root command.
var InitCACmd = &cobra.Command{
	Use:   "init-ca",
	Short: "init-ca creates a new certificate authority",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		// Create logger
		writer := log.NewConcurrentWriter(os.Stdout)
		logger := log.NewLogger(writer, "init-ca")

		err := InitializeConfig(writer)
		if err != nil {
			return
		}

		// Setup directory structure
		if err := auth.CreatePkiDirectories(logger, "."); err != nil {
			return
		}

		// Create file paths
		pki := path.Join(".", "pki")
		crtFile := path.Join(pki, "ca.crt")
		privFile := path.Join(pki, "private", "ca.key")

		// Verify it is ok to delete files if they exist
		if !viper.GetBool("ForceOverwrite") {
			var files []string
			for _, filename := range []string{privFile, crtFile} {
				if _, err := os.Stat(filename); err == nil {
					files = append(files, filename)
				}
			}

			if len(files) > 0 {
				var input string
				fmt.Println("This operation will overwrite these existing files:")
				for _, file := range files {
					fmt.Println("\t", file)
				}
				fmt.Print("Are you sure you want to overwrite these files (yN)? ")
				fmt.Scanln(&input)

				if !strings.Contains(strings.ToLower(input), "y") {
					fmt.Println("New certificate was not created.")
					return
				}
			}
		}

		// generate private key
		privatekey, err := rsa.GenerateKey(rand.Reader, viper.GetInt("Bits"))
		if err != nil {
			logger.Warn("Error generating private key")
			return
		}

		// Create CA
		cert, err := auth.CreateCertificateAuthority(logger, privatekey,
			viper.GetInt("Years"), viper.GetString("Organization"),
			viper.GetString("Country"), viper.GetString("Hosts"))
		if err != nil {
			logger.Warn("Error creating CA", "err", err.Error())
			return
		}

		// Save cert
		auth.SaveCertificate(logger, cert, crtFile)

		// Save private key
		auth.SavePrivateKey(logger, privatekey, privFile)

	},
}

// Pointer to InitCACmd used in initialization
var initCmd *cobra.Command

// Command line args
var (
	KeyBits      int
	Years        int
	Organization string
	Country      string
	Hosts        string
)

func init() {

	InitCACmd.PersistentFlags().IntVarP(&KeyBits, "bits", "", 4096, "Number of bits in key")
	InitCACmd.PersistentFlags().IntVarP(&Years, "years", "", 10, "Number of years until the CA certificate expires")
	InitCACmd.PersistentFlags().StringVarP(&Organization, "organization", "", "kappa-ca", "Organization for CA")
	InitCACmd.PersistentFlags().StringVarP(&Country, "country", "", "USA", "Country of origin for CA")
	InitCACmd.PersistentFlags().StringVarP(&Hosts, "hosts", "", "127.0.0.1", "Comma delimited list of IPs or domains")
	InitCACmd.PersistentFlags().BoolVarP(&ForceOverwrite, "overwrite", "", false, "Overwrite replaces existing certs")
	initCmd = InitCACmd
}

// InitializeCertAuthConfig sets up the command line options for creating a CA
func InitializeCertAuthConfig(logger log.Logger) error {

	viper.SetDefault("Bits", "4096")
	viper.SetDefault("Years", "10")
	viper.SetDefault("Organization", "kappa-ca")
	viper.SetDefault("Country", "USA")

	if initCmd.PersistentFlags().Lookup("bits").Changed {
		logger.Info("", "Bits", KeyBits)
		viper.Set("Bits", KeyBits)
	}
	if initCmd.PersistentFlags().Lookup("years").Changed {
		logger.Info("", "Years", Years)
		viper.Set("Years", Years)
	}
	if initCmd.PersistentFlags().Lookup("organization").Changed {
		logger.Info("", "Organization", Organization)
		viper.Set("Organization", Organization)
	}
	if initCmd.PersistentFlags().Lookup("country").Changed {
		logger.Info("", "Country", Country)
		viper.Set("Country", Country)
	}
	if initCmd.PersistentFlags().Lookup("hosts").Changed {
		logger.Info("", "Hosts", Hosts)
		viper.Set("Hosts", Hosts)
	}

	return nil
}
