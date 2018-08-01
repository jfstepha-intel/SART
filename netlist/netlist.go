package netlist

import ( 
    "fmt"
    "log"
    "sart/rtl"
)        

////////////////////////////////////////////////////////////////////////////////

type Node struct {
    Parent  string      `bson:"module"`
    Name    string      `bson:"name"`
    Type    string
    IsPort  bool
    IsPrim  bool
    IsSeqn  bool
    IsWire  bool
}

func NewNode(parent, name, typ string) *Node {
    return &Node{
        Parent: parent,
        Name  : name,
        Type  : typ,
    }
}

func NewPortNode(parent, name, typ string) *Node {
    p := NewNode(parent, name, typ)
    p.IsPort = true
    return p
}

func NewPrimNode(parent, name, typ string) *Node {
    p := NewNode(parent, name, typ)
    p.IsPrim = true
    return p
}

func NewWireNode(parent, name string) *Node {
    w := NewNode(parent, name, "WIRE")
    w.IsWire = true
    return w
}

func (n Node) String() (str string) {
    str += "["
    switch {
        case n.IsPrim: str += "PRIM "
        case n.IsPort: str += "PORT "
        case n.IsWire: str += "WIRE "
    }
    str += n.Name + "]"
    return
}

////////////////////////////////////////////////////////////////////////////////

type Subnet struct {
    Parent string   `bson:"module"`
    Name   string   `bson:"name"`
}

type Link struct {
    Parent string   `bson:"module"`
    L      string
    R      string
}

////////////////////////////////////////////////////////////////////////////////

type Netlist struct {
    Name    string
    Ports   []*rtl.Port
    Nodes   map[string]*Node    // Holds all nodes
    Inputs  map[string]*Node    // Holds all nodes corresponding to input ports
    Inouts  map[string]*Node    // Holds all nodes corresponding to inout ports
    Outputs map[string]*Node    // Holds all nodes corresponding to output ports
    Subnets map[string]*Netlist
    Links   []Link
}

func NewNetlist(name string) *Netlist {
    n := &Netlist {
        Name    : name,
        Nodes   : make(map[string]*Node),
        Inputs  : make(map[string]*Node),
        Inouts  : make(map[string]*Node),
        Outputs : make(map[string]*Node),
        Subnets : make(map[string]*Netlist),
    }
    return n
}

func New(prefix, mname, iname string) *Netlist {
    m := rtl.LoadModule(mname)

    log.Printf("%s%s [%v]", prefix, iname, m)

    n := NewNetlist(iname)

    // All ports trivially become nodes
    for pos, port := range m.OrderedPorts() {
        pname := port.Name

        p := NewPortNode(iname, pname, port.Type)
        fullname := iname + "/" + pname
        n.Nodes[fullname] = p

        switch p.Type {
            case "INPUT" : n.Inputs[fullname]  = p
            case "INOUT" : n.Inouts[fullname]  = p
            case "OUTPUT": n.Outputs[fullname] = p
            default      : log.Fatal("Unexpected port type:", p.Type)
        }

        nport := rtl.NewPort(iname, pname, pos)
        n.Ports = append(n.Ports, nport)
    }

    // Go through all connections -- actual names of all instance connections.
    // If a name has not already been encountered as a port, add it as a wire.
    for _, conns := range m.Conns {
        for _, conn := range conns {
            signal := conn.Actual
            fullname := iname + "/" + signal
            if !n.HasPort(fullname) && !n.HasWire(fullname) {
                w := NewWireNode(iname, signal)
                n.Nodes[fullname] = w
            }
        }
    }

    // Go through all the instantiations. If primitive add a primitive node. If
    // a defined module, create a subnet for it an add it to the set of subnets
    // at this level.
    for nname, inst := range m.Insts {
        // log.Printf("Inst:%q Type:%s Prim:%v", nname, inst.Type, inst.IsPrim)
        fullname := iname + "/" + nname
        if inst.IsPrim {
            prim := NewPrimNode(iname, nname, inst.Type)
            n.Nodes[fullname] = prim

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
                    n.Connect(node, prim)
                }
            }
        } else {
            subnet := New(prefix+"|  ", inst.Type, iname+"/"+inst.Name)
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
                fname := fullname + "/" + subnet.Ports[c.Pos].Name
                if fnode, ok := subnet.Nodes[fname]; !ok {
                    log.Fatal("Could not locate formal node", fname)
                } else {
                    // log.Printf("%sS: %v <-> %v", prefix, anode, fnode)
                    n.Connect(anode, fnode)
                }
            }
        }
    }

    n.Save()

    return n
}

// Connect adds a link between two nodes in the netlist. Only the fullname of
// the node is saved beacuse the node can be looked up easily with that name if
// needed.
func (n *Netlist) Connect(l *Node, r *Node) {
    link := Link {
        Parent: n.Name,
        L     : l.Parent + "/" + l.Name,
        R     : r.Parent + "/" + r.Name,
    }
    n.Links = append(n.Links, link)
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
    return len(n.Ports)
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
    return len(n.Links)
}
