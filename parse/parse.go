package parse

import (
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "os"
)

type Parser struct {
    l      *lexer
    token  Item
    tokens chan Item
}

func New(r io.Reader) { 
    bytes, err := ioutil.ReadAll(r)
    if err != nil {
        log.Fatal(err)
    }

    parser := &Parser{}
    parser.l, parser.tokens = NewLexer(string(bytes))
    
    // Load first token
    parser.Next()
    log.Println(parser.token)
}

// Next advances a token
func (p *Parser) Next() {
    p.token = <- p.tokens
}

func (p *Parser) TokenIs(types ...ItemType) bool {
    for _, t := range types {
        if p.token.typ == t {
            return true
        }
    }
    return false
}

func (p *Parser) Expect(types ...ItemType) {
    if p.TokenIs(types...) {
        p.Next()
        return
    }
    log.Output(2,
               fmt.Sprintf("Expecting %q but got %v (%v) at line %d, pos %d.",
                           types, p.token, p.token.typ, p.l.line, p.l.lpos))
    os.Exit(1)
}

func (p *Parser) Accept(types ...ItemType) bool {
    if p.TokenIs(types...) {
        p.Next()
        return true
    }
    return false
}

