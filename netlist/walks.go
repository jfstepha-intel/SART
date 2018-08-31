package netlist

import (
    "log"
)

type AceTerms uint64

func (t *AceTerms) Add(a AceTerms) {
    *t |= a
}

func (n *Node) AddRpAce(a *Node) AceTerms {
    n.RpAce.Add(a.RpAce)
    return n.RpAce
}

func (n *Node) AddWpAce(a *Node) AceTerms {
    n.WpAce.Add(a.WpAce)
    return n.WpAce
}

func (n *Netlist) Walk() (changed int) {
    changed += n.WalkDn("")
    log.Println("Down walk changed", changed, "nodes")
    changed += n.WalkUp("")
    log.Println("Up walk changed", changed, "nodes")
    return
}

func (n *Netlist) WalkDn(prefix string) (changed int) {
    // log.Printf("%s Walking down [NETL:%s]", prefix, n.Name, n.IsAce)
    prefix += "|   "

    // log.Printf("%sPropagating inputs", prefix)
    for _, input := range n.Inputs {
        if input.RpAce != 0 {
            for _, rnode := range n.Links[input.Fullname()] {
                changed += n.PropDn(prefix+"|   ", rnode, input)
            }
        }
    }

    for _, subnet := range n.Subnets {
        // log.Printf("%sPropagating subnet %s", prefix, subnet.Name)
        if subnet.IsAce {
            for _, ace := range subnet.Outputs {
                if !ace.IsAce {
                    log.Fatal("error")
                }

                for _, rnode := range n.Links[ace.Fullname()] {
                    changed += n.PropDn(prefix+"|   ", rnode, ace)
                }
            }
        } else {
            changed += subnet.WalkDn(prefix)

            for _, output := range subnet.Outputs {
                if output.RpAce != 0 {
                    for _, rnode := range n.Links[output.Fullname()] {
                        changed += n.PropDn(prefix+"|   ", rnode, output)
                    }
                }
            }
        }
    }

    return
}

func (n *Netlist) WalkUp(prefix string) (changed int) {
    // log.Printf("%sWalking up [NETL:%s] ACE:%v", prefix, n.Name, n.IsAce)
    prefix += "|   "

    // log.Printf("%sPropagating outputs", prefix)
    for _, output := range n.Outputs {
        if output.WpAce != 0 {
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
                if input.WpAce != 0 {
                    for _, lnode := range n.Links[input.Fullname()] {
                        changed += n.PropUp(prefix+"|   ", lnode, input)
                    }
                }
            }
        }
    }

    return
}

func (n *Netlist) PropDn(prefix string, node *Node, ace *Node) (changed int) {
    // log.Printf("%sMarking %s with %s", prefix, node, ace)
    prev := node.RpAce
    next := node.AddRpAce(ace)

    if prev != next {
        changed++
    }

    if node.IsPort {
        // log.Printf("%swalk reached port %s", prefix, node)
        return
    }

    for _, rnode := range n.Links[node.Fullname()] {
        if rnode.IsAce {
            // log.Println("%sWalk reached ace %s", prefix, rnode)
            continue
        }
        changed += n.PropDn(prefix+"|   ", rnode, ace)
    }

    return
}

func (n *Netlist) PropUp(prefix string, node *Node, ace *Node) (changed int) {
    log.Printf("%sPropUp: Marking %s with %s", prefix, node, ace)
    prev := node.WpAce
    next := node.AddWpAce(ace)

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

func (n *Netlist) Stats(prefix string) {
    for _, node := range n.Nodes {
        log.Printf("%s%v", prefix, node)
    }
    // for _, subnet := range n.Subnets {
    //     subnet.Stats(prefix+"|   ")
    // }
}
