package parsesp

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

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
	p.token = <-p.tokens
	if p.tokenis(Error) {
		log.Output(2, fmt.Sprintf("%v", p.token))
		os.Exit(1)
	}
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
	log.Output(2, fmt.Sprintf("token: %v, line: %d", p.token, p.l.line))
	log.Fatalf(err.Error())
}

// productions /////////////////////////////////////////////////////////////////

func (p *parser) statements() {
	for {
		switch {
		// Ignore whitespace
		case p.accept(Newline):
		case p.accept(End):
		case p.tokenis(Param):
			p.param()
		case p.tokenis(Global):
			p.global()
		case p.tokenis(Subckt):
			p.subckt()
		case p.tokenis(EOF):
			return
		default:
			p.stop(UnknownToken)
		}
	}
}

func (p *parser) global() {
	p.expect(Global)
	p.expect(Id)
	p.expect(Newline)
}

func (p *parser) param() {
	p.expect(Param)
	p.expect(Property)
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
	for p.tokenis(Input, Output, Inout) {
		p.portspec(m)
	}

	for p.tokenis(Connect) {
		p.connect()
	}

	// Next will be instantiations of other subckts. Those lines will start
	// with identifiers.
	for p.tokenis(Id) {
		p.instance(m)
	}

	// Watch for the .ENDS directive followed by the name of the subckt.
	lno := p.l.line
	p.expect(Ends)
	p.accept(Id)

	log.Printf("line: %d subckt: %s", lno, m.Name)
	m.Save()
}

func (p *parser) portspec(m *rtl.Module) {
	signal_type := p.token.val
	p.expect(Input, Inout, Output)

	p.expect(Colon)

	for p.tokenis(Id) {
		signal_name := p.token.val
		m.SetPortType(signal_name, signal_type)
		p.expect(Id)
	}

	for p.accept(Newline) {
	}

	for p.tokenis(Plus) {
		ids := p.plusline()
		for _, signal_name := range ids {
			m.SetPortType(signal_name, signal_type)
		}
	}

	for p.accept(Newline) {
	}
}

func (p *parser) connect() {
	p.expect(Connect)
	p.expect(Id)
	p.expect(Id)
	for p.accept(Newline) {
	}
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
