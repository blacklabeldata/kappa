package serf

import (
	"fmt"
	"net"
	"testing"

	"github.com/hashicorp/serf/serf"
	"github.com/stretchr/testify/assert"
)

func TestGetKappaEventName(t *testing.T) {

	// GetKappaEventName should prepend Kappa event prefix to event type
	assert.Equal(t, "kappa-event:some-event", GetKappaEventName("some-event"))
	assert.Equal(t, "kappa-event:", GetKappaEventName(""))
}

func TestIsKappaEvent(t *testing.T) {

	// IsKappaEvent should return true if the event name starts with the Kappa event prefix.
	assert.True(t, IsKappaEvent("kappa-event:some-event"))

	// IsKappaEvent should return false if the event name does not start with the Kappa event prefix.
	assert.False(t, IsKappaEvent("some-event"))
}

func TestGetRawEventName(t *testing.T) {

	// GetRawEventName strips the Kappa event prefix
	assert.Equal(t, "some-event", GetRawEventName("kappa-event:some-event"))
	assert.Equal(t, "some-event", GetRawEventName("some-event"))
	assert.Equal(t, "", GetRawEventName("kappa-event:"))
}

func TestValidateNode_NoRoleTag(t *testing.T) {
	m := serf.Member{
		Tags: map[string]string{},
	}

	ok, _, _ := ValidateNode(m)
	assert.False(t, ok, "ok should be false without role tag")
}

func TestValidateNode_InvalidRoleTag(t *testing.T) {
	m := serf.Member{
		Tags: map[string]string{
			"role": "server",
		},
	}

	ok, _, _ := ValidateNode(m)
	assert.False(t, ok, "ok should be false with invalid role tag")
}

func TestValidateNode_NoClusterTag(t *testing.T) {
	m := serf.Member{
		Tags: map[string]string{
			"role": "kappa-server",
		},
	}

	ok, role, cluster := ValidateNode(m)
	assert.False(t, ok, "ok should be false without cluster tag")
	assert.Equal(t, "", role, "role should be empty")
	assert.Equal(t, "", cluster, "cluster should be empty")
}

func TestValidateNode_ValidRoleTag(t *testing.T) {
	m := serf.Member{
		Tags: map[string]string{
			"role":    "kappa-server",
			"cluster": "kappa",
		},
	}

	ok, role, cluster := ValidateNode(m)
	assert.True(t, ok, "ok should be true with valid role tag")
	assert.Equal(t, "kappa-server", role, "role should be kappa-server")
	assert.Equal(t, "kappa", cluster, "cluster should be kappa")
}

func TestGetKappaServer_NoTags(t *testing.T) {
	m := serf.Member{
		Tags: map[string]string{},
	}

	node, err := GetKappaServer(m)
	assert.Nil(t, node, "node should be nil")
	assert.NotNil(t, err, "err should not be nil")
}

func TestGetKappaServer_NoPortTag(t *testing.T) {
	m := serf.Member{
		Tags: map[string]string{
			"role":    "kappa-server",
			"cluster": "kappa",
		},
	}

	node, err := GetKappaServer(m)
	assert.Nil(t, node, "node should be nil")
	assert.NotNil(t, err, "err should not be nil")
}

func TestGetKappaServer_InvalidPortTag(t *testing.T) {
	m := serf.Member{
		Addr: net.ParseIP("127.0.0.1"),
		Tags: map[string]string{
			"role":    "kappa-server",
			"cluster": "kappa",
			"port":    "abc",
		},
	}

	node, err := GetKappaServer(m)
	assert.Nil(t, node, "node should be nil")
	assert.NotNil(t, err, "err should not be nil")
}

func TestGetKappaServer_NoBootstrap(t *testing.T) {
	m := serf.Member{
		Name: "node",
		Addr: net.ParseIP("127.0.0.1"),
		Tags: map[string]string{
			"role":    "kappa-server",
			"cluster": "kappa",
			"port":    "9000",
		},
	}

	node, err := GetKappaServer(m)
	assert.NotNil(t, node, "node should be nil")
	assert.Nil(t, err, "err should be nil")

	assert.Equal(t, "node", node.Name, "Name should be node")
	assert.Equal(t, "kappa-server", node.Role, "Role should be kappa-server")
	assert.Equal(t, "kappa", node.Cluster, "Cluster should be kappa")
	assert.Equal(t, 9000, node.SSHPort, "SSHPort should be 9000")
	assert.Equal(t, false, node.Bootstrap, "Bootstrap should be false")
	assert.Equal(t, net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9000}, node.Addr, "Addr should be 127.0.0.1")
}

func TestGetKappaServer_Bootstrap(t *testing.T) {
	m := serf.Member{
		Name: "node",
		Addr: net.ParseIP("127.0.0.1"),
		Tags: map[string]string{
			"role":      "kappa-server",
			"cluster":   "kappa",
			"port":      "9000",
			"bootstrap": "",
		},
	}

	node, err := GetKappaServer(m)
	assert.NotNil(t, node, "node should be nil")
	assert.Nil(t, err, "err should be nil")

	assert.Equal(t, "node", node.Name, "Name should be node")
	assert.Equal(t, "kappa-server", node.Role, "Role should be kappa-server")
	assert.Equal(t, "kappa", node.Cluster, "Cluster should be kappa")
	assert.Equal(t, 9000, node.SSHPort, "SSHPort should be 9000")
	assert.Equal(t, true, node.Bootstrap, "Bootstrap should be true")
	assert.Equal(t, net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9000}, node.Addr, "Addr should be 127.0.0.1")
}

func TestNodeDetails_String(t *testing.T) {
	n := NodeDetails{
		Name:    "node-1",
		Role:    "server",
		Cluster: "kappa",
		Addr:    net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9000},
		SSHPort: 9000,
	}
	b, err := n.Addr.IP.MarshalText()
	assert.Nil(t, err)

	// Test assertions
	s := fmt.Sprintf("NodeDetails{Name: \"%s\", Role: \"%s\", Cluster: \"%s\", Addr: \"%s:%s\"}", n.Name, n.Role, n.Cluster, string(b), n.SSHPort)
	assert.Equal(t, s, n.String())
}

func TestNodeDetails_StringError(t *testing.T) {
	n := NodeDetails{
		Name:    "node-1",
		Role:    "server",
		Cluster: "kappa",
		Addr:    net.TCPAddr{IP: make([]byte, 1), Port: 9000},
		SSHPort: 9000,
	}
	_, err := n.Addr.IP.MarshalText()
	assert.NotNil(t, err)

	// Test assertions
	s := fmt.Sprintf("NodeDetails{Name: \"%s\", Role: \"%s\", Cluster: \"%s\"}", n.Name, n.Role, n.Cluster)
	assert.Equal(t, s, n.String())
}
