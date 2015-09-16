package commands

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/user"
	"path"
	"strings"

	log "github.com/mgutz/logxi/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cli "github.com/blacklabeldata/kappa/client"
	"github.com/blacklabeldata/kappa/common"
	"github.com/blacklabeldata/kappa/skl"
	"github.com/blacklabeldata/xbinary"
	"golang.org/x/crypto/ssh"
	// "golang.org/x/crypto/ssh/terminal"
	"github.com/subsilent/crypto/ssh/terminal"
)

// ClientCmd is the CLI command
var ClientCmd = &cobra.Command{
	Use:   "client [ssh://username@host:port]",
	Short: "client starts a terminal with the given kappa server",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		// Create logger
		writer := log.NewConcurrentWriter(os.Stdout)
		logger := log.NewLogger(writer, "cli")

		err := InitializeClientConfig(logger)
		if err != nil {
			return
		}

		// Get SSH Key file
		keyFile := viper.GetString("ClientKey")
		// logger.Info("Reading private key", "file", sshKeyFile)

		// Read SSH Key
		keyBytes, err := ioutil.ReadFile(keyFile)
		if err != nil {
			fmt.Println("Private key could not be read:", err.Error())
			fmt.Println(cmd.Help())
			return
		}

		// Get private key
		privateKey, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			fmt.Println("Private key is invalid:", err.Error())
			fmt.Println(cmd.Help())
			return
		}

		// get username and host
		if len(args) < 1 {
			fmt.Println("Missing server URL")
			fmt.Println(cmd.Help())
			return
		}

		u, err := url.Parse(args[0])
		if err != nil {
			fmt.Println("Error parsing host - expected format ssh://username@host:port : ", err.Error())
			fmt.Println(cmd.Help())
			return
		} else if u.User == nil {
			fmt.Println("Missing username - expected format: ssh://username@host:port")
			fmt.Println(cmd.Help())
			return
		} else if u.User.Username() == "" {
			fmt.Println("Missing username - expected format: ssh://username@host:port")
			fmt.Println(cmd.Help())
			return
		}

		// Configure client connection
		config := &ssh.ClientConfig{
			User: u.User.Username(),
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(privateKey),
			},
		}

		// Create client connection
		client, err := ssh.Dial("tcp", u.Host, config)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer client.Close()

		// Open channel
		channel, requests, err := client.OpenChannel("kappa-client", []byte{})
		go ssh.DiscardRequests(requests)

		// Read history
		var entries []string
		usr, err := user.Current()
		historyPath := path.Join(usr.HomeDir, ".kappa_history")
		historyBytes, err := ioutil.ReadFile(historyPath)
		if err == nil {
			tmpEntries := strings.Split(string(historyBytes), "\n")
			for _, e := range tmpEntries {
				if len(e) > 0 {
					entries = append(entries, e)
				}
			}
		}

		// Create history manager
		history := History{entries, make([]string, 0)}

		// Create terminal
		term := terminal.NewTerminal(os.Stdin, string(common.DefaultColorCodes.LightBlue)+"kappa > "+string(common.DefaultColorCodes.Reset))
		term.LoadInitialHistory(entries)

		// Try to make the terminal raw
		oldState, err := terminal.MakeRaw(0)
		if err != nil {
			logger.Warn("Error making terminal raw: ", err.Error())
		}
		defer terminal.Restore(0, oldState)

		// Write ascii text
		term.Write([]byte("\r\n"))
		for _, line := range common.ASCII {
			term.Write([]byte(line))
			term.Write([]byte("\r\n"))
		}

		// Write login message
		term.Write([]byte("\r\n\n"))
		cli.GetMessage(term, common.DefaultColorCodes)
		term.Write([]byte("\n"))

		// Start REPL
		for {
			input, err := term.ReadLine()

			if err != nil {
				break
			}

			// Process line
			line := strings.TrimSpace(string(input))
			if len(line) > 0 {

				// Log input and handle exit requests
				if line == "exit" || line == "quit" {
					break
				} else if line == "history" {
					for _, e := range history.oldEntries {
						term.Write([]byte(" " + e + "\r\n"))
					}
					for _, e := range history.newEntries {
						term.Write([]byte(" " + e + "\r\n"))
					}
					continue
				} else if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "--") {

					term.Write(common.DefaultColorCodes.LightGrey)
					term.Write([]byte(line + "\r\n"))
					term.Write(common.DefaultColorCodes.Reset)
					continue
				}

				// Parse statement
				stmt, err := skl.ParseStatement(line)

				// Return parse error in red
				if err != nil {
					term.Write(common.DefaultColorCodes.LightRed)
					term.Write([]byte(" " + err.Error()))
					term.Write([]byte("\r\n"))
					term.Write(common.DefaultColorCodes.Reset)
					term.RemoveLastLine()
					continue
				}

				w := common.ResponseWriter{common.DefaultColorCodes, term}

				length := make([]byte, 4)
				xbinary.LittleEndian.PutInt32(length, 0, int32(len(line)))

				// Write data
				channel.Write(length)
				channel.Write([]byte(line))

				// term.Write(common.DefaultColorCodes.LightBlue)
				// term.Write([]byte(fmt.Sprintf("%d : '%s'\r\n", length, line)))
				// term.Write(common.DefaultColorCodes.Reset)

				channel.Read(length)

				size, err := xbinary.LittleEndian.Int32(length, 0)
				data := make([]byte, int(size))
				channel.Read(data)

				// term.Write(common.DefaultColorCodes.Green)
				// term.Write([]byte(fmt.Sprintf("%d : '%s'\r\n", length, string(data))))
				// term.Write(common.DefaultColorCodes.Reset)
				// if err != nil {
				// 	w.Fail(common.InternalServerError, err.Error())
				// }
				// channel.SendRequest("skl", true, []byte(line))

				// Execute statements
				// w.Success(common.OK, string(data))
				w.Success(common.OK, stmt.String())
				// executor.Execute(&w, stmt)

				// Write line to history file
				// historyFile.WriteString(line + "\n")
				history.Append(line + "\n")
			}
		}

		history.WriteToFile(historyPath)
		os.Stdout.Write(common.DefaultColorCodes.LightGreen)
		os.Stdout.Write([]byte("\r\n"))
		os.Stdout.Write([]byte(" Yo homes, smell you later!"))
		os.Stdout.Write([]byte("\r\n"))
		os.Stdout.Write(common.DefaultColorCodes.Reset)
	},
}

