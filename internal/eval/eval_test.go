package eval

import (
	"testing"

	"github.com/tacoda/lark/internal/sexp"
)

// TestGolden is the phase-2 golden suite: each program runs in a fresh global
// env and must produce the expected value.
func TestGolden(t *testing.T) {
	cases := []struct {
		src  string
		want sexp.Value
	}{
		// self-evaluating and quote
		{"42", sexp.Int(42)},
		{"3.5", sexp.Float(3.5)},
		{`"hi"`, sexp.Str("hi")},
		{"#t", sexp.Bool(true)},
		{"()", sexp.NilVal},
		{"'sym", sexp.Sym("sym")},
		{"(quote (a b))", sexp.List(sexp.Sym("a"), sexp.Sym("b"))},
		{"'(1 . 2)", &sexp.Pair{Car: sexp.Int(1), Cdr: sexp.Int(2)}},
		// quasiquote
		{"`(+ 1 ,(+ 1 1))", sexp.List(sexp.Sym("+"), sexp.Int(1), sexp.Int(2))},
		{"`(a ,@(list 1 2) b)", sexp.List(sexp.Sym("a"), sexp.Int(1), sexp.Int(2), sexp.Sym("b"))},
		{"``(x ,y)", sexp.List(sexp.Sym("quasiquote"), sexp.List(sexp.Sym("x"), sexp.List(sexp.Sym("unquote"), sexp.Sym("y"))))},
		// arithmetic
		{"(+ 1 2 3)", sexp.Int(6)},
		{"(+ 1 2.5)", sexp.Float(3.5)},
		{"(- 10 3 2)", sexp.Int(5)},
		{"(- 5)", sexp.Int(-5)},
		{"(* 2 3 4)", sexp.Int(24)},
		{"(/ 12 4)", sexp.Int(3)},
		{"(/ 7 2)", sexp.Int(3)}, // truncating int division
		{"(/ 7.0 2)", sexp.Float(3.5)},
		{"(mod 7 3)", sexp.Int(1)},
		{"(mod -7 3)", sexp.Int(2)},  // floored
		{"(rem -7 3)", sexp.Int(-1)}, // truncated
		{"(quo -7 2)", sexp.Int(-3)},
		{"(abs -3)", sexp.Int(3)},
		{"(abs -2.5)", sexp.Float(2.5)},
		{"(min 3 1 2)", sexp.Int(1)},
		{"(max 3 1 2)", sexp.Int(3)},
		{"(< 1 2 3)", sexp.Bool(true)},
		{"(< 1 3 2)", sexp.Bool(false)},
		{"(= 2 2.0)", sexp.Bool(true)},
		{"(>= 3 3 2)", sexp.Bool(true)},
		// equality & truthiness (only #f is false)
		{"(not #f)", sexp.Bool(true)},
		{"(not 0)", sexp.Bool(false)},
		{"(eq? 'a 'a)", sexp.Bool(true)},
		{"(eq? '(1) '(1))", sexp.Bool(false)}, // pairs by identity
		{"(eqv? 2 2.0)", sexp.Bool(false)},    // different types
		{"(equal? '(1 (2)) '(1 (2)))", sexp.Bool(true)},
		{"(if 0 1 2)", sexp.Int(1)},   // 0 is truthy
		{"(if '() 1 2)", sexp.Int(1)}, // nil is truthy
		{"(if #f 1 2)", sexp.Int(2)},
		{"(if #f 1)", sexp.NilVal},
		// conditionals
		{"(cond (#f 1) (#t 2) (else 3))", sexp.Int(2)},
		{"(cond (#f 1) (else 3))", sexp.Int(3)},
		{"(cond (#f 1))", sexp.NilVal},
		{"(cond (#f) (#t))", sexp.Bool(true)}, // clause with test only returns test
		{"(case 2 ((1) 'one) ((2 3) 'two) (else 'other))", sexp.Sym("two")},
		{"(case 9 ((1) 'one) (else 'other))", sexp.Sym("other")},
		{"(and)", sexp.Bool(true)},
		{"(and 1 2 3)", sexp.Int(3)},
		{"(and 1 #f 3)", sexp.Bool(false)},
		{"(or)", sexp.Bool(false)},
		{"(or #f 0)", sexp.Int(0)},
		{"(or #f #f)", sexp.Bool(false)},
		{"(when #t 1 2)", sexp.Int(2)},
		{"(when #f 1)", sexp.NilVal},
		{"(unless #f 5)", sexp.Int(5)},
		{"(unless #t 5)", sexp.NilVal},
		{"(begin 1 2 3)", sexp.Int(3)},
		{"(begin)", sexp.NilVal},
		// define / set! / lambda / closures
		{"(define x 10) (+ x 1)", sexp.Int(11)},
		{"(define (f x) (* x x)) (f 7)", sexp.Int(49)},
		{"(define (add a b) (+ a b)) (add 2 3)", sexp.Int(5)},
		{"((lambda (x) (+ x 1)) 41)", sexp.Int(42)},
		{"(define (f . args) args) (f 1 2 3)", sexp.List(sexp.Int(1), sexp.Int(2), sexp.Int(3))},
		{"(define (g a . rest) (list a rest)) (g 1 2 3)", sexp.List(sexp.Int(1), sexp.List(sexp.Int(2), sexp.Int(3)))},
		{"(define x 1) (set! x 99) x", sexp.Int(99)},
		{"(define (make-adder n) (lambda (x) (+ x n))) ((make-adder 5) 10)", sexp.Int(15)},
		{"(define x 1) (define (f x) (+ x 100)) (list x (f 1))", sexp.List(sexp.Int(1), sexp.Int(101))},
		{"(define (counter) (define n 0) (lambda () (set! n (+ n 1)) n)) (define c (counter)) (c) (c) (c)", sexp.Int(3)},
		// recursion & binding forms
		{"(define (fact n) (if (= n 0) 1 (* n (fact (- n 1))))) (fact 10)", sexp.Int(3628800)},
		{"(let ((x 1) (y 2)) (+ x y))", sexp.Int(3)},
		{"(let* ((x 1) (y (+ x 1))) y)", sexp.Int(2)},
		{"(let ((x 1)) (define y 2) (+ x y))", sexp.Int(3)},
		{"(letrec ((ev? (lambda (n) (if (= n 0) #t (od? (- n 1))))) (od? (lambda (n) (if (= n 0) #f (ev? (- n 1)))))) (ev? 10))", sexp.Bool(true)},
		{"(define i 0) (while (< i 5) (set! i (+ i 1))) i", sexp.Int(5)},
		// lists
		{"(cons 1 2)", &sexp.Pair{Car: sexp.Int(1), Cdr: sexp.Int(2)}},
		{"(car '(1 2))", sexp.Int(1)},
		{"(cdr '(1 2))", sexp.List(sexp.Int(2))},
		{"(list 1 2 3)", sexp.List(sexp.Int(1), sexp.Int(2), sexp.Int(3))},
		{"(append '(1 2) '(3) '())", sexp.List(sexp.Int(1), sexp.Int(2), sexp.Int(3))},
		{"(append '(1) 2)", &sexp.Pair{Car: sexp.Int(1), Cdr: sexp.Int(2)}},
		{"(reverse '(1 2 3))", sexp.List(sexp.Int(3), sexp.Int(2), sexp.Int(1))},
		{"(length '(a b c))", sexp.Int(3)},
		{"(length '())", sexp.Int(0)},
		{"(nth '(a b c) 1)", sexp.Sym("b")},
		{"(map (lambda (x) (* 2 x)) '(1 2 3))", sexp.List(sexp.Int(2), sexp.Int(4), sexp.Int(6))},
		{"(map + '(1 2) '(10 20))", sexp.List(sexp.Int(11), sexp.Int(22))},
		{"(filter (lambda (x) (> x 2)) '(1 2 3 4))", sexp.List(sexp.Int(3), sexp.Int(4))},
		{"(fold-left + 0 '(1 2 3 4))", sexp.Int(10)},
		{"(fold-left cons '() '(1 2 3))", &sexp.Pair{Car: &sexp.Pair{Car: &sexp.Pair{Car: sexp.NilVal, Cdr: sexp.Int(1)}, Cdr: sexp.Int(2)}, Cdr: sexp.Int(3)}},
		{"(apply + 1 2 '(3 4))", sexp.Int(10)},
		{"(apply list '(1 2))", sexp.List(sexp.Int(1), sexp.Int(2))},
		// predicates
		{"(null? '())", sexp.Bool(true)},
		{"(null? '(1))", sexp.Bool(false)},
		{"(pair? '(1))", sexp.Bool(true)},
		{"(pair? '())", sexp.Bool(false)},
		{"(list? '(1 2))", sexp.Bool(true)},
		{"(list? '(1 . 2))", sexp.Bool(false)},
		{"(symbol? 'a)", sexp.Bool(true)},
		{"(string? \"a\")", sexp.Bool(true)},
		{"(number? 3.5)", sexp.Bool(true)},
		{"(number? 'a)", sexp.Bool(false)},
		{"(boolean? #f)", sexp.Bool(true)},
		{"(procedure? car)", sexp.Bool(true)},
		{"(procedure? (lambda (x) x))", sexp.Bool(true)},
		{"(procedure? 5)", sexp.Bool(false)},
		{"(vector? #(1))", sexp.Bool(true)},
		{"(error? (error \"boom\"))", sexp.Bool(true)},
		{"(atom? 'a)", sexp.Bool(true)},
		{"(atom? '(a))", sexp.Bool(false)},
		// strings & symbols
		{"(symbol->string 'abc)", sexp.Str("abc")},
		{"(string->symbol \"abc\")", sexp.Sym("abc")},
		{"(string-append \"a\" \"b\" \"c\")", sexp.Str("abc")},
		{"(string-length \"hello\")", sexp.Int(5)},
		{"(substring \"hello\" 1 3)", sexp.Str("el")},
		{"(substring \"hello\" 2)", sexp.Str("llo")},
		// vectors
		{"(vector 1 2 3)", &sexp.Vec{Elems: []sexp.Value{sexp.Int(1), sexp.Int(2), sexp.Int(3)}}},
		{"(vector-ref #(a b c) 1)", sexp.Sym("b")},
		{"(vector-length #(1 2))", sexp.Int(2)},
		{"(list->vector '(1 2))", &sexp.Vec{Elems: []sexp.Value{sexp.Int(1), sexp.Int(2)}}},
		{"(vector->list #(1 2))", sexp.List(sexp.Int(1), sexp.Int(2))},
		{"(define v (vector 1 2)) (vector-set! v 0 9) (vector-ref v 0)", sexp.Int(9)},
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

func TestEvalErrors(t *testing.T) {
	for _, src := range []string{
		"undefined-var",
		"(car '())",
		"(car 5)",
		"(+ 1 'a)",
		"(/ 1 0)",
		"(mod 1 0)",
		"((lambda (x) x))",     // too few args
		"((lambda (x) x) 1 2)", // too many args
		"(1 2 3)",              // not a procedure
		"(set! nope 1)",
		"(if)",
		"(quote)",
		"(lambda)",
		"(let ((x)) x)",
		"(vector-ref #(1) 5)",
		"(nth '(1) 9)",
		"(substring \"abc\" 2 1)",
		"`(,@1)", // splicing a non-list
	} {
		if _, err := Run(src, GlobalEnv()); err == nil {
			t.Errorf("Run(%q): expected error, got none", src)
		}
	}
}

// TestTailCallsDoNotGrowStack is the plan's TCO acceptance test: a million
// tail calls must complete without stack overflow.
func TestTailCallsDoNotGrowStack(t *testing.T) {
	src := `(define (loop n) (if (= n 0) 'done (loop (- n 1)))) (loop 1000000)`
	got, err := Run(src, GlobalEnv())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !sexp.Equal(got, sexp.Sym("done")) {
		t.Fatalf("got %s, want done", sexp.Write(got))
	}
}

// TestMutualTailRecursion checks TCO across mutually recursive functions.
func TestMutualTailRecursion(t *testing.T) {
	src := `
(define (ev? n) (if (= n 0) #t (od? (- n 1))))
(define (od? n) (if (= n 0) #f (ev? (- n 1))))
(ev? 200001)`
	got, err := Run(src, GlobalEnv())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !sexp.Equal(got, sexp.Bool(false)) {
		t.Fatalf("got %s, want #f", sexp.Write(got))
	}
}
