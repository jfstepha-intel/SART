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
    kModule      // module
    EndModule    // endmodule
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
    input string
    start int
    pos   int
    width int
    line  int
    lpos  int
    items chan Item
}

func NewLexer(input string) (*lexer, chan Item) {
    l := &lexer {
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
    for r := l.next(); r != '\n'; r = l.next() {}
    l.line++
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

func isAlpha(r rune) bool {
    return strings.IndexRune(alpha, r) >= 0
}

func lexId(l *lexer) statefn {
    l.acceptRun(alnum)
    str := l.input[l.start:l.pos]
    switch str {
    case "module"      : l.emit(kModule)
    case "endmodule"   : l.emit(EndModule)
    default            : l.emit(Id)
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

        case r == '\n':
            l.line++
            l.lpos = 1
            l.ignore()

        case r == '\r':
            l.line++
            l.lpos = 1
            l.ignore()

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
