package server

import (
	"testing"

	"github.com/hashicorp/serf/serf"
	log "github.com/mgutz/logxi/v1"
	"github.com/stretchr/testify/assert"
)

func TestReconciler(t *testing.T) {
	reconcilerCh := make(chan serf.Member, 2)

	// Create reconciler
	reconciler := &SerfReconciler{
		ReconcileCh: reconcilerCh,
	}

	// Handle members
	reconciler.Reconcile(serf.Member{
		Name:   "node-1",
		Status: serf.StatusAlive,
	})
	reconciler.Reconcile(serf.Member{
		Name:   "node-2",
		Status: serf.StatusAlive,
	})

	// Read reconcile events
	select {
	case m := <-reconcilerCh:
		assert.Equal(t, "node-1", m.Name)
		assert.Equal(t, serf.StatusAlive, m.Status)
	default:
		t.Fail()
	}

	// Read second message
	select {
	case m := <-reconcilerCh:
		assert.Equal(t, "node-2", m.Name)
		assert.Equal(t, serf.StatusAlive, m.Status)
	default:
		t.Fail()
	}
}

func TestReconcilerReap(t *testing.T) {
	reconcilerCh := make(chan serf.Member, 2)

	// Create reconciler
	reconciler := &SerfReconciler{
		ReconcileCh: reconcilerCh,
	}

	// Handle Event
	reconciler.Reconcile(serf.Member{
		Name: "node-1",
	})
	reconciler.Reconcile(serf.Member{
		Name: "node-2",
	})

	// Read reconcile events
	select {
	case m := <-reconcilerCh:
		assert.Equal(t, "node-1", m.Name)
	default:
		t.Fail()
	}

	// Read second message
	select {
	case m := <-reconcilerCh:
		assert.Equal(t, "node-2", m.Name)
	default:
		t.Fail()
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
