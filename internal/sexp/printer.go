package sexp

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// Width is the line width the pretty printer targets.
const Width = 80

// Write returns the canonical single-line form of v. It is lossless:
// Parse(Write(v)) is Equal to v. Quote sugar is never used here; the pretty
// printer may use it.
func Write(v Value) string {
	var b strings.Builder
	write(&b, v)
	return b.String()
}

// Fprint writes the canonical form of v to w.
func Fprint(w io.Writer, v Value) error {
	_, err := io.WriteString(w, Write(v))
	return err
}

// Pretty returns v formatted for humans, breaking lines at Width columns and
// using quote sugar. It is a fixed point: Pretty(Parse(Pretty(v))) ==
// Pretty(v), so emitted code is stable across runs and diffs cleanly.
func Pretty(v Value) string {
	var b strings.Builder
	pretty(&b, v, 0)
	return b.String()
}

// Pprint writes the pretty form of v to w.
func Pprint(w io.Writer, v Value) error {
	_, err := io.WriteString(w, Pretty(v))
	return err
}

func write(b *strings.Builder, v Value) {
	switch v := v.(type) {
	case Nil:
		b.WriteString("()")
	case Bool:
		if v {
			b.WriteString("#t")
		} else {
			b.WriteString("#f")
		}
	case Int:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case Float:
		b.WriteString(formatFloat(float64(v)))
	case Str:
		writeString(b, string(v))
	case Sym:
		b.WriteString(string(v))
	case *Pair:
		writePair(b, v)
	case *Vec:
		b.WriteString("#(")
		for i, e := range v.Elems {
			if i > 0 {
				b.WriteByte(' ')
			}
			write(b, e)
		}
		b.WriteByte(')')
	default:
		if s, ok := v.(fmt.Stringer); ok {
			b.WriteString(s.String())
		} else {
			fmt.Fprintf(b, "%v", v)
		}
	}
}

func writePair(b *strings.Builder, p *Pair) {
	elems, tail := Split(p)
	b.WriteByte('(')
	for i, e := range elems {
		if i > 0 {
			b.WriteByte(' ')
		}
		write(b, e)
	}
	if _, ok := tail.(Nil); !ok {
		b.WriteString(" . ")
		write(b, tail)
	}
	b.WriteByte(')')
}

// formatFloat round-trips through the reader: integral floats get a ".0"
// so they re-read as floats, and Inf/NaN use literal forms.
func formatFloat(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "+Inf"
	}
	if math.IsInf(f, -1) {
		return "-Inf"
	}
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		s += ".0"
	}
	return s
}

func writeString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case 0:
			b.WriteString(`\0`)
		default:
			b.WriteByte(s[i])
		}
	}
	b.WriteByte('"')
}

// sugarHeads maps a head symbol to its reader sugar for the pretty printer.
var sugarHeads = map[string]string{
	"quote":            "'",
	"quasiquote":       "`",
	"unquote":          ",",
	"unquote-splicing": ",@",
}

// headerInline lists special forms whose first argument stays on the head
// line when the list breaks: (define (f x) ...), (lambda (x) ...), (if c ...).
var headerInline = map[string]bool{
	"define": true, "defmacro": true, "lambda": true,
	"let": true, "let*": true, "letrec": true, "if": true,
}

func fits(v Value, col int) bool {
	s := Write(v)
	return !strings.Contains(s, "\n") && col+len(s) <= Width
}

func pretty(b *strings.Builder, v Value, col int) {
	if fits(v, col) {
		b.WriteString(Write(v))
		return
	}
	switch v := v.(type) {
	case *Pair:
		prettyPair(b, v, col)
	case *Vec:
		prettyVec(b, v, col)
	default:
		b.WriteString(Write(v))
	}
}

func prettyPair(b *strings.Builder, p *Pair, col int) {
	elems, tail := Split(p)
	head := elems[0]

	// Quote sugar: '(a b c), `(...), ,x, ,@xs — only for exact 2-lists.
	if s, ok := head.(Sym); ok && len(elems) == 2 {
		if pre, ok := sugarHeads[string(s)]; ok {
			b.WriteString(pre)
			pretty(b, elems[1], col+len(pre))
			return
		}
	}

	b.WriteByte('(')
	pretty(b, head, col+1)
	rest := elems[1:]
	first := 0
	if s, ok := head.(Sym); ok && headerInline[string(s)] && len(rest) > 0 {
		if fits(rest[0], col+1+len(Write(head))+1) {
			b.WriteByte(' ')
			b.WriteString(Write(rest[0]))
			first = 1
		}
	}
	for _, e := range rest[first:] {
		b.WriteByte('\n')
		b.WriteString(strings.Repeat(" ", col+2))
		pretty(b, e, col+2)
	}
	if _, ok := tail.(Nil); !ok {
		b.WriteString(" . ")
		pretty(b, tail, col+2+3)
	}
	b.WriteByte(')')
}

func prettyVec(b *strings.Builder, v *Vec, col int) {
	b.WriteString("#(")
	for _, e := range v.Elems {
		b.WriteByte('\n')
		b.WriteString(strings.Repeat(" ", col+2))
		pretty(b, e, col+2)
	}
	b.WriteByte(')')
}
