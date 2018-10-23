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

func (n *Netlist) ResetWalked() {
	for _, node := range n.Nodes {
		node.Walked = false
	}

	for _, subnet := range n.Subnets {
		subnet.ResetWalked()
	}
}

func (n *Netlist) Walk() (changed int) {
	n.ResetWalked()
	changed += n.WalkDn("")
	log.Println("Dn walk changed", changed, "nodes")

	// n.ResetWalked()
	// changed += n.WalkUp("")
	// log.Println("Up walk changed", changed, "nodes")

	return
}

func (n *Netlist) WalkDn(prefix string) (changed int) {
	// log.Printf("%s Walking down [NETL:%s] ACE:%v", prefix, n.Name, n.IsAce)
	prefix += "|   "

	for _, node := range n.Nodes {
		if node.IsAce {
			for _, rnode := range n.Links[node.Fullname()] {
				changed += n.PropDn(prefix+"|   ", rnode, node)
			}
			// log.Printf("ACE node updated %d nodes %v", changed, node)
		}
	}

	//// // log.Printf("%sPropagating inputs", prefix)
	//// for _, input := range n.Inputs {
	//// 	if !input.RpAce.AllUnset() && !input.IsAce {
	//// 		for _, rnode := range n.Links[input.Fullname()] {
	//// 			changed += n.PropDn(prefix+"|   ", rnode, input)
	//// 		}
	//// 	}
	//// }

	for _, subnet := range n.Subnets {
		changed += subnet.WalkDn(prefix)

		for _, output := range subnet.Outputs {
			if output.IsAce || !output.RpAce.AllUnset() {
				for _, rnode := range n.Links[output.Fullname()] {
					changed += n.PropDn(prefix+"|   ", rnode, output)
				}
			}
		}

		//// // log.Printf("%sPropagating subnet %s", prefix, subnet.Name)
		//// if subnet.IsAce {
		//// 	for _, ace := range subnet.Outputs {
		//// 		if !ace.IsAce {
		//// 			log.Fatal("error")
		//// 		}

		//// 		for _, rnode := range n.Links[ace.Fullname()] {
		//// 			changed += n.PropDn(prefix+"|   ", rnode, ace)
		//// 		}
		//// 	}
		//// } else {
		//// 	changed += subnet.WalkDn(prefix)

		//// 	for _, output := range subnet.Outputs {
		//// 		if !output.RpAce.AllUnset() {
		//// 			for _, rnode := range n.Links[output.Fullname()] {
		//// 				changed += n.PropDn(prefix+"|   ", rnode, output)
		//// 			}
		//// 		}
		//// 	}
		//// }
	}

	return
}

func (n *Netlist) WalkUp(prefix string) (changed int) {
	// log.Printf("%sWalking up [NETL:%s] ACE:%v", prefix, n.Name, n.IsAce)
	prefix += "|   "

	// log.Printf("%sPropagating outputs", prefix)
	for _, output := range n.Outputs {
		if !output.WpAce.AllUnset() {
			for _, lnode := range n.Links[output.Fullname()] {
				changed += n.PropUp(prefix+"|   ", lnode, output)
			}
		}
	}

	for _, subnet := range n.Subnets {
		// log.Printf("%sPropagating subnet %s ACE:%v", prefix, subnet.Name, subnet.IsAce)
		if subnet.IsAce {
			for _, ace := range subnet.Inputs {
				if !ace.IsAce {
					log.Fatal("error")
				}

				for _, lnode := range n.Links[ace.Fullname()] {
					log.Println("Got here")
					changed += n.PropUp(prefix+"|   ", lnode, ace)
				}
			}
		} else {
			changed += subnet.WalkUp(prefix)

			for _, input := range subnet.Inputs {
				if !input.WpAce.AllUnset() {
					for _, lnode := range n.Links[input.Fullname()] {
						changed += n.PropUp(prefix+"|   ", lnode, input)
					}
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
		// propagation here.  This logic should prevent cycles from causing
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

func (n *Netlist) PropUp(prefix string, node *Node, ace *Node) (changed int) {
	if node.Walked {
		return
	}
	node.Walked = true

	// log.Printf("%sPropUp: Marking %s with %s", prefix, node, ace)
	prev := node.WpAce.String()
	node.AddWpAce(ace)
	next := node.WpAce.String()

	if prev != next {
		changed++
	}

	if node.IsPort {
		log.Printf("%sPropUp: Walk reached port %s", prefix, node)
		return
	}

	for _, lnode := range n.Links[node.Fullname()] {
		if lnode.IsAce {
			log.Println("%sPropUp: Walk reached ace %s", prefix, lnode)
			continue
		}
		changed += n.PropUp(prefix+"|   ", lnode, ace)
	}

	return
}

type NetStats struct {
	Nodes int
	Ace   int
	Seqn  int
	Hist  histogram.Histogram
}

func NewNetStats() NetStats {
	s := NetStats{
		Hist: histogram.New(),
	}
	return s
}

func (s NetStats) String() (str string) {
	return fmt.Sprintf("[Nodes:%d] [ACE:%d] [Seqn:%d]\n%v", s.Nodes,
		s.Ace, s.Seqn, s.Hist)
}

func (s *NetStats) Plus(addend NetStats) {
	s.Nodes += addend.Nodes
	s.Ace += addend.Ace
	s.Seqn += addend.Seqn
	s.Hist.Merge(addend.Hist)
}

func (n *Netlist) Stats(acestructs []ace.AceStruct) (stats NetStats) {
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
				eqn += fmt.Sprintf("%0.2f+", acestructs[pos].Rpavf)
			}
			eqn = strings.TrimSuffix(eqn, "+")

			// If no terms reached this node, it is a 1.0 sequential
			if eqn == "" {
				eqn = "1.0"
			}

			stats.Hist.Add(eqn)
		}
	}

	for _, subnet := range n.Subnets {
		stats.Plus(subnet.Stats(acestructs))
	}

	return
}
