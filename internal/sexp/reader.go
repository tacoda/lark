package sexp

import (
	"fmt"
	"strconv"
	"strings"
)

// ReadError describes a syntax error with 1-based line and column.
type ReadError struct {
	Msg  string
	Line int
	Col  int
}

func (e *ReadError) Error() string {
	return fmt.Sprintf("line %d, col %d: %s", e.Line, e.Col, e.Msg)
}

// Parse reads exactly one datum from src. Trailing non-whitespace is an
// error, so Parse is the inverse of Write.
func Parse(src string) (Value, error) {
	r := newReader(src)
	if err := r.skipSpace(); err != nil {
		return nil, err
	}
	if r.peek() == 0 {
		return nil, r.errf("empty input")
	}
	v, err := r.datum()
	if err != nil {
		return nil, err
	}
	if err := r.skipSpace(); err != nil {
		return nil, err
	}
	if r.peek() != 0 {
		return nil, r.errf("trailing input after datum")
	}
	return v, nil
}

// ParseAll reads every datum in src, in order.
func ParseAll(src string) ([]Value, error) {
	r := newReader(src)
	var vals []Value
	for {
		if err := r.skipSpace(); err != nil {
			return nil, err
		}
		if r.peek() == 0 {
			return vals, nil
		}
		v, err := r.datum()
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
}

type reader struct {
	src  string
	pos  int
	line int
	col  int
}

func newReader(src string) *reader { return &reader{src: src, line: 1, col: 1} }

func (r *reader) errf(format string, args ...any) *ReadError {
	return &ReadError{Msg: fmt.Sprintf(format, args...), Line: r.line, Col: r.col}
}

func (r *reader) peek() byte {
	if r.pos >= len(r.src) {
		return 0
	}
	return r.src[r.pos]
}

func (r *reader) peekAt(n int) byte {
	if r.pos+n >= len(r.src) {
		return 0
	}
	return r.src[r.pos+n]
}

func (r *reader) next() byte {
	c := r.peek()
	if c != 0 {
		r.pos++
		if c == '\n' {
			r.line++
			r.col = 1
		} else {
			r.col++
		}
	}
	return c
}

// isDelim reports whether c terminates an atom token.
func isDelim(c byte) bool {
	switch c {
	case 0, ' ', '\t', '\n', '\r', '(', ')', '\'', '`', ',', '"', ';':
		return true
	}
	return false
}

// skipSpace consumes whitespace, ; line comments, and nestable #| |# block
// comments.
func (r *reader) skipSpace() error {
	for {
		c := r.peek()
		switch {
		case c == 0:
			return nil
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			r.next()
		case c == ';':
			for r.peek() != 0 && r.peek() != '\n' {
				r.next()
			}
		case c == '#' && r.peekAt(1) == '|':
			r.next()
			r.next()
			depth := 1
			for depth > 0 {
				if r.peek() == 0 {
					return r.errf("unterminated block comment")
				}
				if r.peek() == '#' && r.peekAt(1) == '|' {
					r.next()
					r.next()
					depth++
				} else if r.peek() == '|' && r.peekAt(1) == '#' {
					r.next()
					r.next()
					depth--
				} else {
					r.next()
				}
			}
		default:
			return nil
		}
	}
}

func (r *reader) datum() (Value, error) {
	if err := r.skipSpace(); err != nil {
		return nil, err
	}
	c := r.peek()
	switch {
	case c == 0:
		return nil, r.errf("unexpected end of input")
	case c == '(':
		r.next()
		return r.list()
	case c == ')':
		return nil, r.errf("unexpected )")
	case c == '\'':
		r.next()
		return r.sugar("quote")
	case c == '`':
		r.next()
		return r.sugar("quasiquote")
	case c == ',':
		r.next()
		if r.peek() == '@' {
			r.next()
			return r.sugar("unquote-splicing")
		}
		return r.sugar("unquote")
	case c == '"':
		return r.str()
	case c == '#':
		return r.hash()
	default:
		return r.atom()
	}
}

// sugar wraps the next datum: 'x -> (quote x), etc.
func (r *reader) sugar(name string) (Value, error) {
	d, err := r.datum()
	if err != nil {
		return nil, err
	}
	return List(Sym(name), d), nil
}

func (r *reader) list() (Value, error) {
	var elems []Value
	for {
		if err := r.skipSpace(); err != nil {
			return nil, err
		}
		c := r.peek()
		if c == 0 {
			return nil, r.errf("unterminated list")
		}
		if c == ')' {
			r.next()
			return ListTail(NilVal, elems...), nil
		}
		if c == '.' && isDelim(r.peekAt(1)) {
			if len(elems) == 0 {
				return nil, r.errf("unexpected .")
			}
			r.next() // consume .
			tail, err := r.datum()
			if err != nil {
				return nil, err
			}
			if err := r.skipSpace(); err != nil {
				return nil, err
			}
			if r.peek() != ')' {
				return nil, r.errf("expected ) after dotted tail")
			}
			r.next()
			return ListTail(tail, elems...), nil
		}
		d, err := r.datum()
		if err != nil {
			return nil, err
		}
		elems = append(elems, d)
	}
}

func (r *reader) hash() (Value, error) {
	switch r.peekAt(1) {
	case 't':
		if isDelim(r.peekAt(2)) {
			r.next()
			r.next()
			return Bool(true), nil
		}
	case 'f':
		if isDelim(r.peekAt(2)) {
			r.next()
			r.next()
			return Bool(false), nil
		}
	case '(':
		r.next()
		r.next()
		var elems []Value
		for {
			if err := r.skipSpace(); err != nil {
				return nil, err
			}
			c := r.peek()
			if c == 0 {
				return nil, r.errf("unterminated vector")
			}
			if c == ')' {
				r.next()
				return &Vec{Elems: elems}, nil
			}
			d, err := r.datum()
			if err != nil {
				return nil, err
			}
			elems = append(elems, d)
		}
	}
	return nil, r.errf("unsupported # syntax")
}

func (r *reader) str() (Value, error) {
	r.next() // opening "
	var b strings.Builder
	for {
		c := r.peek()
		if c == 0 {
			return nil, r.errf("unterminated string")
		}
		if c == '"' {
			r.next()
			return Str(b.String()), nil
		}
		if c == '\\' {
			r.next()
			e := r.peek()
			if e == 0 {
				return nil, r.errf("unterminated string escape")
			}
			r.next()
			switch e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			case '0':
				b.WriteByte(0)
			default:
				return nil, r.errf("unknown escape \\%c", e)
			}
			continue
		}
		b.WriteByte(r.next())
	}
}

