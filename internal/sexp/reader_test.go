package sexp

import (
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		src  string
		want Value
	}{
		{"()", NilVal},
		{"#t", Bool(true)},
		{"#f", Bool(false)},
		{"42", Int(42)},
		{"-7", Int(-7)},
		{"+13", Int(13)},
		{"0x2a", Int(42)},
		{"3.14", Float(3.14)},
		{"2.0", Float(2)},
		{"1e3", Float(1000)},
		{"-1.5e-2", Float(-0.015)},
		{`"hi\n"`, Str("hi\n")},
		{`"a\"b\\c\td"`, Str("a\"b\\c\td")},
		{"foo", Sym("foo")},
		{"list->vector", Sym("list->vector")},
		{"-", Sym("-")},
		{"(a b c)", List(Sym("a"), Sym("b"), Sym("c"))},
		{"(a . b)", &Pair{Car: Sym("a"), Cdr: Sym("b")}},
		{"(a b . c)", ListTail(Sym("c"), Sym("a"), Sym("b"))},
		{"'(1 2)", List(Sym("quote"), List(Int(1), Int(2)))},
		{"`(+ ,x ,@xs)", List(Sym("quasiquote"),
			List(Sym("+"),
				List(Sym("unquote"), Sym("x")),
				List(Sym("unquote-splicing"), Sym("xs"))))},
		{"#(1 2 3)", &Vec{Elems: []Value{Int(1), Int(2), Int(3)}}},
		{"#()", &Vec{Elems: nil}},
		{"; comment\n(+ 1 2) ; trailing", List(Sym("+"), Int(1), Int(2))},
		{"#| block |# 1", Int(1)},
		{"#| nested #| comment |# still |# 2", Int(2)},
		{"(a (b (c)))", List(Sym("a"), List(Sym("b"), List(Sym("c"))))},
		{"(+ 1\n   2)", List(Sym("+"), Int(1), Int(2))},
		{"\t\n 42 \n", Int(42)},
	}
	for _, c := range cases {
		got, err := Parse(c.src)
		if err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", c.src, err)
			continue
		}
		if !Equal(got, c.want) {
			t.Errorf("Parse(%q) = %s, want %s", c.src, Write(got), Write(c.want))
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, src := range []string{
		"", "(", "(a", `"abc`, ")", "(a . )", "#x", "(#)", "(a b . c d)",
		`#| unterminated`, `"bad \q escape"`, "42 43",
	} {
		if _, err := Parse(src); err == nil {
			t.Errorf("Parse(%q): expected error, got none", src)
		}
	}
}

func TestParseAll(t *testing.T) {
	vals, err := ParseAll("(+ 1 2) (* 3 4) ; comment\n'sym")
	if err != nil {
		t.Fatalf("ParseAll: %v", err)
	}
	if len(vals) != 3 {
		t.Fatalf("ParseAll returned %d datums, want 3", len(vals))
	}
	if !Equal(vals[2], List(Sym("quote"), Sym("sym"))) {
		t.Errorf("third datum = %s", Write(vals[2]))
	}
}

func TestReadErrorPosition(t *testing.T) {
	_, err := Parse("(ok 1)\n(bad")
	if err == nil {
		t.Fatal("expected error")
	}
	re, ok := err.(*ReadError)
	if !ok {
		t.Fatalf("error is %T, want *ReadError", err)
	}
	if re.Line != 2 {
		t.Errorf("line = %d, want 2", re.Line)
	}
}

func TestGoldenFile(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample.lark")
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	vals, err := ParseAll(string(data))
	if err != nil {
		t.Fatalf("ParseAll(sample.lark): %v", err)
	}
	if len(vals) != 3 {
		t.Fatalf("sample.lark has %d datums, want 3:\n%s", len(vals), data)
	}
}
