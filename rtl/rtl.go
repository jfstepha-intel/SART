package rtl

import (
    "fmt"
    // "log"
    "regexp"
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
    IsSeq    bool       `bson:"isseq"`
    IsOut    bool       `bson:"isout"`
    IsInp    bool       `bson:"isinp"`
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
    i.SetSeq()
    i.SetDir()
    return i
}

func (i *Inst) SetPrim() {
    i.IsPrim = strings.HasPrefix(i.Type, "sncclnt_ec0")
}

func (i *Inst) SetSeq() {
    i.IsSeq = strings.HasPrefix(i.Type, "sncclnt_ec0f") ||
              strings.HasPrefix(i.Type, "sncclnt_ec0l")
}

var odigits = regexp.MustCompile(`o\d*`)

func (i *Inst) SetDir() {
    if i.IsPrim {
        // Decipher from name only if this is a primitive.
        switch {
            case odigits.MatchString(i.Formal): i.IsOut = true
        }
        i.IsInp = !i.IsOut
    }
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

func (m *Module) AddNewInst(name, typ, formal string, actual []Signal) {
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

    i := NewInst(m.Name, name, typ, formal, actuals)
    m.AddInst(i)
}

func (m *Module) AddNode(node *Node) {
    m.Nodes[node.Name] = node
}

func (m *Module) AddInst(inst *Inst) {
    m.Insts = append(m.Insts, inst)
}

// Signal //////////////////////////////////////////////////////////////////////

type Signal struct {
    Name string
    Hi, Lo int64
}
