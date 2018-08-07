package parsesp

// This file contains the datastructures and functions needed to parse the
// tokens of an instance specification. It is tricky enough to need its own
// state machine and datatypes.

type InstanceTokens []Item

type istatefn func(*parser, *[]string) istatefn

func saveiname(p *parser, inst *[]string) istatefn {
    // log.Println("saveiname", p.token)
    iname := p.token.val
    p.expect(Id)
    *inst = append(*inst, iname)
    switch {
        case p.tokenis(Id)      : return add2list
        case p.accept(Newline)  : return newline1
        default                 : return p.errorf("saveiname: %v line:%d",
                                                  p.token, p.l.line)
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
        default                 : return p.errorf("poplist: %v", p.token)
    }
}

func properties(p *parser, inst *[]string) istatefn {
    // log.Println("properties", p.token)
    prop := p.token.val
    p.expect(Property)
    *inst = append(*inst, prop)
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
