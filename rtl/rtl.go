package rtl

import (
    // "log"
    // "gopkg.in/mgo.v2/bson"
)

type connmap   map[string][]string

type Node struct {
    Parent string       `bson:"module"`
    Name   string       `bson:"name"`
    Type   string       `bson:"type"`
    Width  int          `bson:"width"`
}

type Inst struct {
    Parent   string     `bson:"module"`
    Name     string     `bson:"name"`
    Type     string     `bson:"type"`
    Formal   string     `bson:"formal"`
    Actual []string     `bson:"actual"`
}

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

func NewNode(parent, name, typ string, width int) *Node {
    p := &Node {
        Parent: parent,
        Name  : name,
        Type  : typ,
        Width : width,
    }
    return p
}

func NewInst(parent, name, typ, formal string, actual []string) *Inst {
    i := &Inst{
        Parent: parent,
        Name  : name,
        Type  : typ,
        Formal: formal,
        Actual: actual,
    }
    return i
}

func (m *Module) AddNewNode(name, typ string, width int) {
    n := NewNode(m.Name, name, typ, width)
    m.AddNode(n)
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
