package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/blacklabeldata/kappa/datamodel"
	"github.com/blacklabeldata/xbinary"

	"golang.org/x/crypto/ssh"
	tomb "gopkg.in/tomb.v2"
)

// PublicKeyCallback returns a function to validate public keys for user login.
func PublicKeyCallback(sys datamodel.System) (func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error), error) {
	if sys == nil {
		return nil, errors.New("ssh server: System cannot be nil")
	}

	// Get user store
	users, err := sys.Users()
	if err != nil {
		return nil, fmt.Errorf("ssh server: user store: %s", err)
	}

	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (perm *ssh.Permissions, err error) {
		// fmt.Println(string(key.Marshal()))

		// Get user if exists, otherwise return error
		user, err := users.Get(conn.User())
		if err != nil {
			return
		}

		// Check keyring for public key
		if keyring := user.KeyRing(); !keyring.Contains(key.Marshal()) {
			err = fmt.Errorf("invalid public key")
			return
		}

		// Add pubkey and username to permissions
		perm = &ssh.Permissions{
			Extensions: map[string]string{
				"pubkey":   string(key.Marshal()),
				"username": conn.User(),
			},
		}
		return
	}, nil
}

type EchoHandler struct {
}

func (e *EchoHandler) Handle(parentTomb tomb.Tomb, sshConn *ssh.ServerConn, channel ssh.Channel, requests <-chan *ssh.Request) error {
	defer channel.Close()

	// Create tomb for terminal goroutines
	var t tomb.Tomb

	type msg struct {
		length uint32
		data   []byte
	}
	in := make(chan msg)
	defer close(in)

	// Sessions have out-of-band requests such as "shell",
	// "pty-req" and "env".  Here we handle only the
	// "shell" request.
	t.Go(func() error {
		var buffer bytes.Buffer

		// Read channel
		t.Go(func() error {

			length := make([]byte, 4)
			for {
				n, err := channel.Read(length)
				if err != nil {
					return err
				} else if n != 4 {
					return errors.New("Invalid message length")
				}

				// Decode length
				l, err := xbinary.LittleEndian.Uint32(length, 0)
				if err != nil {
					return err
				}

				// Read data
				n64, err := buffer.ReadFrom(io.LimitReader(channel, int64(l)))
				if err != nil {
					return err
				} else if n64 != int64(l) {
					return errors.New("error: reading message")
				}

				select {
				case <-parentTomb.Dying():
					return nil
				case <-t.Dying():
					return nil
				case in <- msg{l, buffer.Bytes()}:
				}
			}
		})

		length := make([]byte, 4)
	OUTER:
		for {
			select {
			case <-parentTomb.Dying():
				t.Kill(nil)
				break OUTER

			case m := <-in:
				if m.length == 0 {
					return nil
				}

				// Encode length
				_, err := xbinary.LittleEndian.PutUint32(length, 0, m.length)
				if err != nil {
					t.Kill(err)
					return nil
				}

				// Write echo response
				channel.Write(length)
				channel.Write(m.data)
			}
		}
		return nil
	})
	return t.Wait()
}
