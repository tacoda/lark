# Lark Language Specification (v0)

Lark is a small Scheme. Code and data share one representation: every program
is a datum, and every datum can be parsed, printed, transformed, and evaluated.
This document is the whole language — special forms plus core builtins,
nothing else.

## Values

| Type | Literals | Notes |
|---|---|---|
| nil | `()` | the empty list; the only nil |
| boolean | `#t` `#f` | only `#f` is false — `0` and `()` are truthy |
| integer | `42` `-7` `0x2a` | int64 |
| float | `3.14` `2.0` `1e-9` `+Inf` `NaN` | float64 |
| string | `"hi\n"` | escapes: `\" \\ \n \t \r \0` |
| symbol | `foo` `list->vector` | case-sensitive |
| pair/list | `(a b c)` `(a . b)` | cons cells |
| vector | `#(1 2 3)` | |
| procedure | `(lambda ...)` builtins macros | |
| error | result of `(error ...)` | data, check with `error?` |

Reader sugar: `'x` = `(quote x)`, `` `x `` = `(quasiquote x)`, `,x` =
`(unquote x)`, `,@x` = `(unquote-splicing x)`. Comments: `; line`,
`#| nestable block |#`.

## Special forms

Special-form names are reserved, but a macro binding of the same name shadows
the form. Tail positions (if branches, begin/cond/case bodies, and/or last
operand, macro expansions, procedure calls) do not grow the stack.

```
(quote x)                  → x, unevaluated
(quasiquote t)             → t with ,unquote / ,@unquote-splicing applied (nestable)
(if test then)             → then if test not #f, else nil
(if test then else)
(define name expr)         → bind name in current env
(define (f a b) body...)   → function shorthand
(set! name expr)           → mutate nearest existing binding
(lambda (a b) body...)     → closure; (lambda args body...) binds all; (lambda (a . rest) ...)
(let ((x e) ...) body...)  → inits evaluated in outer env
(let* ((x e) ...) body...) → each init sees the previous
(letrec ((x e) ...) body...)
(begin e...)               → sequence, last is tail
(cond (test body...) ... (else body...))
(case key ((d...) body...) ... (else body...))   → compares with equal?
(and e...)                 → last value, or first #f; (and) → #t
(or e...)                  → first non-#f value; (or) → #f
(when test body...) / (unless test body...)
(while test body...)       → loop; body not in tail position
(defmacro name (params) body...)   → macro; also (defmacro (name params) body...)
```

Macros receive **unevaluated** arguments and return code. Expansion happens
in tail position, with a 100000-expansion cap per evaluation.

## Core builtins

Arithmetic (int64 if all args int, else float64; `/` truncates on ints,
errors on division by zero; `mod` is floored, `rem`/`quo` truncated):

```
(+ ...) (- ...) (* ...) (/ ...) (mod a b) (quo a b) (rem a b)
(abs x) (min ...) (max ...) (< a b ...) (<= ...) (> ...) (>= ...) (= ...)
```

Equality: `(eq? a b)` identity for pairs/vectors, eqv for atoms; `(eqv? a b)`
same-type value equality; `(equal? a b)` deep structural; `(not x)`.

Lists: `(cons a b) (car p) (cdr p) (list ...) (append l ... tail)
(reverse l) (length l) (nth l i)   ; 0-based
(map f l ...)   ; stops at shortest list
(filter pred l) (fold-left f init l) (apply f a ... lst)`

Predicates: `(null? x) (pair? x) (list? x) (symbol? x) (string? x)
(number? x) (boolean? x) (procedure? x) (vector? x) (error? x) (atom? x)`

Strings & symbols: `(symbol->string s) (string->symbol s)
(string-append s ...) (string-length s)   ; bytes
(substring s start [end])`

Vectors: `(vector ...) (vector-ref v i) (vector-set! v i x)
(vector-length v) (list->vector l) (vector->list v)`

Errors: `(error msg data...)` → an error value.

## Agent layer

These expose the implementation to lark code. `parse` and `eval` return an
**error value** on failure (check with `error?`), not a crash.

```
(parse "(+ 1 2)")        → datum
(emit v)                 → canonical string; (emit v #t) → pretty
(eval form)              → value, evaluated in the caller's env
(procedure-src proc)     → (lambda ...) / (defmacro ...) source, re-evaluable
(macroexpand-1 form)     → one expansion, or form unchanged
(macroexpand form)       → expand until the head is not a macro
```

`emit` output always re-parses to the same value: `parse(emit(v)) == v`.

## CLI

```
lark                 # REPL (multi-line input supported)
lark run file.lark   # evaluate file, print last value
lark -e '(+ 1 2)'    # evaluate expression, print result
lark parse file.lark # print AST of each top-level form
lark emit file.lark  # pretty-print each top-level form
```

## Deliberate non-goals (v0)

No I/O, filesystem, network, or time primitives. No stdlib modules beyond
the core above. No bignums/rationals (int64 overflow wraps). No hygiene —
macros are unhygienic. No `try`/`catch`. No self-hosting.
