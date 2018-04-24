package parse

import (
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "os"
    "strconv"

    "sart/rtl"
)

var UnknownToken = fmt.Errorf("Unknown token")

type parser struct {
    l      *lexer
    token  Item
    tokens chan Item
}

func New(name string, r io.Reader) { 
    bytes, err := ioutil.ReadAll(r)
    if err != nil {
        log.Fatal(err)
    }

    parser := &parser{}
    parser.l, parser.tokens = NewLexer(name, string(bytes))
    
    // Load first token
    parser.next()

    parser.statements()
}

// next advances a token
func (p *parser) next() {
    p.token = <- p.tokens
}

func (p *parser) tokenis(types ...ItemType) bool {
    for _, t := range types {
        if p.token.typ == t {
            return true
        }
    }
    return false
}

func (p *parser) expect(types ...ItemType) {
    if p.tokenis(types...) {
        p.next()
        return
    }
    log.Output(2,
               fmt.Sprintf("Expecting %v but got %v at line %d, pos %d.",
                           types, p.token, p.l.line, p.l.lpos))
    os.Exit(1)
}

func (p *parser) accept(types ...ItemType) bool {
    if p.tokenis(types...) {
        p.next()
        return true
    }
    return false
}

func (p parser) stop(err error) {
    log.Println(p.l.name)
    log.Printf("token: %v, line: %d", p.token, p.l.line)
    log.Fatalf(err.Error())
}

// productions /////////////////////////////////////////////////////////////////

func (p *parser) statements() {
    for {
        switch {
        case p.tokenis(kModule):
            p.module_decl()

        case p.tokenis(EOF):
            return

        default: p.stop(UnknownToken)
        }
    }
}

func (p *parser) module_decl() {
    p.expect(kModule)

    name := p.token.val
    p.expect(Id)
    m := rtl.New(name)

    if p.accept(LParen) {
        p.list_of_ports(m)
        p.expect(RParen)
    }

    p.expect(Semicolon)

    for !p.tokenis(EndModule) {
        p.module_item(m)
    }

    p.expect(EndModule)
    m.Save()
    log.Println(m.Name)
}

func (p *parser) list_of_ports(m *rtl.Module) {
    if p.tokenis(RParen) { // empty list of ports
        return
    }
    pname := p.token.val
    p.expect(Id)
    m.AddPort(pname, "")
    for p.accept(Comma) {
        pname := p.token.val
        p.expect(Id)
        m.AddPort(pname, "")
    }
}

func (p *parser) module_item(m *rtl.Module) {
    switch {
    // module items can be input/output/wire declarations
    case p.tokenis(Wire, Input, Inout, Output):
        p.net_decl(m)
        return

    // supply0 vss;
    case p.accept(Supply0):
        p.expect(Id)
        p.expect(Semicolon)
        return
    }

    itype := p.token.val
    p.expect(Id)

    iname := p.token.val
    p.expect(Id)

    m.AddInst(iname, itype)

    p.expect(LParen)
    p.instance_connections(m, iname)
    p.expect(RParen)
    p.expect(Semicolon)
}

func (p *parser) net_decl(m *rtl.Module) {
    typ := p.token.val
    p.expect(Wire, Input, Inout, Output)

    validrange := false
    var hi, lo int64
    if p.tokenis(LBrack) {
        hi, lo = p.bitrange()
        validrange = true
    }

    width := 1
    if validrange {
        // width = hi - lo + 1; [1:0] => 2 bits wide
        width += int(hi-lo)
    }

    name := p.token.val
    p.expect(Id)
    if typ == "wire" {
        m.AddSignal(name, width)
    } else {
        m.SetPortType(name, typ)
        m.SetPortWidth(name, width)
    }

    for p.accept(Comma) {
        name := p.token.val
        p.expect(Id)
        if typ == "wire" {
            m.AddSignal(name, width)
        } else {
            m.SetPortType(name, typ)
            m.SetPortWidth(name, width)
        }
    }

    p.expect(Semicolon)
}

func (p *parser) bitrange() (hi, lo int64) {
    p.expect(LBrack)
    hs := p.token.val
    p.expect(Number)

    p.expect(Colon)

    ls := p.token.val
    p.expect(Number)
    p.expect(RBrack)

    var err error

    hi, err = strconv.ParseInt(hs, 10, 64)
    if err != nil {
        p.stop(err)
    }

    lo, err = strconv.ParseInt(ls, 10, 64)
    if err != nil {
        p.stop(err)
    }

    return
}

func (p *parser) instance_connections(m *rtl.Module, iname string) {
    // Connections can be empty
    if p.tokenis(RParen) {
        return
    }

    p.instance_connection(m, iname)
    for p.accept(Comma) {
        p.instance_connection(m, iname)
    }
}

func (p *parser) instance_connection(m *rtl.Module, iname string) {
    p.expect(Dot)

    formal := p.token.val
    p.expect(Id)

    p.expect(LParen)

    actual := []string{}

    if p.accept(LBrace) {
        actual = p.list_of_primary()
        p.expect(RBrace)
    } else {
        actual = append(actual, p.primary())
    }

    p.expect(RParen)
    m.AddInstConn(iname, formal, actual...)
}

func (p *parser) primary() (str string) {
    if p.tokenis(RParen) { // empty primary expression
        return
    }

    str = p.token.val
    p.expect(Id)

    // Pick up a subsequent index or bitrange as well
    if p.accept(LBrack) {
        str += "["
        n := p.token.val
        p.expect(Number)
        str += n
        if p.accept(Colon) {
            n := p.token.val
            p.expect(Number)
            str += ":" + n
        }
        p.expect(RBrack)
        str += "]"
    }

    return
}

func (p *parser) list_of_primary() (prims []string) {
    prim := p.primary()
    if prim != "" {
        prims = append(prims, prim)
    }
    for p.accept(Comma) {
        prim := p.primary()
        if prim != "" {
            prims = append(prims, prim)
        }
    }
    return
}
