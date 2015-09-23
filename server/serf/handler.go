package serf

import (
	"github.com/hashicorp/serf/serf"
	log "github.com/mgutz/logxi/v1"
)

// EventHandler processes generic Serf events. Depending on the
// event type, more processing may be needed.
type EventHandler interface {
	HandleEvent(serf.Event)
}

// MemberEventHandler handles membership change events.
type MemberEventHandler interface {
	HandleMemberEvent(serf.MemberEvent)
}

// UserEventHandler handles user events.
type UserEventHandler interface {
	HandleUserEvent(serf.UserEvent)
}

// Reconciler is used to reconcile Serf events wilth an external process, like Raft.
type Reconciler interface {
	Reconcile(serf.MemberEvent)
}

// SerfEventHandler is used to dispatch various Serf events to separate event handlers.
type SerfEventHandler struct {
	NodeJoined MemberEventHandler
	NodeLeft   MemberEventHandler
	NodeFailed MemberEventHandler
	Reconciler Reconciler
	UserEvent  UserEventHandler
	Logger     log.Logger
}

// HandleEvent processes a generic Serf event and dispatches it to the appropriate
// destination.
func (s SerfEventHandler) HandleEvent(e serf.Event) {
	var reconcile bool
	switch e.EventType() {

	// If the event is a Join event, call NodeJoined and then reconcile event with
	// persistent storage.
	case serf.EventMemberJoin:
		reconcile = true
		if s.NodeJoined != nil {
			s.NodeJoined.HandleMemberEvent(e.(serf.MemberEvent))
		}

	// If the event is a Leave event, call NodeLeft and then reconcile event with
	// persistent storage.
	case serf.EventMemberLeave:
		reconcile = true
		if s.NodeLeft != nil {
			s.NodeLeft.HandleMemberEvent(e.(serf.MemberEvent))
		}

	// If the event is a Failed event, call NodeFailed and then reconcile event with
	// persistent storage.
	case serf.EventMemberFailed:
		reconcile = true
		if s.NodeFailed != nil {
			s.NodeFailed.HandleMemberEvent(e.(serf.MemberEvent))
		}

	// If the event is a Reap event, reconcile event with persistent storage.
	case serf.EventMemberReap:
		reconcile = true

	// If the event is a user event, call UserEvent
	case serf.EventUser:
		if s.UserEvent != nil {
			s.UserEvent.HandleUserEvent(e.(serf.UserEvent))
		}
	case serf.EventMemberUpdate: // Ignore
	case serf.EventQuery: // Ignore
	default:
		s.Logger.Warn("unhandled Serf Event: %#v", e)
		return
	}

	// Reconcile event with external storage
	if reconcile && s.Reconciler != nil {
		s.Reconciler.Reconcile(e.(serf.MemberEvent))
	}
}
