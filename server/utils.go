package server

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"strconv"

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

// ValidateNode returns whether a node is a Kappa server as well as its cluster.
func ValidateNode(member serf.Member) (ok bool, role, cluster string) {

	// Get role name
	if role, ok = member.Tags["role"]; !ok {
		return false, "", ""
	} else if role != "kappa-server" {
		return false, "", ""
	}

	// Get cluster name
	if cluster, ok = member.Tags["cluster"]; ok {
		return true, role, cluster
	}
	return false, "", ""
}

// GetKappaServer should validate all the Serf tags for the given member and returns
// NodeDetails and any that occured error.
func GetKappaServer(m serf.Member) (n *NodeDetails, err error) {

	// Validate server node
	ok, role, cluster := ValidateNode(m)
	if !ok {
		return nil, errors.New("Invalid server node")
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

	n = &NodeDetails{
		Name:      m.Name,
		Role:      role,
		Cluster:   cluster,
		SSHPort:   p,
		Bootstrap: bootstrap,
		Addr:      addr,
	}
	return
}

// NodeDetails stores details about a single serf.Member
type NodeDetails struct {
	Name       string
	Role       string
	Cluster    string
	DataCenter string
	Service    string
	SSHPort    int
	Bootstrap  bool
	Addr       net.TCPAddr
	Expect     int
}

func (n NodeDetails) String() (s string) {
	// NodeDetails{Name: "somename", Role: "role", Cluster: "cluster", Addr: "127.0.0.1:9000"}
	if b, err := n.Addr.IP.MarshalText(); err != nil {
		s = fmt.Sprintf("NodeDetails{Name: \"%s\", Role: \"%s\", Cluster: \"%s\"}", n.Name, n.Role, n.Cluster)
	} else {
		s = fmt.Sprintf("NodeDetails{Name: \"%s\", Role: \"%s\", Cluster: \"%s\", Addr: \"%s:%d\"}", n.Name, n.Role, n.Cluster, string(b), n.SSHPort)
	}
	return
}

// ensurePath is used to make sure a path exists
func ensurePath(path string, dir bool) error {
	if !dir {
		path = filepath.Dir(path)
	}
	return os.MkdirAll(path, 0755)
}
