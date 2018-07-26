package parsesp

import (
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "os"
    // "strings"
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

    name := p.token.val
    m := rtl.NewModule(name)

    for p.accept(Id, Property) {
    }
    p.expect(Newline)

    for p.tokenis(Plus) {
        p.plusline()
    }

    p.portspec(m)
    p.portspec(m)
    p.portspec(m)

    for p.accept(Newline) {
    }

    if p.tokenis(Star) {
        p.comment()
    }

    for p.tokenis(Id) {
        p.instance()
    }

    p.expect(Ends)
    p.expect(Id)
    log.Println("subckt:", m.Name)
    m.Save()
}

func (p *parser) comment() {
    p.expect(Star)
    for p.accept(Id, Star, Colon, Number) {
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

func (p *parser) instance() {
    p.expect(Id) // instance name
    for p.accept(Id, Property) {
    }
    p.expect(Newline)

    for p.tokenis(Plus) {
        p.plusline()
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
