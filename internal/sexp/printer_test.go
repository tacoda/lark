package sexp

import (
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	cases := []struct {
		v    Value
		want string
	}{
		{NilVal, "()"},
		{Bool(true), "#t"},
		{Bool(false), "#f"},
		{Int(-42), "-42"},
		{Float(2), "2.0"},
		{Float(3.14), "3.14"},
		{Float(1000), "1000.0"},
		{Str("a\"b\n\\"), `"a\"b\n\\"`},
		{Sym("foo"), "foo"},
		{List(Sym("quote"), Int(1)), "(quote 1)"}, // Write never uses sugar
		{List(Sym("+"), Int(1), Int(2)), "(+ 1 2)"},
		{&Pair{Car: Sym("a"), Cdr: Sym("b")}, "(a . b)"},
		{ListTail(Sym("c"), Sym("a"), Sym("b")), "(a b . c)"},
		{&Vec{Elems: []Value{Int(1), Sym("x")}}, "#(1 x)"},
		{&Vec{}, "#()"},
	}
	for _, c := range cases {
		if got := Write(c.v); got != c.want {
			t.Errorf("Write(%v) = %q, want %q", c.v, got, c.want)
		}
	}
}

func TestPrettyShortStaysFlat(t *testing.T) {
	v := List(Sym("+"), Int(1), Int(2))
	if got := Pretty(v); got != "(+ 1 2)" {
		t.Errorf("Pretty = %q, want flat", got)
	}
}

func TestPrettyBreaksLongCall(t *testing.T) {
	v := List(Sym("aaaaaaaaaa"), Sym("bbbbbbbbbb"), Sym("cccccccccc"),
		Sym("dddddddddd"), Sym("eeeeeeeeee"), Sym("ffffffffff"), Sym("gggggggggg"),
		Sym("hhhhhhhhhh"))
	want := "(aaaaaaaaaa\n" +
		"  bbbbbbbbbb\n" +
		"  cccccccccc\n" +
		"  dddddddddd\n" +
		"  eeeeeeeeee\n" +
		"  ffffffffff\n" +
		"  gggggggggg\n" +
		"  hhhhhhhhhh)"
	if got := Pretty(v); got != want {
		t.Errorf("Pretty:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrettyDefineKeepsHeaderInline(t *testing.T) {
	body := Sym(strings.Repeat("b", 77))
	v := List(Sym("define"), List(Sym("f"), Sym("x")), body)
	want := "(define (f x)\n  " + strings.Repeat("b", 77) + ")"
	if got := Pretty(v); got != want {
		t.Errorf("Pretty:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrettyUsesQuoteSugar(t *testing.T) {
	v := List(Sym("quote"), List(Sym("aaaaaaaaaa"), Sym("bbbbbbbbbb"), Sym("cccccccccc"),
		Sym("dddddddddd"), Sym("eeeeeeeeee"), Sym("ffffffffff"), Sym("gggggggggg"),
		Sym("hhhhhhhhhh")))
	got := Pretty(v)
	if !strings.HasPrefix(got, "'(aaaaaaaaaa\n") {
		t.Errorf("Pretty should use ' sugar, got:\n%s", got)
	}
}

func TestPrettyDotted(t *testing.T) {
	v := &Pair{Car: Sym("a"), Cdr: Sym("b")}
	if got := Pretty(v); got != "(a . b)" {
		t.Errorf("Pretty = %q, want (a . b)", got)
	}
}
