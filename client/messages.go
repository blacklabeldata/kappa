package client

import (
	"crypto/rand"
	"io"
	"math/big"

	"github.com/blacklabeldata/kappa/common"
)

type Message interface {
	Write(w io.Writer, colors common.ColorCodes)
}

type SimpleMessage struct {
	Text string
}

func (s SimpleMessage) Write(w io.Writer, colors common.ColorCodes) {
	w.Write([]byte(" " + s.Text + "\r\n"))
}

// Quote is a message with an author
type Quote struct {
	Color  []byte
	Author string
	Text   string
}

func (q Quote) Write(w io.Writer, colors common.ColorCodes) {
	w.Write(q.Color)
	w.Write([]byte(" " + q.Author))
	w.Write(colors.Reset)
	w.Write([]byte(": " + q.Text))
	w.Write([]byte("\r\n"))
}

// Conversation displays a list of quotes
type Conversation struct {
	Quotes []Quote
}

// Write actually writes the message to the terminal
func (c Conversation) Write(w io.Writer, colors common.ColorCodes) {
	for _, q := range c.Quotes {
		q.Write(w, colors)
	}
	w.Write([]byte("\r\n"))
}

// LoginMessage writes a message at login
func GetMessage(w io.Writer, colors common.ColorCodes) {
	var messages = []Message{
		SimpleMessage{"Welcome to Kappa DB, Yo!"},
		Quote{colors.LightMagenta, "Jessy Pinkman", "Yeah, Bitch! Magnets!"},
		Quote{colors.LightMagenta, "Jessy Pinkman", "Yeah, Science!"},
		Quote{colors.LightBlue, "Saul Goodman", "Better call Saul."},
		Conversation{
			Quotes: []Quote{
				Quote{colors.LightGreen, "Walter White", "One particular element comes to mind..."},
				Quote{colors.LightMagenta, "Jessy Pinkman", "Ohhhhh... wire.."},
			},
		},
	}

	// Choose message
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(messages))))
	if err != nil {
		index = big.NewInt(0)
	}
	messages[index.Int64()].Write(w, colors)
}
