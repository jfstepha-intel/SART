package rtl

import (
    "fmt"
    "log"
)

// Module Node /////////////////////////////////////////////////////////////////

type Node struct {
    Parent string       `bson:"module"`
    Name   string       `bson:"name"`
    Type   string       `bson:"type"`
}

func NewNode(parent, name, typ string) *Node {
    p := &Node {
        Parent: parent,
        Name  : name,
        Type  : typ,
    }
    return p
}

// Instance ////////////////////////////////////////////////////////////////////

type Inst struct {
    Parent string       `bson:"module"`
    Name   string       `bson:"name"`
    Type   string       `bson:"type"`
    IsPrim bool         `bson:"isprim"`
    IsSeq  bool         `bson:"isseq"`
}

func NewInst(parent, iname, itype string) *Inst {
    i := &Inst {
        Parent: parent,
        Name  : iname,
        Type  : itype,
    }

    return i
}

// Instance connections ////////////////////////////////////////////////////////

type Conn struct {
    Parent   string     `bson:"module"`
    Iname    string     `bson:"iname"`
    Itype    string     `bson:"itype"`
    Formal   string     `bson:"formal"`
    Actual []string     `bson:"actual"`
    IsOut    bool       `bson:"isout"`
    IsPrim   bool       `bson:"isprim"`
}

func NewConn(parent, iname, itype, formal string, actual []string) *Conn {
    i := &Conn{
        Parent: parent,
        Iname : iname,
        Itype : itype,
        Formal: formal,
        Actual: actual,
    }
    return i
}

// Module //////////////////////////////////////////////////////////////////////

type Module struct {
    Name    string
    Nodes   map[string]*Node
    Insts   map[string]*Inst
    Conns   map[string][]*Conn
}

func NewModule(name string) *Module {
    m := &Module{
        Name : name,
        Nodes: make(map[string]*Node),
        Insts: make(map[string]*Inst),
        Conns: make(map[string][]*Conn),
    }
    return m
}

func (m *Module) AddNewNode(name, typ string, hi, lo int64) {
    // If hi and lo are both zero there was no range specified. So assume
    // unindexed single bit.
    if hi == 0 && lo == 0 {
        node := NewNode(m.Name, name, typ)
        m.AddNode(node)
    } else { // Otherwise emit one per index.
        for i := hi; i >= lo; i-- {
            newname := fmt.Sprintf("%s[%d]", name, i)
            node := NewNode(m.Name, newname, typ)
            m.AddNode(node)
        }
    }
}

func (m *Module) AddNewInst(iname, itype string) {
    inst := NewInst(m.Name, iname, itype)
    m.AddInst(inst)
}

func (m *Module) AddNewConn(iname, itype, formal string, actual []Signal) {
    var actuals []string

    // For each actual signal, if the hi or lo are non-zero, then emit names
    // with index suffixes.
    for _, a := range actual {
        if a.Hi == 0 && a.Lo == 0 {
            actuals = append(actuals, a.Name)
        } else {
            for i := a.Hi; i >= a.Lo; i -- {
                newname := fmt.Sprintf("%s[%d]", a.Name, i)
                actuals = append(actuals, newname)
            }
        }
    }

    conn := NewConn(m.Name, iname, itype, formal, actuals)
    m.AddConn(conn)
}

func (m *Module) AddNode(node *Node) {
    m.Nodes[node.Name] = node
}

func (m *Module) AddInst(inst *Inst) {
    m.Insts[inst.Name] = inst
}

func (m *Module) AddConn(conn *Conn) {
    m.Conns[conn.Iname] = append(m.Conns[conn.Iname], conn)
}

func (m Module) IsSeq(iname string) bool {
    if inst, ok := m.Insts[iname]; ok {
        return inst.IsSeq
    }
    log.Fatalf("No instance called %q in module %s", iname, m.Name)
    return false
}

func (m Module) NumNodes() (count int) {
    for range m.Nodes {
        count++
    }
    return
}

func (m Module) NumInsts() (count int) {
    for range m.Insts {
        count++
    }
    return
}

func (m Module) NumConns() (count int) {
    for range m.Conns {
        count++
    }
    return
}

// Signal //////////////////////////////////////////////////////////////////////

type Signal struct {
    Name string
    Hi, Lo int64
}
