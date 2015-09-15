package server

import (
	"io"
	"time"
)

// DatabaseConfig contains all the information to start the Kappa server.
type DatabaseConfig struct {

	// AdminCertificateFile is the path to the admin user's certificate.
	AdminCertificateFile string

	// CACertificateFile is the path to the CA certificate.
	CACertificateFile string

	// DataPath is the root directory for all data produced by the database.
	DataPath string

	// LogOutput is the writer to which all logs are
	// written to. If nil, it defaults to os.Stdout.
	LogOutput io.Writer

	// SSHBindAddress is the address on which the SSH server listens.
	SSHBindAddress string

	// SSHConnectionDeadline is the deadline for maximum connection attempts.
	SSHConnectionDeadline time.Duration

	// SSHPrivateKeyFile refers to the private key file of the SSH server.
	SSHPrivateKeyFile string
}