type History struct {
	oldEntries []string
	newEntries []string
}

func (h *History) Append(line string) {
	var lastEntry string
	if len(h.newEntries) > 0 {
		lastEntry = h.newEntries[len(h.newEntries)-1]
	} else if len(h.oldEntries) > 0 {
		lastEntry = h.oldEntries[len(h.oldEntries)-1]
	}

	if line != lastEntry {
		h.newEntries = append(h.newEntries, line)
	}
}

func (h *History) WriteToFile(filepath string) error {

	// Open history file for writing
	historyFile, err := os.OpenFile(filepath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer historyFile.Close()

	// Write lines
	_, err = historyFile.WriteString(strings.Join(h.newEntries, "\n"))
	return err
}

// Pointer to ClientCmd used in initialization
var clientCmd *cobra.Command

// Command line args
var (
	ClientKey string
)

func init() {
	ClientCmd.PersistentFlags().StringVarP(&ClientKey, "identity-file", "i", "", "Private key to identify client")
	clientCmd = ClientCmd
}

// InitializeClientConfig sets up the config options for the database servers.
func InitializeClientConfig(logger log.Logger) error {

	// Load default settings
	// logger.Info("Loading default server settings")

	// User @ Host
	// SSH private key
	// kappa cli -i pki/private/admin.key ssh://admin@127.0.0.1:9022
	// kappa cli -i pki/private/admin.key -u admin -H 127.0.0.1:9022

	viper.SetDefault("ClientKey", "~/.ssh/id_rsa")

	if clientCmd.PersistentFlags().Lookup("identity-file").Changed {
		viper.Set("ClientKey", ClientKey)
	}
	return nil
}
