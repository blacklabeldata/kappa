package server

import (
	log "github.com/mgutz/logxi/v1"

	"github.com/hashicorp/serf/serf"
)

const (
	// StatusReap is used to update the status of a node if we
	// are handling a EventMemberReap
	StatusReap = serf.MemberStatus(-1)
)

// SerfReconciler dispatches membership changes to Raft. If IsLeader is nil,
// the server will panic.
type SerfReconciler struct {
	ReconcileCh chan serf.Member
}

// Reconcile is used to reconcile Serf events with the strongly
// consistent store if we are the current leader
func (s *SerfReconciler) Reconcile(m serf.Member) {
	select {
	case s.ReconcileCh <- m:
	default:
	}
}

// SerfUserEventHandler handles both local and remote user events in Serf.
type SerfUserEventHandler struct {
	Logger      log.Logger
	UserEventCh chan serf.UserEvent
}

// HandleUserEvent is called when a user event is received from both local and remote nodes.
func (s *SerfUserEventHandler) HandleUserEvent(event serf.UserEvent) {
	s.Logger.Debug("kappa: user event: %s", event.Name)

	// Send event to processing channel
	s.UserEventCh <- event
}

// SerfNodeJoinHandler processes cluster Join events.
type SerfNodeJoinHandler struct {
	// ClusterManager ClusterManager
	Cluster NodeList
	Logger  log.Logger
}

// HandleMemberEvent is used to handle join events on the serf cluster.
func (s *SerfNodeJoinHandler) HandleMemberEvent(me serf.MemberEvent) {
	for _, m := range me.Members {
		details, err := GetKappaServer(m)
		if err != nil {
			s.Logger.Warn("kappa: error adding server", err)
			continue
		}
		s.Logger.Info("kappa: adding server", details.String())

		// Add to the local list as well
		s.Cluster.AddNode(*details)

		// // If we still expecting to bootstrap, may need to handle this
		// if s.config.BootstrapExpect != 0 {
		// 	s.maybeBootstrap()
		// }
	}
}

type SerfNodeUpdateHandler struct {
	Cluster NodeList
	Logger  log.Logger
}

// nodeJoin is used to handle join events on the both serf clusters
func (s *SerfNodeUpdateHandler) HandleMemberEvent(me serf.MemberEvent) {
	for _, m := range me.Members {
		details, err := GetKappaServer(m)
		if err != nil {
			s.Logger.Warn("kappa: error updating server", err)
			continue
		}
		s.Logger.Info("kappa: updating server", details.String())

		// Add to the local list as well
		s.Cluster.AddNode(*details)

		// // If we still expecting to bootstrap, may need to handle this
		// if s.config.BootstrapExpect != 0 {
		// 	s.maybeBootstrap()
		// }
	}
}

// // maybeBootsrap is used to handle bootstrapping when a new consul server joins
// func (s *SerfNodeJoinHandler) maybeBootstrap() {
// 	// // TODO: Requires Raft!
// 	// index, err := s.raftStore.LastIndex()
// 	// if err != nil {
// 	// 	s.logger.Error("kappa: failed to read last raft index: %v", err)
// 	// 	return
// 	// }
// 	// // Bootstrap can only be done if there are no committed logs,
// 	// // remove our expectations of bootstrapping
// 	// if index != 0 {
// 	// 	s.config.BootstrapExpect = 0
// 	// 	return
// 	// }

// 	// Scan for all the known servers
// 	members := s.serf.Members()
// 	addrs := make([]string, 0)
// 	for _, member := range members {
// 		details, err := GetKappaServer(member)
// 		if err != nil {
// 			continue
// 		}
// 		if details.Cluster != s.config.ClusterName {
// 			s.Logger.Warn("kappa: Member %v has a conflicting datacenter, ignoring", member)
// 			continue
// 		}
// 		if details.Expect != 0 && details.Expect != s.config.BootstrapExpect {
// 			s.Logger.Warn("kappa: Member %v has a conflicting expect value. All nodes should expect the same number.", member)
// 			return
// 		}
// 		if details.Bootstrap {
// 			s.Logger.Warn("kappa: Member %v has bootstrap mode. Expect disabled.", member)
// 			return
// 		}
// 		addr := &net.TCPAddr{IP: member.Addr, Port: details.SSHPort}
// 		addrs = append(addrs, addr.String())
// 	}

// 	// Skip if we haven't met the minimum expect count
// 	if len(addrs) < s.config.BootstrapExpect {
// 		return
// 	}

// 	// Update the peer set
// 	// TODO: Requires Raft!
// 	// s.logger.Info("kappa: Attempting bootstrap with nodes: %v", addrs)
// 	// if err := s.raft.SetPeers(addrs).Error(); err != nil {
// 	// 	s.logger.Error("kappa: failed to bootstrap peers: %v", err)
// 	// }

// 	// Bootstrapping complete, don't enter this again
// 	s.config.BootstrapExpect = 0
// }

// SerfNodeLeaveHandler processes cluster leave events.
type SerfNodeLeaveHandler struct {
	Cluster NodeList
	Logger  log.Logger
}

// HandleMemberEvent is used to handle fail events in the Serf cluster.
func (s *SerfNodeLeaveHandler) HandleMemberEvent(me serf.MemberEvent) {
	for _, m := range me.Members {
		details, err := GetKappaServer(m)
		if err != nil {
			continue
		}
		s.Logger.Info("kappa: removing server %s", details)

		// Remove from the local list as well
		s.Cluster.RemoveNode(*details)
	}
}