func (r *reader) atom() (Value, error) {
	start := r.pos
	for !isDelim(r.peek()) {
		r.next()
	}
	tok := r.src[start:r.pos]
	if tok == "" {
		return nil, r.errf("unexpected character %q", r.peek())
	}
	if n, ok := parseNumber(tok); ok {
		return n, nil
	}
	return Sym(tok), nil
}

// parseNumber classifies a token as Int or Float; ok=false means symbol.
func parseNumber(tok string) (Value, bool) {
	if i, err := strconv.ParseInt(tok, 10, 64); err == nil {
		return Int(i), true
	}
	if len(tok) > 2 && (tok[:2] == "0x" || tok[:2] == "0X") {
		if i, err := strconv.ParseInt(tok[2:], 16, 64); err == nil {
			return Int(i), true
		}
	}
	if looksNumeric(tok) {
		if f, err := strconv.ParseFloat(tok, 64); err == nil {
			return Float(f), true
		}
	}
	return nil, false
}

func looksNumeric(tok string) bool {
	for i := 0; i < len(tok); i++ {
		if c := tok[i]; c == '.' || c == 'e' || c == 'E' {
			return true
		}
	}
	switch tok {
	case "Inf", "+Inf", "-Inf", "NaN", "nan", "+nan", "-nan":
		return true
	}
	return false
}
