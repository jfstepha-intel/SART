package rtl

import (
    "fmt"
    // "log"
    "strings"
    // "gopkg.in/mgo.v2/bson"
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

// Instance connections ////////////////////////////////////////////////////////

type Inst struct {
    Parent   string     `bson:"module"`
    Name     string     `bson:"name"`
    Type     string     `bson:"type"`
    Formal   string     `bson:"formal"`
    Actual []string     `bson:"actual"`
    IsPrim   bool       `bson:"isprim"`
}

func NewInst(parent, name, typ, formal string, actual []string) *Inst {
    i := &Inst{
        Parent: parent,
        Name  : name,
        Type  : typ,
        Formal: formal,
        Actual: actual,
    }
    i.SetPrim()
    return i
}

func (i *Inst) SetPrim() {
    i.IsPrim = strings.HasPrefix(i.Type, "sncclnt_ec0")
}

// Module //////////////////////////////////////////////////////////////////////

type Module struct {
    Name    string
    Nodes   map[string]*Node
    Insts   []*Inst
}

func NewModule(name string) *Module {
    m := &Module{
        Name : name,
        Nodes: make(map[string]*Node),
    }
    return m
}

func (m *Module) AddNewNode(name, typ string, hi, lo int64) {
    // If hi and lo are both zero there was no range specified. So assume
    // unindexed single bit.
    if hi == 0 && lo == 0 {
        n := NewNode(m.Name, name, typ)
        m.AddNode(n)
    } else { // Otherwise emit one per index.
        for i := hi; i >= lo; i-- {
            newname := fmt.Sprintf("%s[%d]", name, i)
            n := NewNode(m.Name, newname, typ)
            m.AddNode(n)
        }
    }
}

func (m *Module) AddNewInst(name, typ, formal string, actual []string) {
    i := NewInst(m.Name, name, typ, formal, actual)
    m.AddInst(i)
}

func (m *Module) AddNode(node *Node) {
    m.Nodes[node.Name] = node
}

func (m *Module) AddInst(inst *Inst) {
    m.Insts = append(m.Insts, inst)
}
