package server

import (
	"fmt"

	"github.com/hashicorp/serf/serf"
)

// mergeDelegate is used to handle a cluster merge on the gossip
// ring. We check that the peers are in the same cluster and abort the
// merge if there is a mis-match.
type mergeDelegate struct {
	name string
}

// NotifyMerge determines if two serf clusters can be merged. Every new serf.Member must be
// in the same cluster and must have all the correct tags.
func (md *mergeDelegate) NotifyMerge(members []*serf.Member) error {
	for _, m := range members {
		node, err := GetKappaServer(*m)
		if err != nil {
			return err
		}

		if node.Cluster != md.name {
			return fmt.Errorf("Member '%s' part of wrong datacenter '%s'",
				m.Name, node.Cluster)
		}
	}
	return nil
}
