package netlist

import (
	"fmt"
	"log"
	"sart/bitfield"
	"sart/rtl"
	"strings"
)

var DuplicateNode error = fmt.Errorf("-!- Duplicate Node Error")

////////////////////////////////////////////////////////////////////////////////

type Node struct {
	Parent string `bson:"module"`
	Name   string `bson:"name"`
	Type   string
	IsPort bool
	IsPrim bool
	IsSeqn bool
	IsWire bool
	IsAce  bool
	RpAce  *bitfield.BitField
	WpAce  *bitfield.BitField
}

func NewNode(parent, name, typ string, bfsize int) *Node {
	return &Node{
		Parent: parent,
		Name:   name,
		Type:   typ,
		RpAce:  bitfield.New(bfsize),
		WpAce:  bitfield.New(bfsize),
	}
}

func NewPortNode(parent, name, typ string, bfsize int) *Node {
	p := NewNode(parent, name, typ, bfsize)
	p.IsPort = true
	return p
}

func NewPrimNode(parent, name, typ string, bfsize int) *Node {
	p := NewNode(parent, name, typ, bfsize)
	p.IsPrim = true
	return p
}

func NewWireNode(parent, name string, bfsize int) *Node {
	w := NewNode(parent, name, "WIRE", bfsize)
	w.IsWire = true
	return w
}

func (n Node) String() (str string) {
	str += "["
	switch {
	case n.IsPrim:
		str += "PRIM "
	case n.IsPort:
		str += "PORT "
	case n.IsWire:
		str += "WIRE "
	}
	str += n.Fullname()
	if n.IsAce {
		str += " ACE"
	}
	str += "] "
	str += fmt.Sprintf("r:'%v' w:'%v'", n.RpAce, n.WpAce)
	return
}

func (n Node) Fullname() string {
	return n.Parent + "/" + n.Name
}

////////////////////////////////////////////////////////////////////////////////

type Netlist struct {
	Name    string
	IsAce   bool
	Ports   []*rtl.Port
	Nodes   map[string]*Node   // Holds all nodes
	Inputs  map[string]*Node   // Holds all nodes corresponding to input ports
	Inouts  map[string]*Node   // Holds all nodes corresponding to inout ports
	Outputs map[string]*Node   // Holds all nodes corresponding to output ports
	Links   map[string][]*Node // Map from left-node's fullname to right-nodes
	Rlinks  map[string][]*Node // Map from right-node's fullname to left-nodes
	Subnets map[string]*Netlist
}

func NewNetlist(name string) *Netlist {
	n := &Netlist{
		Name:    name,
		Nodes:   make(map[string]*Node),
		Inputs:  make(map[string]*Node),
		Inouts:  make(map[string]*Node),
		Outputs: make(map[string]*Node),
		Subnets: make(map[string]*Netlist),
		Links:   make(map[string][]*Node),
		Rlinks:  make(map[string][]*Node),
	}
	return n
}

func New(prefix, mname, iname string, bfsize, level int) *Netlist {
	m := rtl.LoadModule(mname)

	// log.Printf("%s%s [%v] ACE:%v", prefix, iname, m)

	n := NewNetlist(iname)

	// All ports trivially become nodes
	for pos, port := range m.OrderedPorts() {
		pname := port.Name

		p := NewPortNode(iname, pname, port.Type, bfsize)
		n.AddNode(p)

		// This list is needed only to look up the formal node at the time of
		// netlist construction. It will not be saved to mongo, nor will it be
		// populated in a Load()-ed netlist.
		nport := rtl.NewPort(iname, pname, pos)
		n.Ports = append(n.Ports, nport)
	}

	// Go through all connections -- actual names of all instance connections.
	// If a name has not already been encountered as a port, add it as a wire.
	for _, conns := range m.Conns {
		for _, conn := range conns {
			w := NewWireNode(iname, conn.Actual, bfsize)
			n.AddNode(w)
		}
	}

	// Go through all the instantiations. If primitive add a primitive node. If
	// a defined module, create a subnet for it an add it to the set of subnets
	// at this level.
	for nname, inst := range m.Insts {
		// log.Printf("Inst:%q Type:%s Prim:%v", nname, inst.Type, inst.IsPrim)
		fullname := iname + "/" + nname
		if inst.IsPrim {
			prim := NewPrimNode(iname, nname, inst.Type, bfsize)
			n.AddNode(prim)

			// Update whether or not this node is a sequential
			prim.IsSeqn = inst.IsSeq

			// Go through each connection of this instantiation, locate the
			// corresponding node and link it to this primitive node. By now
			// all signals should exist as either a port or a wire node.
			for _, c := range m.Conns[nname] {
				// This node will be indexed with a name that has the unique
				// prefix of this parent instance -- iname.
				nodename := iname + "/" + c.Actual
				if node, ok := n.Nodes[nodename]; !ok {
					log.Fatal("Could not locate actual node:", nodename)
				} else {
					// log.Printf("%sP: %v <-> %v", prefix, node, prim)
					switch c.Type {
					case "INPUT":
						n.Connect(node, prim)
					case "OUTPUT":
						n.Connect(prim, node)
					case "INOUT":
						n.Connect(node, prim)
						n.Connect(prim, node)
					default:
						log.Fatal("Unexpected conn type:", c.Type)
					}
				}
			}
		} else {
			subnet := New(prefix+"|  ", inst.Type, iname+"/"+inst.Name, bfsize, level+1)
			n.Subnets[fullname] = subnet

			for _, c := range m.Conns[nname] {
				// Locate actual node. This should be a node (port or wire) at
				// this level by now. It will be indexed at this level with a
				// name with the unique prefix of this parent instance -- iname
				aname := iname + "/" + c.Actual
				anode := n.Nodes[aname]
				if anode == nil {
					log.Fatal("Could not locate actual node:", aname)
				}

				// Locate formal node. This should be a port in the subnet at
				// the exact position as this connection's position. If node
				// cannot be located, abort rightaway -- something went wrong.
				if c.Pos >= len(subnet.Ports) {
					log.Fatalf("Seeking port position %d in subnet %v of netlist %v. Number of available ports: %d",
					c.Pos, subnet, n, len(subnet.Ports))
				}
				fname := fullname + "/" + subnet.Ports[c.Pos].Name
				if fnode, ok := subnet.Nodes[fname]; !ok {
					log.Fatal("Could not locate formal node", fname)
				} else {
					// log.Printf("%sS: %v <-> %v", prefix, anode, fnode)
					switch c.Type {
					case "INPUT":
						n.Connect(anode, fnode)
					case "OUTPUT":
						n.Connect(fnode, anode)
					case "INOUT":
						n.Connect(anode, fnode)
						n.Connect(fnode, anode)
					default:
						log.Fatal("Unexpected conn type:", c.Type)
					}
				}
			}
		}
	}

	n.Save()
	log.Printf("Done (%d) %q", level, mname)

	return n
}

