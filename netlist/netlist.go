package netlist

import ( 
    "fmt"
    "log"
    "sart/rtl"
)        

////////////////////////////////////////////////////////////////////////////////

type Node struct {
    Parent  string
    Name    string
    Type    string
    IsPort  bool
    IsPrim  bool
    IsSeqn  bool
    IsWire  bool
    R       []*Node
    L       []*Node
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

func (n *Node) ConnectRigt(l *Node) {
    n.R = append(n.R, l)
}

func (n *Node) ConnectLeft(r *Node) {
    n.L = append(n.L, r)
}

////////////////////////////////////////////////////////////////////////////////

type Port rtl.Port

////////////////////////////////////////////////////////////////////////////////

type Netlist struct {
    Name    string
    Ports   []*rtl.Port
    Nodes   map[string]*Node
    Subnets map[string]*Netlist
}

func NewNetlist(name string) *Netlist {
    n := &Netlist {
        Name    : name,
        Nodes   : make(map[string]*Node),
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
                    n.Link(node, prim)
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
                    n.Link(anode, fnode)
                }
            }
        }
    }

    n.Save()

    return n
}

func (n *Netlist) Link(l *Node, r *Node) {
    l.ConnectRigt(r)
    r.ConnectLeft(l)
}

func (n Netlist) String() (str string) {
    str += fmt.Sprintf("nl:%q Nodes:%d Ports:%d Prims:%d Seqns:%d Wires:%d Subnets:%d",
                       n.Name, n.NumNodes(), n.NumPorts(), n.NumPrims(),
                       n.NumSeqns(), n.NumWires(), n.NumSubnets())
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
