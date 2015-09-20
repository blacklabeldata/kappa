package server

import (
	"net"
	"strings"

	"github.com/hashicorp/serf/serf"
)

const (
	// StatusReap is used to update the status of a node if we
	// are handling a EventMemberReap
	StatusReap = serf.MemberStatus(-1)

	// kappaEventPrefix is pre-pended to a kappa event to distinguish it
	kappaEventPrefix = "kappa-event:"
)

// kappaEventName computes the name of a kappa event
func kappaEventName(name string) string {
	return kappaEventPrefix + name
}

// isKappaEvent checks if a serf event is a Kappa event
func isKappaEvent(name string) bool {
	return strings.HasPrefix(name, kappaEventPrefix)
}

// rawKappaEventName is used to get the raw kappa event name
func rawKappaEventName(name string) string {
	return strings.TrimPrefix(name, kappaEventPrefix)
}

// serfEventHandler is used to handle events from the Serf cluster
func (s *Server) serfEventHandler() error {
	for {
		select {
		case e := <-s.serfEventCh:
			switch e.EventType() {
			case serf.EventMemberJoin:
				s.nodeJoin(e.(serf.MemberEvent))
				s.localMemberEvent(e.(serf.MemberEvent))

			case serf.EventMemberLeave, serf.EventMemberFailed:
				s.nodeFailed(e.(serf.MemberEvent))
				s.localMemberEvent(e.(serf.MemberEvent))

			case serf.EventMemberReap:
				s.localMemberEvent(e.(serf.MemberEvent))
			case serf.EventUser:
				s.localEvent(e.(serf.UserEvent))
			case serf.EventMemberUpdate: // Ignore
			case serf.EventQuery: // Ignore
			default:
				s.logger.Warn("kappa: unhandled Serf Event: %#v", e)
			}

		case <-s.t.Dying():
			return nil
		}
	}
}

// localMemberEvent is used to reconcile Serf events with the strongly
// consistent store if we are the current leader
func (s *Server) localMemberEvent(me serf.MemberEvent) {
	// Do nothing if we are not the leader
	if !s.IsLeader() {
		return
	}

	// Check if this is a reap event
	isReap := me.EventType() == serf.EventMemberReap

	// Queue the members for reconciliation
	for _, m := range me.Members {
		// Change the status if this is a reap event
		if isReap {
			m.Status = StatusReap
		}
		select {
		case s.reconcileCh <- m:
		default:
		}
	}
}

// localEvent is called when we receive an event on the local Serf
func (s *Server) localEvent(event serf.UserEvent) {

	// Handle only consul events
	if !strings.HasPrefix(event.Name, KappaServiceName+":") {
		return
	}

	switch name := event.Name; {
	case name == LeaderEventName:
		s.logger.Info("kappa: New leader elected: %s", event.Payload)

	case isKappaEvent(name):
		event.Name = rawKappaEventName(name)
		s.logger.Debug("kappa: user event: %s", event.Name)

		// Send event to processing channel
		s.kappaEventCh <- event

	default:
		s.logger.Warn("kappa: Unhandled local event: %v", event)
	}
}

// nodeJoin is used to handle join events on the both serf clusters
func (s *Server) nodeJoin(me serf.MemberEvent) {
	for _, m := range me.Members {
		details, err := getKappaServer(m)
		if err != nil {
			continue
		}
		s.logger.Info("kappa: adding server %s", details)

		// Add to the local list as well
		if details.Cluster == s.config.ClusterName {
			s.localLock.Lock()
			s.localKappas[details.Addr.String()] = details
			s.localLock.Unlock()
		}

		// If we still expecting to bootstrap, may need to handle this
		if s.config.BootstrapExpect != 0 {
			s.maybeBootstrap()
		}
	}
}

// maybeBootsrap is used to handle bootstrapping when a new consul server joins
func (s *Server) maybeBootstrap() {
	// // TODO: Requires Raft!
	// index, err := s.raftStore.LastIndex()
	// if err != nil {
	// 	s.logger.Error("kappa: failed to read last raft index: %v", err)
	// 	return
	// }
	// // Bootstrap can only be done if there are no committed logs,
	// // remove our expectations of bootstrapping
	// if index != 0 {
	// 	s.config.BootstrapExpect = 0
	// 	return
	// }

	// Scan for all the known servers
	members := s.serf.Members()
	addrs := make([]string, 0)
	for _, member := range members {
		details, err := getKappaServer(member)
		if err != nil {
			continue
		}
		if details.Cluster != s.config.ClusterName {
			s.logger.Error("kappa: Member %v has a conflicting datacenter, ignoring", member)
			continue
		}
		if details.Expect != 0 && details.Expect != s.config.BootstrapExpect {
			s.logger.Error("kappa: Member %v has a conflicting expect value. All nodes should expect the same number.", member)
			return
		}
		if details.Bootstrap {
			s.logger.Error("kappa: Member %v has bootstrap mode. Expect disabled.", member)
			return
		}
		addr := &net.TCPAddr{IP: member.Addr, Port: details.SSHPort}
		addrs = append(addrs, addr.String())
	}

	// Skip if we haven't met the minimum expect count
	if len(addrs) < s.config.BootstrapExpect {
		return
	}

	// Update the peer set
	// TODO: Requires Raft!
	// s.logger.Info("kappa: Attempting bootstrap with nodes: %v", addrs)
	// if err := s.raft.SetPeers(addrs).Error(); err != nil {
	// 	s.logger.Error("kappa: failed to bootstrap peers: %v", err)
	// }

	// Bootstrapping complete, don't enter this again
	s.config.BootstrapExpect = 0
}

// nodeFailed is used to handle fail events on both the serf clusters
func (s *Server) nodeFailed(me serf.MemberEvent) {
	s.localLock.Lock()
	for _, m := range me.Members {
		details, err := getKappaServer(m)
		if err != nil {
			continue
		}
		s.logger.Info("kappa: removing server %s", details)

		// Remove from the local list as well
		delete(s.localKappas, details.Addr.String())
	}
	s.localLock.Unlock()
}
