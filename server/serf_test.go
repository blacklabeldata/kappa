package server

import (
	"net"
	"testing"

	"github.com/hashicorp/serf/serf"
	log "github.com/mgutz/logxi/v1"
	"github.com/stretchr/testify/assert"
)

func TestReconciler(t *testing.T) {
	var leaderCalled bool
	reconcilerCh := make(chan serf.Member, 2)

	// Create reconciler
	reconciler := &SerfReconciler{
		IsLeader: func() bool {
			leaderCalled = true
			return true
		},
		ReconcileCh: reconcilerCh,
	}

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberJoin,
		[]serf.Member{
			serf.Member{
				Name:   "node-1",
				Addr:   net.ParseIP("127.0.0.1"),
				Port:   9022,
				Tags:   make(map[string]string),
				Status: serf.StatusAlive,
			},
			serf.Member{
				Name:   "node-2",
				Addr:   net.ParseIP("127.0.0.1"),
				Port:   9023,
				Tags:   make(map[string]string),
				Status: serf.StatusAlive,
			}},
	}

	// Handle Event
	reconciler.Reconcile(evt)
	assert.True(t, leaderCalled, "IsLeader should have been called")

	// Read reconcile events
	select {
	case m := <-reconcilerCh:
		assert.Equal(t, evt.Members[0], m)
	default:
		t.Fail()
	}

	// Read second message
	select {
	case m := <-reconcilerCh:
		assert.Equal(t, evt.Members[1], m)
	default:
		t.Fail()
	}
}

func TestReconcilerReap(t *testing.T) {
	var leaderCalled bool
	reconcilerCh := make(chan serf.Member, 2)

	// Create reconciler
	reconciler := &SerfReconciler{
		IsLeader: func() bool {
			leaderCalled = true
			return true
		},
		ReconcileCh: reconcilerCh,
	}

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberReap,
		[]serf.Member{
			serf.Member{
				Name:   "node-1",
				Status: serf.StatusNone,
			},
			serf.Member{
				Name:   "node-2",
				Status: serf.StatusNone,
			}},
	}

	// Handle Event
	reconciler.Reconcile(evt)
	assert.True(t, leaderCalled, "IsLeader should have been called")

	// Read reconcile events
	select {
	case m := <-reconcilerCh:
		assert.Equal(t, evt.Members[0].Name, m.Name)
		assert.Equal(t, StatusReap, m.Status)
	default:
		t.Fail()
	}

	// Read second message
	select {
	case m := <-reconcilerCh:
		assert.Equal(t, evt.Members[1].Name, m.Name)
		assert.Equal(t, StatusReap, m.Status)
	default:
		t.Fail()
	}
}

func TestReconcilerNotLeader(t *testing.T) {
	var leaderCalled bool
	reconcilerCh := make(chan serf.Member, 2)

	// Create reconciler
	reconciler := &SerfReconciler{
		IsLeader: func() bool {
			leaderCalled = true
			return false
		},
		ReconcileCh: reconcilerCh,
	}

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberReap,
		[]serf.Member{
			serf.Member{
				Name:   "node-1",
				Status: serf.StatusNone,
			}},
	}

	// Handle Event
	reconciler.Reconcile(evt)
	assert.True(t, leaderCalled, "IsLeader should have been called")

	// Read reconcile events
	select {
	case <-reconcilerCh:
		t.Fail()
	default:
	}
}

func TestNodeJoin_InvalidNode(t *testing.T) {
	nodelist := NewNodeList()
	logger := log.NullLogger{}

	// Create join handler
	handler := &SerfNodeJoinHandler{
		Cluster: nodelist,
		Logger:  &logger,
	}

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberJoin,
		[]serf.Member{
			serf.Member{
				Name:   "node-1",
				Status: serf.StatusNone,
			}},
	}

	// Process event
	handler.HandleMemberEvent(evt)

	// Verify node was not added
	assert.Equal(t, 0, nodelist.Size(), "Cluster should be empty")
}

func TestNodeJoin_ValidNode(t *testing.T) {
	nodelist := NewNodeList()
	logger := log.NullLogger{}

	// Create join handler
	handler := &SerfNodeJoinHandler{
		Cluster: nodelist,
		Logger:  &logger,
	}

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberJoin,
		[]serf.Member{
			serf.Member{
				Name: "node-1",
				Tags: map[string]string{
					"role":    "kappa-server",
					"cluster": "kappa",
					"port":    "9000",
				},
				Status: serf.StatusAlive,
			}},
	}

	// Process event
	handler.HandleMemberEvent(evt)

	// Verify node was added
	assert.Equal(t, 1, nodelist.Size(), "Cluster should not be empty")
}

func TestNodeUpdate_InvalidNode(t *testing.T) {
	nodelist := NewNodeList()
	logger := log.NullLogger{}

	// Create join handler
	handler := &SerfNodeUpdateHandler{
		Cluster: nodelist,
		Logger:  &logger,
	}

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberUpdate,
		[]serf.Member{
			serf.Member{
				Name:   "node-1",
				Status: serf.StatusNone,
			}},
	}

	// Process event
	handler.HandleMemberEvent(evt)

	// Verify node was not added
	assert.Equal(t, 0, nodelist.Size(), "Cluster should be empty")
}

func TestNodeUpdate_ValidNode(t *testing.T) {
	nodelist := NewNodeList()
	logger := log.NullLogger{}

	// Create join handler
	handler := &SerfNodeUpdateHandler{
		Cluster: nodelist,
		Logger:  &logger,
	}

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberUpdate,
		[]serf.Member{
			serf.Member{
				Name: "node-1",
				Tags: map[string]string{
					"role":    "kappa-server",
					"cluster": "kappa",
					"port":    "9000",
				},
				Status: serf.StatusAlive,
			}},
	}

	// Process event
	handler.HandleMemberEvent(evt)

	// Verify node was added
	assert.Equal(t, 1, nodelist.Size(), "Cluster should not be empty")
}

func TestNodeLeave_InvalidNode(t *testing.T) {
	nodelist := NewNodeList()
	logger := log.NullLogger{}

	// Create join handler
	handler := &SerfNodeLeaveHandler{
		Cluster: nodelist,
		Logger:  &logger,
	}

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberLeave,
		[]serf.Member{
			serf.Member{
				Name:   "node-1",
				Status: serf.StatusNone,
			}},
	}

	// Process event
	handler.HandleMemberEvent(evt)

	// Verify node was not added
	assert.Equal(t, 0, nodelist.Size(), "Cluster should be empty")
}

func TestNodeLeave_ValidNode(t *testing.T) {
	nodelist := NewNodeList()
	logger := log.NullLogger{}

	// Create join handler
	handler := &SerfNodeLeaveHandler{
		Cluster: nodelist,
		Logger:  &logger,
	}

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberLeave,
		[]serf.Member{
			serf.Member{
				Name: "node-1",
				Tags: map[string]string{
					"role":    "kappa-server",
					"cluster": "kappa",
					"port":    "9000",
				},
				Status: serf.StatusAlive,
			}},
	}

	// Add node
	details, _ := GetKappaServer(evt.Members[0])
	nodelist.AddNode(*details)
	assert.Equal(t, 1, nodelist.Size(), "Cluster should not be empty")

	// Process event
	handler.HandleMemberEvent(evt)

	// Verify node was added
	assert.Equal(t, 0, nodelist.Size(), "Cluster should be empty")
}
