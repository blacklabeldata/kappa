package server

import (
	"errors"
	"net"
	"os"
	"path/filepath"

	"github.com/hashicorp/serf/serf"
)

// ensurePath is used to make sure a path exists
func ensurePath(path string, dir bool) error {
	if !dir {
		path = filepath.Dir(path)
	}
	return os.MkdirAll(path, 0755)
}

// isKappaNode returns whether a node is a Kappa server as well as its cluster.
func isKappaNode(member serf.Member) (bool, string) {

	// Get role name
	if role, ok := member.Tags["role"]; !ok {
		return false, ""
	} else if role != "kappa" {
		return false, ""
	}

	// Get cluster name
	if name, ok := member.Tags["cluster"]; ok {
		return ok, name
	}
	return false, ""
}

// validateNode should validate all the Serf tags for the given member and returns
// NodeDetails and any that occured error.
func validateNode(serf.Member) (NodeDetails, error) {
	return NodeDetails{}, errors.New("Not implemented")
}

// NodeDetails stores details about a single serf.Member
type NodeDetails struct {
	Name      string
	Cluster   string
	SSHPort   int
	Bootstrap bool
	Addr      net.TCPAddr
}