// Connect adds a link between two nodes in the netlist. Only the fullname of
// the node is saved beacuse the node can be looked up easily with that name if
// needed.
func (n *Netlist) Connect(l *Node, r *Node) {
	n.Links[l.Fullname()] = append(n.Links[l.Fullname()], r)
	n.Rlinks[r.Fullname()] = append(n.Rlinks[l.Fullname()], l)
}

func (n *Netlist) AddNode(node *Node) {
	fullname := node.Fullname()
	if _, found := n.Nodes[fullname]; found {
		if node.IsWire {
			// Wires with same name will be discovered multiple times but we
			// should not add duplicate nodes for each encouter. Simply return.
			return
		}
		log.Output(2, fmt.Sprintf("Node %s exists: %q", node, fullname))
		log.Fatal(DuplicateNode)
	}

	n.Nodes[fullname] = node

	// Additionally update port maps
	switch node.Type {
	case "INPUT":
		n.Inputs[fullname] = node
	case "INOUT":
		n.Inouts[fullname] = node
	case "OUTPUT":
		n.Outputs[fullname] = node
	}

	n.IsAce = node.IsAce
}

func (n *Netlist) LocateNode(name string) *Node {
	var node *Node
	var subnet *Netlist
	var found bool

	// If a node with this name exists at this level, it can be readily found
	// in Nodes. Return it.
	if node, found = n.Nodes[name]; found {
		return node
	}

	// If not found at this level, it could be a port in a subnet. Assuming the
	// name is a full name, derive the subnet name and port name, then search.
	parts := strings.Split(name, "/")
	subnetname := strings.Join(parts[0:len(parts)-1], "/")

	// If there is no subnet with this name, no node was found
	if subnet, found = n.Subnets[subnetname]; !found {
		return nil
	}

	// To be found, the name should match a node in this subnet
	if node, found = subnet.Nodes[name]; found {
		return node
	}

	return nil
}

func (n Netlist) String() (str string) {
	str += fmt.Sprintf("nl:%q Nodes:%d Ports:%d Prims:%d Seqns:%d Wires:%d Subnets:%d Links:%d",
		n.Name, n.NumNodes(), n.NumPorts(), n.NumPrims(),
		n.NumSeqns(), n.NumWires(), n.NumSubnets(), n.NumLinks())
	return
}

func (n Netlist) HasPort(signal string) bool {
	if node, ok := n.Nodes[signal]; ok {
		return node.IsPort
	}
	return false
}

func (n Netlist) HasWire(signal string) bool {
	if _, ok := n.Nodes[signal]; ok {
		return true
	}
	return false
}

func (n Netlist) NumNodes() (count int) {
	for range n.Nodes {
		count++
	}
	return
}

func (n Netlist) NumPorts() (count int) {
	for _, node := range n.Nodes {
		if node.IsPort {
			count++
		}
	}
	return
}

func (n Netlist) NumPrims() (count int) {
	for _, node := range n.Nodes {
		if node.IsPrim {
			count++
		}
	}
	return
}

func (n Netlist) NumSeqns() (count int) {
	for _, node := range n.Nodes {
		if node.IsSeqn {
			count++
		}
	}
	return
}

func (n Netlist) NumWires() (count int) {
	for _, node := range n.Nodes {
		if node.IsWire {
			count++
		}
	}
	return
}

func (n Netlist) NumSubnets() (count int) {
	for range n.Subnets {
		count++
	}
	return
}

func (n Netlist) NumLinks() (count int) {
	for range n.Links {
		count++
	}
	return
}
