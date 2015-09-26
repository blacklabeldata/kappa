package serf

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

func TestNewNodeManager(t *testing.T) {
	assert.NotNil(t, NewNodeManager())
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ManagerTestSuite struct {
	suite.Suite
	Details NodeDetails
}

func (suite *ManagerTestSuite) SetupSuite() {
	suite.Details = NodeDetails{
		Name:      "node",
		Cluster:   "kappa",
		SSHPort:   9022,
		Bootstrap: false,
		Addr:      net.TCPAddr{net.ParseIP("127.0.0.1"), 7946, ""},
	}
}

func (suite *ManagerTestSuite) TestAddNode() {
	mgr := &nodeManager{nodes: make(map[string]NodeDetails)}

	// Add node
	mgr.AddNode(suite.Details)
	d, ok := mgr.nodes[suite.Details.Addr.String()]

	// Test expectations
	suite.True(ok)
	suite.Equal(suite.Details, d)
}

func (suite *ManagerTestSuite) TestRemoveNode() {
	mgr := &nodeManager{nodes: make(map[string]NodeDetails)}

	// Add node
	mgr.AddNode(suite.Details)

	// Remove node
	mgr.RemoveNode(suite.Details)
	_, ok := mgr.nodes[suite.Details.Addr.String()]

	// Test expectations
	suite.False(ok)
}

func (suite *ManagerTestSuite) TestGetNodes() {
	mgr := &nodeManager{nodes: make(map[string]NodeDetails)}

	// Add node
	mgr.AddNode(suite.Details)

	details := NodeDetails{
		Name:      "node-2",
		Cluster:   "kappa",
		SSHPort:   9022,
		Bootstrap: false,
		Addr:      net.TCPAddr{net.ParseIP("127.0.0.1"), 7947, ""},
	}
	mgr.AddNode(details)

	// Get nodes
	nodes := mgr.GetNodes()

	// Test expectations
	d, ok := nodes[suite.Details.Addr.String()]
	suite.True(ok)
	suite.Equal(suite.Details, d)

	d, ok = nodes[details.Addr.String()]
	suite.True(ok)
	suite.Equal(details, d)
}

func (suite *ManagerTestSuite) TestExists() {
	mgr := &nodeManager{nodes: make(map[string]NodeDetails)}

	// Add node
	mgr.AddNode(suite.Details)
	details := NodeDetails{
		Name:      "node-2",
		Cluster:   "kappa",
		SSHPort:   9022,
		Bootstrap: false,
		Addr:      net.TCPAddr{net.ParseIP("127.0.0.1"), 7947, ""},
	}
	mgr.AddNode(details)

	// Test expectations
	suite.True(mgr.Exists(suite.Details.Addr.String()))
	suite.True(mgr.Exists(details.Addr.String()))
	mgr.RemoveNode(details)
	suite.False(mgr.Exists(details.Addr.String()))
}
