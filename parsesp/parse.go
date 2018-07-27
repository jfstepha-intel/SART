package parsesp

import (
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "os"
    "strings"
    "sart/rtl"
)

var UnknownToken = fmt.Errorf("Unknown token")
var BadState = fmt.Errorf("Bad State")

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

    log.Println(len(bytes))

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
               fmt.Sprintf("Expecting %v but got %v at line %d, pos %d. (%s)",
                           types, p.token, p.l.line, p.l.lpos, p.l.name))
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
        // Ignore whitespace
        case p.accept(Newline):
        case p.tokenis(Star)  : p.comment()
        case p.tokenis(Global): p.global()
        case p.tokenis(Subckt): p.subckt()
        case p.tokenis(EOF)   : return
        default               : p.stop(UnknownToken)
        }
    }
}

func (p *parser) global() {
    p.expect(Global)
    p.expect(Id)
    p.expect(Newline)
}

func (p *parser) subckt() {
    p.expect(Subckt)

    // If the name of the subckt is too long, it could bet bumped down to a
    // newline starting with Plus.
    if p.accept(Newline) {
        p.expect(Plus)
    }

    // The first identifier is the name of the subckt.
    name := p.token.val
    m := rtl.NewModule(name)

    // Subsequent identifiers till the newline are ports.
    for p.accept(Id, Property) {
    }
    p.expect(Newline)

    // If there are more ports, the subsequent lines with port names will start
    // with a Plus.
    for p.tokenis(Plus) {
        p.plusline()
    }

    // INPUT, OUTPUT and INOUT
    p.portspec(m)
    p.portspec(m)
    p.portspec(m)

    // There usually are newlines after this; ignore.
    for p.accept(Newline) {
    }

    // There usually are comments or line delimitters around here; ignore.
    if p.tokenis(Star) {
        p.comment()
    }

    // Next will be instantiations of other subckts. Those lines will start
    // with identifiers.
    for p.tokenis(Id) {
        p.instance(m)
    }

    // Watch for the .ENDS directive followed by the name of the subckt.
    p.expect(Ends)
    p.expect(Id)

    log.Println("subckt:", m.Name)
    m.Save()
}

func (p *parser) comment() {
    p.expect(Star)
    // Ignore everything until newline
    for !p.tokenis(Newline) {
        p.next()
    }
    p.expect(Newline)
}

func (p *parser) portspec(m *rtl.Module) {
    p.accept(Star)

    signal_type := p.token.val
    p.expect(Input, Inout, Output)

    p.expect(Colon)

    for p.tokenis(Id) {
        signal_name := p.token.val
        m.AddNewWire(signal_name, signal_type, 0, 0)
        p.expect(Id)
    }

    p.expect(Newline)

    for p.accept(Star) {
        if p.tokenis(Plus) {
            ids := p.plusline()
            for _, signal_name := range ids {
                m.AddNewWire(signal_name, signal_type, 0, 0)
            }
        }
    }
}

func (p *parser) instance(m *rtl.Module) {
    // log.Println(p.token)
    payload := []string{}
    for state := saveiname; state != nil; {
        state = state(p, &payload)
    }
    iname := payload[0]
    itype := payload[len(payload)-1]

    if strings.HasPrefix(iname, "X") {
        m.AddNewInst(iname, itype)
    }
}

func (p *parser) plusline() (ids []string) {
    p.expect(Plus)
    for p.tokenis(Id, Property) {
        ids = append(ids, p.token.val)
        p.expect(Id, Property)
    }
    p.expect(Newline)
    return
}

////////////////////////////////////////////////////////////////////////////////

type istatefn func(*parser, *[]string) istatefn

func (p *parser) errorf(format string, args ...interface{}) istatefn {
    log.Output(2, fmt.Sprintf(format, args...))
    log.Fatal(BadState)
    return nil
}

func saveiname(p *parser, inst *[]string) istatefn {
    // log.Println("saveiname", p.token)
    iname := p.token.val
    p.expect(Id)
    *inst = append(*inst, iname)
    switch {
        case p.tokenis(Id)      : return add2list
        case p.accept(Newline)  : return newline1
        default                 : return p.errorf("saveiname: %v", p.token)
    }
}

func add2list(p *parser, inst *[]string) istatefn {
    // log.Println("add2list", p.token)
    actual := p.token.val
    p.expect(Id)
    *inst = append(*inst, actual)
    switch {
        case p.tokenis(Id)      : return add2list
        case p.tokenis(Property): return properties
        case p.accept(Newline)  : return newline1
        default                 : return p.errorf("add2list: %v", p.token)
    }
}

func newline1(p *parser, inst *[]string) istatefn {
    // log.Println("newline1", p.token)
    switch {
        case p.accept(Plus)     : return idorprop
        default                 : return poplist
    }
}

func idorprop(p *parser, inst *[]string) istatefn {
    // log.Println("idorprop", p.token)
    switch {
        case p.tokenis(Id)      : return add2list
        case p.tokenis(Property): return properties
        default                 : return p.errorf("idorprop: %v", p.token)
    }
}

func poplist(p *parser, inst *[]string) istatefn {
    // log.Println("poplist", p.token)
    switch {
        case p.tokenis(Id, Ends): return success
        case p.tokenis(Property): return properties
        default                 : return p.errorf("idorprop: %v", p.token)
    }
}

func properties(p *parser, inst *[]string) istatefn {
    // log.Println("properties", p.token)
    p.expect(Property)
    switch {
        case p.tokenis(Property): return properties
        case p.accept(Newline)  : return newline2
        default                 : return p.errorf("properties: %v", p.token)
    }
}

func newline2(p *parser, inst *[]string) istatefn {
    // log.Println("newline2", p.token)
    switch {
        case p.accept(Plus)     : return newline3
        default                 : return success
    }
}

func newline3(p *parser, inst *[]string) istatefn {
    // log.Println("newline3", p.token)
    switch {
        case p.tokenis(Property): return properties
        default                 : return p.errorf("newline3: %v", p.token)
    }
}

func success(p *parser, inst *[]string) istatefn {
    return nil
}
