package rtl

import (
    "fmt"
    "log"
)

// Module Port /////////////////////////////////////////////////////////////////

type Port struct {
    Parent string       `bson:"module"`
    Name   string       `bson:"name"`
    Type   string       `bson:"type"`
    Pos    int          `bson:"pos"`
}

func NewPort(parent, name string, pos int) *Port {
    p := &Port {
        Parent: parent,
        Name  : name,
        Type  : "",
        Pos   : pos,
    }
    return p
}

func (p *Port) SetType(typ string) {
    p.Type = typ
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
    Ports   map[string]*Port
    Insts   map[string]*Inst
    Conns   map[string][]*Conn
}

func NewModule(name string) *Module {
    m := &Module{
        Name : name,
        Ports: make(map[string]*Port),
        Insts: make(map[string]*Inst),
        Conns: make(map[string][]*Conn),
    }
    return m
}

// When a new port is discovered and added, the port type is not known yet. Use
// the SetPortType method to set it when it becomes available.
func (m *Module) AddNewPort(name string, pos int) {
    port := NewPort(m.Name, name, pos)
    m.AddPort(port)
}

func (m Module) SetPortType(name, typ string) {
    if _, ok := m.Ports[name]; !ok {
        log.Fatalln("Unknown port:", name)
    }
    m.Ports[name].SetType(typ)
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

func (m *Module) AddPort(port *Port) {
    m.Ports[port.Name] = port
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

func (m Module) NumPorts() (count int) {
    for range m.Ports {
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
