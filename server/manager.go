package server

import "sync"

// NodeList maintains a list of remote nodes in the cluster.
type NodeList interface {

	// AddNode adds a new node to the cluster.
	AddNode(NodeDetails)

	// RemoveNode removes a node from the cluster.
	RemoveNode(NodeDetails)

	// GetNodes returns a list of nodes currently in the cluster
	GetNodes() []NodeDetails

	// Size returns the number of nodes in the cluster.
	Size() int

	// FindByDataCenter returns a list of nodes which are in the given data denter.
	FindByDataCenter(string) []NodeDetails

	// FindByRole returns a list of nodes which have the given role.
	FindByRole(string) []NodeDetails

	// FindByService returns a list of nodes which are running the given service.
	FindByService(string) []NodeDetails

	// Filter returns a list of nodes which satisfy the given filter.
	Filter(func(NodeDetails) bool) []NodeDetails
}

// NewNodeList creates a new nodeList which tracks the number of nodes in the cluster.
func NewNodeList() NodeList {
	return &nodeList{nodes: make(map[string]NodeDetails)}
}

// nodeList implements the NodeList interface.
type nodeList struct {
	sync.Mutex
	nodes map[string]NodeDetails
}

// AddNode adds a node to the manager.
func (n *nodeList) AddNode(d NodeDetails) {
	n.Lock()
	n.nodes[d.Addr.String()] = d
	n.Unlock()
}

// RemoveNode removes a node from the cluster.
func (n *nodeList) RemoveNode(d NodeDetails) {
	n.Lock()
	delete(n.nodes, d.Addr.String())
	n.Unlock()
}

// GetNodes returns the current list of nodes in the cluster.
func (n *nodeList) GetNodes() (nodes []NodeDetails) {
	n.Lock()
	nodes = make([]NodeDetails, 0, len(n.nodes))
	for _, v := range n.nodes {
		nodes = append(nodes, v)
	}
	n.Unlock()
	return
}

// Size returns the number of nodes in the cluster.
func (n *nodeList) Size() (size int) {
	n.Lock()
	size = len(n.nodes)
	n.Unlock()
	return
}

// Filter filters the node list with the given predicate. All nodes which pass the filter will be included in the return array.
func (n *nodeList) Filter(filter func(NodeDetails) bool) (nodes []NodeDetails) {
	n.Lock()
	nodes = make([]NodeDetails, 0, len(n.nodes))
	for _, v := range n.nodes {
		if filter(v) {
			nodes = append(nodes, v)
		}
	}
	n.Unlock()
	return
}

// FindByRole filters the node list by the given role. All nodes matching the role will be returned.
func (n *nodeList) FindByRole(role string) (nodes []NodeDetails) {
	return n.Filter(func(d NodeDetails) bool {
		return role == d.Role
	})
}

// FindByDataCenter filters the node list by the given data center. All nodes matching the data center will be returned.
func (n *nodeList) FindByDataCenter(dc string) (nodes []NodeDetails) {
	return n.Filter(func(d NodeDetails) bool {
		return dc == d.DataCenter
	})
}

// FindByService filters the node list by the given service. All nodes matching the service will be returned.
func (n *nodeList) FindByService(svc string) (nodes []NodeDetails) {
	return n.Filter(func(d NodeDetails) bool {
		return svc == d.Service
	})
}
