package serf

import (
	"net"
	"testing"

	"github.com/hashicorp/serf/serf"
	log "github.com/mgutz/logxi/v1"
	"github.com/stretchr/testify/suite"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestEventHandlerSuite(t *testing.T) {
	suite.Run(t, new(EventHandlerTestSuite))
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type EventHandlerTestSuite struct {
	suite.Suite
	Handler SerfEventHandler
	Member  serf.Member
}

// Make sure that VariableThatShouldStartAtFive is set to five
// before each test
func (suite *EventHandlerTestSuite) SetupTest() {
	suite.Handler = SerfEventHandler{
		Logger: &log.NullLogger{},
	}

	suite.Member = serf.Member{
		Name:        "",
		Addr:        net.ParseIP("127.0.0.1"),
		Port:        9022,
		Tags:        make(map[string]string),
		Status:      serf.StatusAlive,
		ProtocolMin: serf.ProtocolVersionMin,
		ProtocolMax: serf.ProtocolVersionMax,
		ProtocolCur: serf.ProtocolVersionMax,
		DelegateMin: serf.ProtocolVersionMin,
		DelegateMax: serf.ProtocolVersionMax,
		DelegateCur: serf.ProtocolVersionMax,
	}
}

// Test NodeJoin events are processed properly
func (suite *EventHandlerTestSuite) TestNodeJoined() {

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberJoin,
		[]serf.Member{suite.Member},
	}

	// Add NodeJoined handler
	m := &MockMemberEventHandler{}
	m.On("HandleMemberEvent", evt).Return()
	suite.Handler.NodeJoined = m

	// Add Reconciler
	r := &MockReconciler{}
	r.On("Reconcile", evt).Return()
	suite.Handler.Reconciler = r

	// Process event
	suite.Handler.HandleEvent(evt)
	m.AssertCalled(suite.T(), "HandleMemberEvent", evt)
	r.AssertCalled(suite.T(), "Reconcile", evt)
}

// Test NodeLeave messages are dispatched properly
func (suite *EventHandlerTestSuite) TestNodeLeave() {

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberLeave,
		[]serf.Member{suite.Member},
	}

	// Add NodeLeft handler
	m := &MockMemberEventHandler{}
	m.On("HandleMemberEvent", evt).Return()
	suite.Handler.NodeLeft = m

	// Add Reconciler
	r := &MockReconciler{}
	r.On("Reconcile", evt).Return()
	suite.Handler.Reconciler = r

	// Process event
	suite.Handler.HandleEvent(evt)
	m.AssertCalled(suite.T(), "HandleMemberEvent", evt)
	r.AssertCalled(suite.T(), "Reconcile", evt)
}

// Test NodeFailed messages are dispatched properly
func (suite *EventHandlerTestSuite) TestNodeFailed() {

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberFailed,
		[]serf.Member{suite.Member},
	}

	// Add NodeFailed handler
	m := &MockMemberEventHandler{}
	m.On("HandleMemberEvent", evt).Return()
	suite.Handler.NodeFailed = m

	// Add Reconciler
	r := &MockReconciler{}
	r.On("Reconcile", evt).Return()
	suite.Handler.Reconciler = r

	// Process event
	suite.Handler.HandleEvent(evt)
	m.AssertCalled(suite.T(), "HandleMemberEvent", evt)
	r.AssertCalled(suite.T(), "Reconcile", evt)
}

// Test NodeReaped messages are dispatched properly
func (suite *EventHandlerTestSuite) TestNodeReaped() {

	// Create Member Event
	evt := serf.MemberEvent{
		serf.EventMemberReap,
		[]serf.Member{suite.Member},
	}

	// Add Reconciler
	r := &MockReconciler{}
	r.On("Reconcile", evt).Return()
	suite.Handler.Reconciler = r

	// Process event
	suite.Handler.HandleEvent(evt)
	r.AssertCalled(suite.T(), "Reconcile", evt)
}

// Test UserEvent messages are dispatched properly
func (suite *EventHandlerTestSuite) TestUserEvent() {

	// Create Member Event
	evt := serf.UserEvent{
		LTime:    serf.LamportTime(0),
		Name:     "Event",
		Payload:  make([]byte, 0),
		Coalesce: false,
	}

	// Add UserEvent handler
	m := &MockUserEventHandler{}
	m.On("HandleUserEvent", evt).Return()
	suite.Handler.UserEvent = m

	// Add Reconciler
	r := &MockReconciler{}
	r.On("Reconcile", evt).Return()
	suite.Handler.Reconciler = r

	// Process event
	suite.Handler.HandleEvent(evt)
	m.AssertCalled(suite.T(), "HandleUserEvent", evt)
	r.AssertNotCalled(suite.T(), "Reconcile", evt)
}
