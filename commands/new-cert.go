package commands

import (
    "crypto/rand"
    "crypto/rsa"
    "fmt"
    "os"
    "path"
    "strings"

    log "github.com/mgutz/logxi/v1"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"

    "github.com/blacklabeldata/kappa/auth"
)

// NewCertCmd is the kappa root command.
var NewCertCmd = &cobra.Command{
    Use:   "new-cert",
    Short: "new-cert creates a new certificate",
    Long:  ``,
    Run: func(cmd *cobra.Command, args []string) {

        // Create logger
        writer := log.NewConcurrentWriter(os.Stdout)
        logger := log.NewLogger(writer, "new-cert")

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
        reqFile := path.Join(pki, "reqs", viper.GetString("Name")+".req")
        privFile := path.Join(pki, "private", viper.GetString("Name")+".key")
        crtFile := path.Join(pki, "public", viper.GetString("Name")+".crt")

        // Verify it is ok to delete files if they exist
        if !viper.GetBool("ForceOverwrite") {
            var files []string
            for _, filename := range []string{reqFile, privFile, crtFile} {
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

        // Create Certificate request
        csr, req, err := auth.CreateCertificateRequest(logger, privatekey,
            viper.GetString("Name"), viper.GetString("Organization"),
            viper.GetString("Country"), viper.GetString("Hosts"))
        if err != nil {
            logger.Warn("Error creating CA", "err", err.Error())
            return
        }

        // Create Certificate
        crt, err := auth.CreateCertificate(logger, csr, privatekey,
            viper.GetInt("Years"), viper.GetString("Hosts"))
        if err != nil {
            logger.Warn("Error creating certificate", "err", err.Error())
            return
        }

        // Save cert request
        auth.SaveCertificateRequest(logger, req, reqFile)

        // Save private key
        auth.SavePrivateKey(logger, privatekey, privFile)

        // Save certificate
        auth.SaveCertificate(logger, crt, crtFile)
    },
}

// Pointer to NewCertCmd used in initialization
var newCertCmd *cobra.Command

// Command line args
var (
    Name           string
    ForceOverwrite bool
)

func init() {

    NewCertCmd.PersistentFlags().IntVarP(&KeyBits, "bits", "", 4096, "Number of bits in key")
    NewCertCmd.PersistentFlags().StringVarP(&Hosts, "hosts", "", "127.0.0.1", "IP of cert")
    NewCertCmd.PersistentFlags().IntVarP(&Years, "years", "", 10, "Number of years until the certificate expires")
    NewCertCmd.PersistentFlags().StringVarP(&Organization, "organization", "", "kappa-ca", "Organization for CA")
    NewCertCmd.PersistentFlags().StringVarP(&Country, "country", "", "USA", "Country of origin for CA")
    NewCertCmd.PersistentFlags().StringVarP(&Name, "name", "", "localhost", "Name of certificate")
    NewCertCmd.PersistentFlags().BoolVarP(&ForceOverwrite, "overwrite", "", false, "Overwrite replaces existing certs")
    newCertCmd = NewCertCmd
}

// InitializeNewCertConfig sets up the command line options for creating a new certificate
func InitializeNewCertConfig(logger log.Logger) error {
    viper.SetDefault("Name", "localhost")
    viper.SetDefault("ForceOverwrite", "false")

    if newCertCmd.PersistentFlags().Lookup("name").Changed {
        logger.Info("", "Name", Name)
        viper.Set("Name", Name)
    }
    if newCertCmd.PersistentFlags().Lookup("overwrite").Changed {
        logger.Info("", "ForceOverwrite", ForceOverwrite)
        viper.Set("ForceOverwrite", ForceOverwrite)
    }

    return nil
}
