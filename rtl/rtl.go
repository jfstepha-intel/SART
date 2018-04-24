package rtl

import (
    "log"
    // "gopkg.in/mgo.v2/bson"
)

type portmap   map[string]*Port
type signalmap map[string]*Signal
type instmap   map[string]*Inst
type connmap   map[string][]string

type Port struct {
    Name  string
    Type  string
    Width int
}

type Signal struct {
    Name  string
    Width int
}

type Inst struct {
    Name string
    Type string
    Conn connmap
}

type Module struct {
    Name    string
    Ports   []Port
    Signals []Signal
    Insts   []Inst
    pmap    portmap
    smap    signalmap
    imap    instmap
}

func New(name string) *Module {
    m := &Module{
        Name: name,
        pmap: make(portmap),
        smap: make(signalmap),
        imap: make(instmap),
    }
    return m
}

func (m *Module) AddPort(name, typ string) {
    // log.Printf("%s: Adding port %s(type:%s)", m.Name, name, typ)
    p := Port{name, typ, 1}
    m.Ports = append(m.Ports, p)
    m.pmap[name] = &p
}

func (m *Module) SetPortType(name, typ string) {
    if _, ok := m.pmap[name]; !ok {
        log.Fatalf("%s: Unknown port %s", m.Name, name)
    }
    m.pmap[name].Type = typ
    // log.Printf("%s: Setting type of port %s to %s", m.Name, name, typ)
}

func (m *Module) SetPortWidth(name string, width int) {
    if _, ok := m.pmap[name]; !ok {
        log.Fatalf("%s: Unknown port %s", m.Name, name)
    }
    m.pmap[name].Width = width
    // log.Printf("%s: Setting width of port %s to %d", m.Name, name, width)
}

func (m *Module) AddSignal(name string, width int) {
    // log.Printf("%s: Adding signal %s(width:%d)", m.Name, name, width)
    s := Signal{name, width}
    m.Signals = append(m.Signals, s)
    m.smap[name] = &s
}

func (m *Module) AddInst(name, typ string) {
    // log.Printf("%s: Adding instance %s(type:%s)", m.Name, name, typ)
    i := Inst{
        Name: name,
        Type: typ,
        Conn: make(connmap),
    }
    m.Insts = append(m.Insts, i)
    m.imap[name] = &i
}

func (m *Module) AddInstConn(name, formal string, actual ...string) {
    if _, ok := m.imap[name]; !ok {
        log.Fatalf("%s: Unknown instance %s", name)
    }
    m.imap[name].Conn[formal] = append(m.imap[name].Conn[formal], actual...)
    // log.Printf("%s: Adding instance connection for %s (%s <- %v)", m.Name, name, formal, actual)
}
