package rtl

import (
    "log"
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
    Name string
    Type string
    Conn map[string][]string
}

type Module struct {
    Name    string
    Nodes   map[string]*Node
    Insts   map[string]*Inst
}

func New(name string) *Module {
    m := &Module{
        Name : name,
        Nodes: make(map[string]*Node),
        Insts: make(map[string]*Inst),
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

func (m *Module) AddInst(name, typ string) {
    // log.Printf("%s: Adding instance %s(type:%s)", m.Name, name, typ)
    i := &Inst{
        Name: name,
        Type: typ,
        Conn: make(map[string][]string),
    }
    m.Insts[name] = i
}

func (m *Module) AddInstConn(name, formal string, actual ...string) {
    if _, ok := m.Insts[name]; !ok {
        log.Fatalf("%s: Unknown instance %s", name)
    }
    m.Insts[name].Conn[formal] = append(m.Insts[name].Conn[formal], actual...)
    // log.Printf("%s: Adding instance connection for %s (%s <- %v)", m.Name, name, formal, actual)
}
