package netlist

import (
    "log"
)

func (n *Netlist) WalkDown() {
    for name, node := range n.Inputs {
        log.Println(name, node.Fullname())
        for lfn, rnode := range n.Links[node.Fullname()] {
            log.Println("  ", lfn, rnode)
        }
    }
}
