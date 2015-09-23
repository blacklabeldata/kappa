package serf

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/hashicorp/serf/serf"
)

const (

	// KappaEventPrefix is pre-pended to a kappa event to distinguish it
	KappaEventPrefix = "kappa-event:"
)

// GetKappaEventName computes the name of a kappa event
func GetKappaEventName(name string) string {
	return KappaEventPrefix + name
}

// IsKappaEvent checks if a serf event is a Kappa event
func IsKappaEvent(name string) bool {
	return strings.HasPrefix(name, KappaEventPrefix)
}

// GetRawEventName is used to get the raw kappa event name
func GetRawEventName(name string) string {
	return strings.TrimPrefix(name, KappaEventPrefix)
}

// IsKappaNode returns whether a node is a Kappa server as well as its cluster.
func IsKappaNode(member serf.Member) (bool, string) {

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

// GetKappaServer should validate all the Serf tags for the given member and returns
// NodeDetails and any that occured error.
func GetKappaServer(m serf.Member) (n *NodeDetails, err error) {

	// Get node cluster
	cluster, ok := m.Tags["cluster"]
	if !ok {
		err = errors.New("error: member missing cluster tag")
		return
	}

	// Get node SSH port
	port, ok := m.Tags["port"]
	if !ok {
		err = errors.New("error: member missing ssh port")
		return
	}

	// Convert port to int
	p, err := strconv.Atoi(port)
	if err != nil {
		err = fmt.Errorf("error: member ssh port cannot be converted to string: '%s'", port)
		return
	}

	// Get node bootstrap
	// All nodes which have this tag are bootstrapped
	_, bootstrap := m.Tags["bootstrap"]

	// Get SSH addr
	addr := net.TCPAddr{IP: m.Addr, Port: p}

	n = &NodeDetails{}
	n.Name = m.Name
	n.Cluster = cluster
	n.SSHPort = p
	n.Bootstrap = bootstrap
	n.Addr = addr
	return
}

// NodeDetails stores details about a single serf.Member
type NodeDetails struct {
	Name      string
	Cluster   string
	SSHPort   int
	Bootstrap bool
	Addr      net.TCPAddr
	Expect    int
}

func (n NodeDetails) String() string {
	return fmt.Sprintf("%#v", n)
}
