package rtl

import (
    // "log"
    // "gopkg.in/mgo.v2/bson"
)

type connmap   map[string][]string

type Node struct {
    Parent string `bson:"module"`
    Name   string `bson:"name"`
    Type   string `bson:"type"`
    Width  int    `bson:"width"`
}

type Inst struct {
    Parent string           `bson:"module"`
    Name   string           `bson:"name"`
    Type   string           `bson:"type"`
    Formal string           `bson:"formal"`
    Actual []string         `bson:"actual"`
}

type Module struct {
    Name    string
    Nodes   map[string]*Node
    Insts   []*Inst
}

func New(name string) *Module {
    m := &Module{
        Name : name,
        Nodes: make(map[string]*Node),
    }
    return m
}

func (m *Module) AddNode(name, typ string, width int) {
    // log.Printf("%s: Adding port %s(type:%s)", m.Name, name, typ)
    p := &Node {
        Parent: m.Name,
        Name  : name,
        Type  : typ,
        Width : width,
    }
    m.Nodes[name] = p
}

func (m *Module) AddInst(name, typ, formal string, actual []string) {
    // log.Printf("%s: Adding instance %s(type:%s)", m.Name, name, typ)
    i := &Inst{
        Parent: m.Name,
        Name  : name,
        Type  : typ,
        Formal: formal,
        Actual: actual,
    }
    m.Insts = append(m.Insts, i)
}
