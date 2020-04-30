package lexer

import (
	"fmt"

	"github.com/chirino/graphql/internal/scanner"
	"github.com/chirino/graphql/qerrors"

	"strconv"
	"strings"
)

type syntaxError string
type Location = qerrors.Location

type Lexer struct {
	sc   *scanner.Scanner
	next rune
}

func NewLexer(s string) *Lexer {
	sc := &scanner.Scanner{}
	sc.Init(strings.NewReader(s))
	return &Lexer{sc: sc}
}

func (l *Lexer) CatchSyntaxError(f func()) (errRes error) {
	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(syntaxError); ok {
				errRes = qerrors.Errorf("syntax error: %s", err).WithLocations(l.Location())
				return
			}
			panic(err)
		}
	}()

	f()
	return
}

func (l *Lexer) Peek() rune {
	return l.next
}

// Consume whitespace and tokens equivalent to whitespace (e.g. commas and comments).
//
// Consumed comment characters will build the description for the next type or field encountered.
// The description is available from `DescComment()`, and will be reset every time `Consume()` is
// executed.
func (l *Lexer) Consume() {
	for {
		l.next = l.sc.Scan()
		if l.next == ',' {
			// Similar to white space and line terminators, commas (',') are used to improve the
			// legibility of source text and separate lexical tokens but are otherwise syntactically and
			// semantically insignificant within GraphQL documents.
			//
			// http://facebook.github.io/graphql/draft/#sec-Insignificant-Commas
			continue
		}
		break
	}
}

func (l *Lexer) ConsumeIdent() string {
	name := l.sc.TokenText()
	l.ConsumeToken(scanner.Ident)
	return name
}

func (l *Lexer) ConsumeIdentWithLoc() (string, Location) {
	loc := l.Location()
	name := l.sc.TokenText()
	l.ConsumeToken(scanner.Ident)
	return name, loc
}

func (l *Lexer) PeekKeyword(keyword string) bool {
	return l.next == scanner.Ident && l.sc.TokenText() == keyword
}

func (l *Lexer) ConsumeKeyword(keywords ...string) string {
	if l.next != scanner.Ident || !isOneOf(l.sc.TokenText(), keywords...) {
		l.SyntaxError(fmt.Sprintf("unexpected %q, expecting %q", l.sc.TokenText(), keywords))
	}
	result := l.sc.TokenText()
	l.Consume()
	return result
}

func isOneOf(one string, of ...string) bool {
	for _, v := range of {
		if one == v {
			return true
		}
	}
	return false
}

func (l *Lexer) ConsumeLiteral() string {
	switch l.next {
	case scanner.Int, scanner.Float, scanner.String, scanner.BlockString, scanner.Ident:
		lit := l.sc.TokenText()
		l.Consume()
		return lit
	default:
		l.SyntaxError(fmt.Sprintf("unexpected %q, expecting literal", l.next))
		panic("unreachable")
	}
}

func (l *Lexer) ConsumeToken(expected rune) {
	if l.next != expected {
		l.SyntaxError(fmt.Sprintf("unexpected %q, expecting %s", l.sc.TokenText(), scanner.TokenString(expected)))
	}
	l.Consume()
}

type ShowType byte

var PossibleDescription = ShowType(0)
var ShowStringDescription = ShowType(1)
var ShowBlockDescription = ShowType(2)
var NoDescription = ShowType(3)

type Description struct {
	ShowType ShowType
	Text     string
	Loc      Location
}

func (d Description) String() string {
	return d.Text
}

func (l *Lexer) ConsumeDescription() (d Description) {
	d.Loc = l.Location()
	if l.Peek() == scanner.String {
		d.ShowType = ShowStringDescription
		d.Text = l.ConsumeString()
	} else if l.Peek() == scanner.BlockString {
		d.ShowType = ShowBlockDescription
		text := l.sc.TokenText()
		text = text[3 : len(text)-3]
		l.ConsumeToken(scanner.BlockString)
		d.Text = text
	} else {
		d.ShowType = NoDescription
	}
	return
}

func (l *Lexer) ConsumeString() string {
	loc := l.Location()
	unquoted, err := strconv.Unquote(l.sc.TokenText())
	if err != nil {
		panic(fmt.Sprintf("Invalid string literal at %s: %s ", loc, err))
	}
	l.ConsumeToken(scanner.String)
	return unquoted
}

func (l *Lexer) SyntaxError(message string) {
	panic(syntaxError(message))
}

func (l *Lexer) Location() Location {
	return Location{
		Line:   l.sc.Line,
		Column: l.sc.Column,
	}
}
