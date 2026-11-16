package eval

import (
	"testing"

	"github.com/tacoda/lark/internal/sexp"
)

// TestAgentBuiltins covers parse / emit / eval / procedure-src.
func TestAgentBuiltins(t *testing.T) {
	cases := []struct {
		src  string
		want sexp.Value
	}{
		{"(parse \"(+ 1 2)\")", sexp.List(sexp.Sym("+"), sexp.Int(1), sexp.Int(2))},
		{"(parse \"42\")", sexp.Int(42)},
		{"(parse \"'(a . b)\")", sexp.List(sexp.Sym("quote"), &sexp.Pair{Car: sexp.Sym("a"), Cdr: sexp.Sym("b")})},
		{"(parse \"(a . b)\")", &sexp.Pair{Car: sexp.Sym("a"), Cdr: sexp.Sym("b")}},
		{"(emit (parse \"(+ 1 2)\"))", sexp.Str("(+ 1 2)")},
		{"(emit '(a b))", sexp.Str("(a b)")},
		{"(emit '(quote a))", sexp.Str("(quote a)")},
		{"(emit (parse \"'(a b)\"))", sexp.Str("(quote (a b))")}, // Write never uses sugar
		{"(eval '(+ 1 2))", sexp.Int(3)},
		{"(eval '(define zz 9)) zz", sexp.Int(9)}, // eval defines in the caller env
		{"(eval (parse \"(* 6 7)\"))", sexp.Int(42)},
		{"(procedure-src (lambda (x) (+ x 1)))", sexp.List(sexp.Sym("lambda"), sexp.List(sexp.Sym("x")), sexp.List(sexp.Sym("+"), sexp.Sym("x"), sexp.Int(1)))},
		{"(defmacro m (x) `(+ ,x 1)) (procedure-src m)", sexp.List(sexp.Sym("defmacro"), sexp.Sym("m"), sexp.List(sexp.Sym("x")), sexp.List(sexp.Sym("quasiquote"), sexp.List(sexp.Sym("+"), sexp.List(sexp.Sym("unquote"), sexp.Sym("x")), sexp.Int(1))))},
		// eval of procedure-src round-trips a lambda into a callable
		{"((eval (procedure-src (lambda (x) (+ x 1)))) 41)", sexp.Int(42)},
	}
	for _, c := range cases {
		got, err := Run(c.src, GlobalEnv())
		if err != nil {
			t.Errorf("Run(%q): unexpected error: %v", c.src, err)
			continue
		}
		if !sexp.Equal(got, c.want) {
			t.Errorf("Run(%q) = %s, want %s", c.src, sexp.Write(got), sexp.Write(c.want))
		}
	}
}

// TestAgentErrorsAreData: parse/eval failures come back as error values,
// not Go errors, so lark code can inspect them with error?.
func TestAgentErrorsAreData(t *testing.T) {
	cases := []struct {
		src  string
		want bool
	}{
		{`(error? (parse "(bad"))`, true},
		{`(error? (parse "(+ 1 2)"))`, false},
		{`(error? (eval 'undefined-var))`, true},
		{`(error? (eval '(+ 1 2)))`, false},
		{`(error? (procedure-src car))`, true},
		{`(error? (procedure-src (lambda (x) x)))`, false},
	}
	for _, c := range cases {
		got, err := Run(c.src, GlobalEnv())
		if err != nil {
			t.Errorf("Run(%q): unexpected error: %v", c.src, err)
			continue
		}
		if b, ok := got.(sexp.Bool); !ok || bool(b) != c.want {
			t.Errorf("Run(%q) = %s, want #%t", c.src, sexp.Write(got), c.want)
		}
	}
}

// TestAgentLoop is the phase-4 acceptance test: a lark program reads source,
// parses it, renames a symbol by walking the AST, emits it, and evals the
// result — all from inside lark.
func TestAgentLoop(t *testing.T) {
	src := `
(define (rename form old new)
  (cond ((eq? form old) new)
        ((pair? form) (cons (rename (car form) old new)
                            (rename (cdr form) old new)))
        ((vector? form) (list->vector
                          (map (lambda (e) (rename e old new)) (vector->list form))))
        (else form)))
(define src "(define (greet name) (list 'hello name))")
(define ast (parse src))
(define renamed (rename ast 'greet 'welcome))
(define emitted (emit renamed))
(eval renamed)
(list emitted (welcome 'world))`
	got, err := Run(src, GlobalEnv())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := sexp.List(
		sexp.Str("(define (welcome name) (list (quote hello) name))"),
		sexp.List(sexp.Sym("hello"), sexp.Sym("world")),
	)
	if !sexp.Equal(got, want) {
		t.Fatalf("got %s, want %s", sexp.Write(got), sexp.Write(want))
	}
}
