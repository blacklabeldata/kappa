package server

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
	assert.NotNil(t, NewNodeList())
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
		Role:      "kappa-server",
		SSHPort:   9022,
		Bootstrap: false,
		Addr:      net.TCPAddr{net.ParseIP("127.0.0.1"), 7946, ""},
	}
}

func (suite *ManagerTestSuite) TestAddNode() {
	mgr := &nodeList{nodes: make(map[string]NodeDetails)}

	// Add node
	mgr.AddNode(suite.Details)
	d, ok := mgr.nodes[suite.Details.Addr.String()]

	// Test expectations
	suite.True(ok)
	suite.Equal(suite.Details, d)
}

func (suite *ManagerTestSuite) TestRemoveNode() {
	mgr := &nodeList{nodes: make(map[string]NodeDetails)}

	// Add node
	mgr.AddNode(suite.Details)

	// Remove node
	mgr.RemoveNode(suite.Details)
	_, ok := mgr.nodes[suite.Details.Addr.String()]

	// Test expectations
	suite.False(ok)
}

func (suite *ManagerTestSuite) TestGetNodes() {
	mgr := &nodeList{nodes: make(map[string]NodeDetails)}

	// Add node
	mgr.AddNode(suite.Details)

	details := NodeDetails{
		Name:      "node-2",
		Cluster:   "kappa",
		Role:      "kappa-server",
		SSHPort:   9022,
		Bootstrap: false,
		Addr:      net.TCPAddr{net.ParseIP("127.0.0.1"), 7947, ""},
	}
	mgr.AddNode(details)

	// Get nodes
	nodes := mgr.GetNodes()

	// Test expectations
	var node1, node2 bool
	for _, node := range nodes {
		if node.Name == "node" {
			node1 = true
			suite.Equal(suite.Details, node)
		} else if node.Name == "node-2" {
			node2 = true
			suite.Equal(details, node)
		}
	}
	suite.True(node1)
	suite.True(node2)
	suite.Equal(2, len(nodes), "Nodes length should be 2")
}

func (suite *ManagerTestSuite) TestSize() {
	mgr := &nodeList{nodes: make(map[string]NodeDetails)}

	// Add node
	suite.Equal(0, mgr.Size(), "Size should be 0")
	mgr.AddNode(suite.Details)
	suite.Equal(1, mgr.Size(), "Size should be 1")
}

func (suite *ManagerTestSuite) TestFindByRole() {
	mgr := &nodeList{nodes: make(map[string]NodeDetails)}

	// Add node
	mgr.AddNode(suite.Details)

	details := NodeDetails{
		Name:      "node-2",
		Cluster:   "kappa",
		Role:      "kappa",
		SSHPort:   9022,
		Bootstrap: false,
		Addr:      net.TCPAddr{net.ParseIP("127.0.0.1"), 7947, ""},
	}
	mgr.AddNode(details)

	// Filter nodes
	nodes := mgr.FindByRole("kappa-server")
	suite.Equal(1, len(nodes), "FindByRole should have only found one node")
	suite.Equal(suite.Details, nodes[0], "FindByRole should have fould suite.Details")
}

func (suite *ManagerTestSuite) TestFindByDataCenter() {
	mgr := &nodeList{nodes: make(map[string]NodeDetails)}

	// Add node
	mgr.AddNode(suite.Details)

	details := NodeDetails{
		Name:       "node-2",
		Cluster:    "kappa",
		DataCenter: "dc1",
		Role:       "kappa",
		SSHPort:    9022,
		Bootstrap:  false,
		Addr:       net.TCPAddr{net.ParseIP("127.0.0.1"), 7947, ""},
	}
	mgr.AddNode(details)

	// Filter nodes
	nodes := mgr.FindByDataCenter("dc1")
	suite.Equal(1, len(nodes), "FindByDataCenter should have only found one node")
	suite.Equal(details, nodes[0])
}

func (suite *ManagerTestSuite) TestFindByService() {
	mgr := &nodeList{nodes: make(map[string]NodeDetails)}

	// Add node
	mgr.AddNode(suite.Details)

	details := NodeDetails{
		Name:      "node-2",
		Cluster:   "kappa",
		Service:   "kappa-server",
		Role:      "kappa",
		SSHPort:   9022,
		Bootstrap: false,
		Addr:      net.TCPAddr{net.ParseIP("127.0.0.1"), 7947, ""},
	}
	mgr.AddNode(details)

	// Filter nodes
	nodes := mgr.FindByService("kappa-server")
	suite.Equal(1, len(nodes), "FindByService should have only found one node")
	suite.Equal(details, nodes[0])
}
