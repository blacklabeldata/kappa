package handlers

import (
	"fmt"
	"strings"

	log "github.com/mgutz/logxi/v1"

	"github.com/blacklabeldata/kappa/client"
	"github.com/blacklabeldata/kappa/common"
	"github.com/blacklabeldata/kappa/datamodel"
	"github.com/blacklabeldata/kappa/executor"
	"github.com/blacklabeldata/kappa/skl"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	tomb "gopkg.in/tomb.v2"
)

func NewShellHandler(logger log.Logger) SSHHandler {
	return &shellHandler{}
}

type shellHandler struct {
	logger log.Logger
	system datamodel.System
}

func (s *shellHandler) Handle(parentTomb tomb.Tomb, sshConn *ssh.ServerConn, channel ssh.Channel, requests <-chan *ssh.Request) error {
	defer channel.Close()

	users, err := s.system.Users()
	if err != nil {
		return err
	}

	user, err := users.Get(sshConn.Permissions.Extensions["username"])
	if err != nil {
		return err
	}

	// Create tomb for terminal goroutines
	var t tomb.Tomb

	// Sessions have out-of-band requests such as "shell",
	// "pty-req" and "env".  Here we handle only the
	// "shell" request.
	for {
		select {
		case <-parentTomb.Dying():
			t.Kill(nil)
			return t.Wait()
		case req := <-requests:

			ok := false
			switch req.Type {
			case "shell":
				ok = true

				if len(req.Payload) > 0 {
					fmt.Println(string(req.Payload))
					// We don't accept any
					// commands, only the
					// default shell.
					ok = false
				}

			case "pty-req":
				// Responding 'ok' here will let the client
				// know we have a pty ready for input
				ok = true

				go s.startTerminal(t, channel, s.system, user)
			default:
				// fmt.Println("default req: ", req)
			}

			req.Reply(ok, nil)
		}
	}
	return nil
}

func (s *shellHandler) startTerminal(parentTomb tomb.Tomb, channel ssh.Channel, system datamodel.System, user datamodel.User) {
	defer channel.Close()

	prompt := "kappa> "
	term := terminal.NewTerminal(channel, prompt)

	// // Try to make the terminal raw
	// oldState, err := terminal.MakeRaw(0)
	// if err != nil {
	//     logger.Warn("Error making terminal raw: ", err.Error())
	// }
	// defer terminal.Restore(0, oldState)

	// Write ascii text
	term.Write([]byte("\r\n"))
	for _, line := range common.ASCII {
		term.Write([]byte(line))
		term.Write([]byte("\r\n"))
	}

	// Write login message
	term.Write([]byte("\r\n\n"))
	client.GetMessage(channel, common.DefaultColorCodes)
	term.Write([]byte("\n"))

	// Create query executor
	executor := executor.NewExecutor(executor.NewSession("", user), common.NewTerminal(term, prompt), system)

	// Start REPL
	for {

		select {
		case <-parentTomb.Dying():
			return
		default:
			input, err := term.ReadLine()
			if err != nil {
				fmt.Errorf("Readline() error")
				break
			}

			// Process line
			line := strings.TrimSpace(input)
			if len(line) > 0 {

				// Log input and handle exit requests
				if line == "exit" || line == "quit" {
					s.logger.Info("Closing connection")
					break
				} else if line == "quote me" {
					term.Write([]byte("\r\n"))
					client.GetMessage(channel, common.DefaultColorCodes)
					term.Write([]byte("\r\n"))
					continue
				} else if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "--") {

					channel.Write(common.DefaultColorCodes.LightGrey)
					channel.Write([]byte(line + "\r\n"))
					channel.Write(common.DefaultColorCodes.Reset)
					continue
				}

				// Parse statement
				stmt, err := skl.ParseStatement(line)

				// Return parse error in red
				if err != nil {
					s.logger.Warn("Bad Statement", "statement", line, "error", err)
					channel.Write(common.DefaultColorCodes.LightRed)
					channel.Write([]byte(err.Error()))
					channel.Write([]byte("\r\n"))
					channel.Write(common.DefaultColorCodes.Reset)
					continue
				}

				// Execute statements
				w := common.ResponseWriter{common.DefaultColorCodes, channel}
				executor.Execute(&w, stmt)
			}
		}
	}
}
