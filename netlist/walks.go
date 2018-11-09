package netlist

import (
	"fmt"
	"log"
	"sart/ace"
	"sart/histogram"
	"sart/queue"
	"strings"
)

func (n *Node) AddRpAce(a *Node) {
	n.RpAce.SetBitsOf(*a.RpAce)
}

func (n *Node) AddWpAce(a *Node) {
	n.WpAce.SetBitsOf(*a.WpAce)
}

func (n *Netlist) Walk() (changed int) {
	d := n.WalkDn("")
	log.Println("Dn walk changed", d, "nodes")

	u := n.WalkUp("")
	log.Println("Up walk changed", u, "nodes")

	return d + u
}

func (n *Netlist) WalkDn(prefix string) (changed int) {
	// log.Printf("%s Walking down [NETL:%s] ACE:%v", prefix, n.Name, n.IsAce)
	prefix += "|   "

	for _, node := range n.Nodes {
		if node.IsAce {
			for _, rnode := range n.Links[node.Fullname()] {
				changed += n.PropDn(prefix+"|   ", rnode, node)
			}
		}
	}

	for _, subnet := range n.Subnets {
		changed += subnet.WalkDn(prefix)

		for _, output := range subnet.Outputs {
			if output.IsAce || !output.RpAce.AllUnset() {
				for _, rnode := range n.Links[output.Fullname()] {
					changed += n.PropDn(prefix+"|   ", rnode, output)
				}
			}
		}
	}

	return
}

func (n *Netlist) WalkUp(prefix string) (changed int) {
	// log.Printf("%s Walking up [NETL:%s] ACE:%v", prefix, n.Name, n.IsAce)
	prefix += "|   "

	for _, node := range n.Nodes {
		if node.IsAce {
			for _, lnode := range n.Rlinks[node.Fullname()] {
				changed += n.PropUp(prefix+"|   ", lnode, node)
			}
		}
	}

	for _, subnet := range n.Subnets {
		changed += subnet.WalkUp(prefix)

		for _, input := range subnet.Inputs {
			if input.IsAce || !input.WpAce.AllUnset() {
				for _, lnode := range n.Rlinks[input.Fullname()] {
					changed += n.PropUp(prefix+"|   ", lnode, input)
				}
			}
		}
	}

	return
}

func (netlist *Netlist) PropDn(prefix string, node *Node, ace *Node) (changed int) {
	if node.IsAce {
		return
	}

	q := queue.New()
	q.Push(node)

	for !q.Empty() {

		n := q.Pop().(*Node)

		// If this node is ACE, propagation stops here.
		if n.IsAce {
			continue
		}

		prev := n.RpAce.String()
		n.AddRpAce(ace)
		next := n.RpAce.String()

		// If the value is unchanged after update it means that this ACE value
		// was already propagated down through this node. Can terminate
		// propagation here. This logic should prevent cycles from causing
		// runaways.
		if prev == next {
			continue
		}

		changed++

		// If we've reached a port, we can terminate. This is because if we are
		// within a subnet, the parent will continue the walk from the subnet
		// outputs. If this is a parent this would be the end of the walk
		// anyway.
		if n.IsPort {
			continue
		}

		// Every node that this node feeds into needs to propagate down this
		// ACE node's values.
		for _, rnode := range netlist.Links[n.Fullname()] {
			q.Push(rnode)
		}
	}

	return
}

func (netlist *Netlist) PropUp(prefix string, node *Node, ace *Node) (changed int) {
	if node.IsAce {
		return
	}

	q := queue.New()
	q.Push(node)

	for !q.Empty() {

		n := q.Pop().(*Node)

		// If this node is ACE, propagation stops here.
		if n.IsAce {
			continue
		}

		prev := n.RpAce.String()
		n.AddRpAce(ace)
		next := n.RpAce.String()

		// If the value is unchanged after update it means that this ACE value
		// was already propagated up through this node. Can terminate
		// propagation here. This logic should prevent cycles from causing
		// runaways.
		if prev == next {
			continue
		}

		changed++

		// If we've reached a port, we can terminate. This is because if we are
		// within a subnet, the parent will continue the walk from the subnet
		// inputs. If this is a parent this would be the end of the walk
		// anyway.
		if n.IsPort {
			continue
		}

		// Every node that is connected to this node needs to propagate up this
		// ACE node's values.
		for _, lnode := range netlist.Rlinks[n.Fullname()] {
			q.Push(lnode)
		}
	}

	return
}

type NetStats struct {
	Nodes   int
	Ace     int
	Seqn    int
	EqnHist histogram.Histogram
}

func NewNetStats() NetStats {
	s := NetStats{
		EqnHist: histogram.New(),
	}
	return s
}

func (s NetStats) String() (str string) {
	return fmt.Sprintf("[Nodes:%d] [ACE:%d] [Seqn:%d]\n%v", s.Nodes,
		s.Ace, s.Seqn, s.EqnHist)
}

func (s *NetStats) Plus(addend NetStats) {
	s.Nodes += addend.Nodes
	s.Ace += addend.Ace
	s.Seqn += addend.Seqn
	s.EqnHist.Merge(addend.EqnHist)
}

func (n Netlist) Stats(acestructs []ace.AceStruct, level int) (stats NetStats) {
	stats = NewNetStats()

	stats.Nodes = len(n.Nodes)

	for _, node := range n.Nodes {
		if node.IsAce {
			stats.Ace++
		}

		if node.IsSeqn {
			stats.Seqn++
			eqn := ""
			for _, pos := range node.RpAce.Test() {
				eqn += fmt.Sprintf("%0.4f+", acestructs[pos].Rpavf)
			}
			eqn = strings.TrimSuffix(eqn, "+")

			// If no terms reached this node, it is a 1.0 sequential
			if eqn == "" {
				eqn = "1.0000"
			}

			stats.EqnHist.Add(eqn)
		}
	}

	for _, subnet := range n.Subnets {
		stats.Plus(subnet.Stats(acestructs, level+1))
	}

	if level < 2 {
		log.Println(n)
		log.Println(stats)
	}

	return
}
