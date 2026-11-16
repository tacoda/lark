package eval

import (
	"testing"

	"github.com/tacoda/lark/internal/sexp"
)

// TestMacros is the phase-3 golden suite: lark-defined unless/while,
// threading macros, defmacro shorthand, and macroexpand.
func TestMacros(t *testing.T) {
	cases := []struct {
		src  string
		want sexp.Value
	}{
		// defmacro with (name params) sugar
		{"(defmacro (sq x) (* x x)) (sq 9)", sexp.Int(81)},
		// defmacro plain form, quasiquote body
		{"(defmacro inc (x) `(+ 1 ,x)) (inc 5)", sexp.Int(6)},
		// rest params
		{"(defmacro my-list args (cons 'list args)) (my-list 1 2 3)", sexp.List(sexp.Int(1), sexp.Int(2), sexp.Int(3))},
		// lark-defined unless shadows the built-in special form
		{"(defmacro unless (t . body) `(if ,t #f (begin ,@body))) (unless #f 42)", sexp.Int(42)},
		{"(defmacro unless (t . body) `(if ,t #f (begin ,@body))) (unless #t 42)", sexp.Bool(false)},
		{"(defmacro unless (t . body) `(if ,t #f (begin ,@body))) (unless #f 1 2 3)", sexp.Int(3)},
		// lark-defined while as a macro over letrec + tail call (TCO through it)
		{"(defmacro while (t . body) `(letrec ((go (lambda () (when ,t ,@body (go))))) (go)))\n" +
			"(define i 0) (while (< i 100000) (set! i (+ i 1))) i", sexp.Int(100000)},
		// threading macros (computed at expansion time, no quasiquote needed)
		{"(defmacro -> (x . forms)\n" +
			"  (fold-left (lambda (acc f) (if (pair? f) (cons (car f) (cons acc (cdr f))) (list f acc))) x forms))\n" +
			"(-> 5 (+ 1) (* 2))", sexp.Int(12)},
		{"(defmacro ->> (x . forms)\n" +
			"  (fold-left (lambda (acc f) (if (pair? f) (append f (list acc)) (list f acc))) x forms))\n" +
			"(->> '(1 2) (append '(3)) reverse)", sexp.List(sexp.Int(2), sexp.Int(1), sexp.Int(3))},
		// macro calling a helper function defined in lark
		{"(define (dup-arg f) (list (car f) (car (cdr f)) (car (cdr f))))\n" +
			"(defmacro twice (f) (dup-arg f))\n" +
			"(twice (+ 1))", sexp.Int(2)},
		// macroexpand-1 / macroexpand
		{"(defmacro inc (x) `(+ 1 ,x)) (macroexpand-1 '(inc 5))", sexp.List(sexp.Sym("+"), sexp.Int(1), sexp.Int(5))},
		{"(macroexpand-1 '(+ 1 2))", sexp.List(sexp.Sym("+"), sexp.Int(1), sexp.Int(2))},
		{"(macroexpand-1 'x)", sexp.Sym("x")},
		{"(defmacro inc (x) `(+ 1 ,x)) (macroexpand '(inc 5))", sexp.List(sexp.Sym("+"), sexp.Int(1), sexp.Int(5))},
		{"(defmacro a (x) `(b ,x)) (defmacro b (x) `(+ ,x 1)) (macroexpand '(a 5))", sexp.List(sexp.Sym("+"), sexp.Int(5), sexp.Int(1))},
		// macros are first-class values
		{"(defmacro inc (x) `(+ 1 ,x)) (procedure? inc)", sexp.Bool(true)},
		{"(defmacro inc (x) `(+ 1 ,x)) (macroexpand-1 (list 'inc 41))", sexp.List(sexp.Sym("+"), sexp.Int(1), sexp.Int(41))},
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

func TestMacroErrors(t *testing.T) {
	for _, src := range []string{
		"(defmacro loopy (x) '(loopy x)) (loopy 1)",    // expansion limit
		"(defmacro m (x) '(m x)) (macroexpand '(m 1))", // macroexpand cap
		"(defmacro add2 (a b) `(+ ,a ,b)) (add2 1)",    // too few args
		"(defmacro)",                 // bad form
		"(defmacro m (x) x) (m 1 2)", // too many args
	} {
		if _, err := Run(src, GlobalEnv()); err == nil {
			t.Errorf("Run(%q): expected error, got none", src)
		}
	}
}

// TestMacroExpansionIsTailPosition: a macro expanding to a tail call must
// not grow the stack.
func TestMacroExpansionIsTailPosition(t *testing.T) {
	src := "(defmacro while (t . body) `(letrec ((go (lambda () (when ,t ,@body (go))))) (go)))\n" +
		"(define i 0) (while (< i 500000) (set! i (+ i 1))) i"
	got, err := Run(src, GlobalEnv())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !sexp.Equal(got, sexp.Int(500000)) {
		t.Fatalf("got %s, want 500000", sexp.Write(got))
	}
}
