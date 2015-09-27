package server

import "sync"

// NodeManager tracks the current nodes in the cluster.
type NodeManager interface {
	AddNode(NodeDetails)
	RemoveNode(NodeDetails)
	GetNodes() map[string]NodeDetails
	Exists(string) bool
}

// NewNodeManager creates a basic NodeManager to keep up with which nodes are in the cluster.
func NewNodeManager() NodeManager {
	return &nodeManager{nodes: make(map[string]NodeDetails)}
}

// nodeManager implements the NodeManager interface.
type nodeManager struct {
	sync.Mutex
	nodes map[string]NodeDetails
}

// AddNode adds a node to the manager.
func (n *nodeManager) AddNode(d NodeDetails) {
	n.Lock()
	n.nodes[d.Addr.String()] = d
	n.Unlock()
}

// RemoveNode removes a node from the manager.
func (n *nodeManager) RemoveNode(d NodeDetails) {
	n.Lock()
	delete(n.nodes, d.Addr.String())
	n.Unlock()
}

// GetNodes returns the current list of nodes in the cluster.
func (n *nodeManager) GetNodes() (nodes map[string]NodeDetails) {
	n.Lock()
	nodes = n.nodes
	n.Unlock()
	return
}

// Exists returns whether or not a node is currently in the cluster.
func (n *nodeManager) Exists(node string) (exists bool) {
	n.Lock()
	_, exists = n.nodes[node]
	n.Unlock()
	return
}
