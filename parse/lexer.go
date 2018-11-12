package parse

import (
    "fmt"
    "strings"
    "unicode/utf8"
)

const eof = -(iota + 1)

const (
    Error ItemType = iota
    EOF
    Slash        // /
    LParen       // (
    RParen       // )
    LBrack       // [
    RBrack       // ]
    LBrace       // {
    RBrace       // }
    Comma        // ,
    Semicolon    // ;
    Colon        // :
    Dot          // .
    Equals       // =
    kModule      // module
    EndModule    // endmodule
    Input        // input
    Inout        // inout
    Output       // output
    Wire         // wire
    Supply0      // supply0
    Assign       // assign
    Number       // 1234
    ConstBits    // 1'b1
    Id           // Identifier
)

type ItemType int

type Item struct {
    typ ItemType
    val string
}

func (i Item) String() string {
    switch i.typ {
        case EOF:
            return "EOF"
        case Error:
            return i.val
    }
    return fmt.Sprintf("%q", i.val)
}

type statefn func(*lexer) statefn

type lexer struct {
    name  string
    input string
    start int
    pos   int
    width int
    line  int
    lpos  int
    items chan Item
}

func NewLexer(name, input string) (*lexer, chan Item) {
    l := &lexer {
        name : name,
        input: input,
        line : 1,
        items: make(chan Item),
    }

    go l.run()

    return l, l.items
}

func (l *lexer) run() {
    for state := lexText; state != nil; {
        state = state(l)
    }
    close(l.items)
}

func (l *lexer) emit(t ItemType)  {
    l.items <- Item{t, l.input[l.start:l.pos]}
    l.start = l.pos
}

func (l *lexer) next() (r rune) {
    if l.pos >= len(l.input) {
        l.width = 0
        return eof
    }
    r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
    l.pos += l.width
    l.lpos++
    return r
}

func (l *lexer) backup() {
    l.pos -= l.width
}

func (l *lexer) ignore() {
    l.start = l.pos
}

func (l *lexer) peek() rune {
    r := l.next()
    l.backup()
    return r
}

func (l *lexer) accept(valid string) bool {
    if strings.IndexRune(valid, l.next()) >= 0 {
        return true
    }
    l.backup()
    return false
}

func (l *lexer) acceptRun(valid string) {
    for strings.IndexRune(valid, l.next()) >= 0 {}
    l.backup()
}

func (l *lexer) errorf(format string, args ...interface{}) statefn {
    l.items <- Item {
        typ: Error,
        val: fmt.Sprintf(format, args...),
    }
    return nil
}

func lexLineComment(l *lexer) statefn {
    l.accept("/")
    l.accept("/")
    for r := l.next(); r != '\n'; r = l.next() {
        l.ignore()
    }
    l.line++
    l.ignore()
    return lexText
}

func lexSlash(l *lexer) statefn {
    l.accept("/")
    switch l.next() {
    // case '*':
    //     l.backup(); l.backup();
    //     return lexBlockComment
    case '/':
        l.backup(); l.backup();
        return lexLineComment
    default:
        l.backup()
        l.emit(Slash)
    }
    return lexText
}

const alpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_"
const alnum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
const digit = "0123456789"
const bit   = "01"
const hex   = "abcdefABCDEF"

func isDigit(r rune) bool {
    return strings.IndexRune(digit, r) >= 0
}

func isAlpha(r rune) bool {
    return strings.IndexRune(alpha, r) >= 0
}

func lexEscId(l *lexer) statefn {
    if l.next() == '\\' {
        l.ignore()
    }
    for r := l.next(); r != ' '; r = l.next() {
        if r == '\n' {
            l.line++
            l.lpos = 1
        }
    }
    // put the ' ' back, it is not part of the identifier
    l.backup()
    l.emit(Id)
    return lexText
}

func lexId(l *lexer) statefn {
    l.acceptRun(alnum)
    str := l.input[l.start:l.pos]
    switch str {
    case "module"      : l.emit(kModule)
    case "endmodule"   : l.emit(EndModule)
    case "input"       : l.emit(Input)
    case "inout"       : l.emit(Inout)
    case "output"      : l.emit(Output)
    case "wire"        : l.emit(Wire)
    case "supply0"     : l.emit(Supply0)
    case "assign"      : l.emit(Assign)
    default            : l.emit(Id)
    }
    return lexText
}

func lexNumber(l *lexer) statefn {
    l.acceptRun(digit)

    if l.accept("'") {
        // prefixes for binary, decimal, hex no idea what 's' is for
        // l.accept("bdhsH")
        // l.acceptRun(digit + hex + "_x?")
        l.accept("b")
        l.acceptRun(digit)

        // // a word 'b00001111' maybe split like 'b0000 1111'
        // for l.accept(" ") {
        //     l.acceptRun(digit)
        // }
        l.emit(ConstBits)
    } else {
     l.emit(Number)
    }

    return lexText
}

func lexText(l *lexer) statefn {
    for {
        r := l.next()
        if r == eof { break }
        switch {
        case r == '/':
            l.backup()
            return lexSlash

        case r == ' ': l.ignore()
        case r == '\t': l.ignore()

        case r == '\n':
            l.line++
            l.lpos = 1
            l.ignore()

        case r == '\r':
            l.line++
            l.lpos = 1
            l.ignore()

        case r == '(': l.emit(LParen)
        case r == ')': l.emit(RParen)
        case r == '[': l.emit(LBrack)
        case r == ']': l.emit(RBrack)
        case r == '{': l.emit(LBrace)
        case r == '}': l.emit(RBrace)
        case r == ',': l.emit(Comma)
        case r == ';': l.emit(Semicolon)
        case r == ':': l.emit(Colon)
        case r == '.': l.emit(Dot)
        case r == '=': l.emit(Equals)

        case r == '\\':
            l.backup()
            return lexEscId

        case isDigit(r):
            l.backup()
            return lexNumber

        case isAlpha(r):
            l.backup()
            return lexId

        default:
            return l.errorf("Don't know what to do with %q %c %x at line:%d", r, r, r, l.line)
        }

    }
    l.emit(EOF)
    return nil
} 
