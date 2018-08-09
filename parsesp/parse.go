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
    p.expect(Id)

    m := rtl.NewModule(name)
    portpos := 0

    // Subsequent identifiers till the newline are ports.
    for p.tokenis(Id) {
        portname := p.token.val
        p.expect(Id)
        m.AddNewPort(portname, portpos)
        portpos++
    }
    for p.accept(Property) {
    }
    p.expect(Newline)

    // If there are more ports, the subsequent lines with port names will start
    // with a Plus.
    for p.tokenis(Plus) {
        ports := p.plusline()

        for _, portname := range ports {
            m.AddNewPort(portname, portpos)
            portpos++
        }
    }

    // INPUT, OUTPUT and INOUT
    p.portspec(m)
    p.portspec(m)
    p.portspec(m)

    done := false
    for !done {
        switch {
            // There usually are newlines here; ignore any number.
            case p.accept(Newline):

            // There usually are comments or line delimitters, too; ignore.
            case p.tokenis(Star): p.comment()

            // Connect statements.
            // TODO need to decide whether to capture this for SART.
            case p.tokenis(Connect): p.connect()

            // Stop after handling any number of the above situations.
            default: done = true
        }
    }

    // Next will be instantiations of other subckts. Those lines will start
    // with identifiers.
    for p.tokenis(Id) {
        p.instance(m)
    }

    // Watch for the .ENDS directive followed by the name of the subckt.
    lno := p.l.line
    p.expect(Ends)
    p.expect(Id)

    log.Printf("line: %d subckt: %s", lno, m.Name)
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
        m.SetPortType(signal_name, signal_type)
        p.expect(Id)
    }

    p.expect(Newline)

    for p.accept(Star) {
        if p.tokenis(Plus) {
            ids := p.plusline()
            for _, signal_name := range ids {
                m.SetPortType(signal_name, signal_type)
            }
        }
    }
}

func (p *parser) connect() {
    p.expect(Connect)
    p.expect(Id)
    p.expect(Id)
}

func (p *parser) instance(m *rtl.Module) {
    // log.Println(p.token)
    payload := &InstanceTokens{}
    for state := saveiname; state != nil; {
        state = state(p, payload)
    }

    iname, itype, actuals, props := payload.Resolve()

    m.AddNewInst(iname, itype)

    for pos, actual := range actuals {
        m.AddNewConn(iname, itype, actual, pos)
    }

    for _, prop := range props {
        m.AddNewProp(iname, itype, prop)
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

func (p *parser) errorf(format string, args ...interface{}) istatefn {
    log.Output(2, fmt.Sprintf(format, args...))
    log.Fatal(BadState)
    return nil
}
