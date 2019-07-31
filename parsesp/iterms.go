package parsesp

// This file contains the datastructures and functions needed to parse the
// tokens of an instance specification. It is tricky enough to need its own
// state machine and datatypes.

import (
	"log"
)

type InstanceTokens []Item

func (i *InstanceTokens) Add(token Item) {
	(*i) = append((*i), token)
}

func (i *InstanceTokens) PopFirst() Item {
	item := (*i)[0]
	(*i) = (*i)[1:]
	return item
}

func (i InstanceTokens) Last() Item {
	item := i[len(i)-1]
	return item
}

func (i *InstanceTokens) PopLast() Item {
	item := i.Last()
	(*i) = (*i)[:len((*i))-1]
	return item
}

func (i InstanceTokens) Resolve() (iname, itype string, actuals, props []string) {
	// The first token is the instance name.
	first := i.PopFirst()
	if first.typ != Id {
		log.Fatalln("Expecting Id for iname. Got:", first)
	}
	iname = first.val

	for _, token := range i {
		switch token.typ {
		case Property:
			props = append(props, token.val)
		case Id:
			fallthrough
		case Number:
			actuals = append(actuals, token.val)
		default:
			log.Fatalln("Expecting Id/Number for actual signal. Got:", token)
		}
	}
	return
}

type istatefn func(*parser, *InstanceTokens) istatefn

func saveiname(p *parser, inst *InstanceTokens) istatefn {
	// log.Println("saveiname", p.token)
	iname := p.token
	p.expect(Id)
	inst.Add(iname)
	switch {
	case p.tokenis(Number):
		return add2list
	case p.tokenis(Id):
		return add2list
	case p.accept(Newline):
		return newline1
	default:
		return p.errorf("saveiname: %v line:%d",
			p.token, p.l.line)
	}
}

func add2list(p *parser, inst *InstanceTokens) istatefn {
	// log.Println("add2list", p.token)
	actual := p.token
	p.expect(Id, Number, Property)
	inst.Add(actual)
	switch {
	case p.tokenis(Id, Number, Property):
		return add2list
	case p.accept(Newline):
		return newline1
	default:
		return p.errorf("add2list: %v", p.token)
	}
}

func newline1(p *parser, inst *InstanceTokens) istatefn {
	// log.Println("newline1", p.token)
	switch {
	case p.accept(Plus):
		return idorprop
	default:
		return poplist
	}
}

func idorprop(p *parser, inst *InstanceTokens) istatefn {
	// log.Println("idorprop", p.token)
	switch {
	case p.tokenis(Id):
		return add2list
	case p.tokenis(Property):
		return add2list
	default:
		return p.errorf("idorprop: %v", p.token)
	}
}

func poplist(p *parser, inst *InstanceTokens) istatefn {
	// log.Println("poplist", p.token)
	switch {
	case p.tokenis(Id, Ends):
		return success
	default:
		return p.errorf("poplist: %v", p.token)
	}
}

func properties(p *parser, inst *InstanceTokens) istatefn {
	// log.Println("properties", p.token)
	prop := p.token
	p.expect(Property)
	inst.Add(prop)
	switch {
	case p.tokenis(Property):
		return properties
	case p.accept(Newline):
		return newline2
	default:
		return p.errorf("properties: %v", p.token)
	}
}

func newline2(p *parser, inst *InstanceTokens) istatefn {
	// log.Println("newline2", p.token)
	switch {
	case p.accept(Plus):
		return newline3
	default:
		return success
	}
}

func newline3(p *parser, inst *InstanceTokens) istatefn {
	// log.Println("newline3", p.token)
	switch {
	case p.tokenis(Property):
		return properties
	default:
		return p.errorf("newline3: %v", p.token)
	}
}

func success(p *parser, inst *InstanceTokens) istatefn {
	return nil
}
