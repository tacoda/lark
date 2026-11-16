# Lark — Implementation Plan (v0)

## Status

- Phase 0 (scaffolding: license, CI, layout): **done**
- Phase 1 (sexp: data model, reader, printer + round-trip properties): **done**
- Phase 2 (eval: special forms, closures, TCO, core builtins): **done**
- Phase 3 (defmacro, macroexpand-1/macroexpand, macro expansion in tail position): **done**
- Phase 4 (agent layer: parse/emit/eval/procedure-src builtins): next

Lisp for agents: a small Scheme written in Go, designed for *agentic writing*.
An agent holding lark can construct, inspect, transform, and evaluate programs
as ordinary data — no separate parser, AST schema, or codegen step.

## Vision

Agents write code through tokens and read it through context windows. Lark is
optimized for both directions:

- **AST-as-data** — code and data are the same structure (homoiconicity). The
  reader, printer, and evaluator all operate on one shared representation.
- **Lossless round-trips** — `parse → emit → parse` is identity; every datum
  evaluates to itself when quoted. What an agent writes is what runs.
- **Tiny, complete core** — the whole language is a handful of special forms
  and ~60 builtins, specifiable in a document small enough to live in context.
- **Deterministic output** — the printer is a canonical pretty-printer, so
  emitted code is stable across runs and diffs cleanly.

## Design principles

1. **Agent-first, human-friendly second.** Where they conflict (terse forms,
   explicit over implicit, no magic), agents win.
2. **The core is the whole language.** No stdlib modules in v0 — primitives
   only. Anything else is data the caller provides.
3. **Everything inspects.** Any value can be printed; any function's source is
   reachable; the environment is explorable.
4. **Errors are data.** Failures surface as structured values, never as opaque
   panics.

## v0 language sketch

### Types

| Type | Literal / examples |
|---|---|
| nil | `()` (also the empty list; one value) |
| boolean | `#t` `#f` |
| integer | `42` `-7` `0x2a` (64-bit, arbitrary precision deferred) |
| float | `3.14` `1e-9` |
| string | `"hi\n"` (Go escape syntax) |
| symbol | `foo` `list->vector` (case-sensitive, kebab-case idiomatic) |
| list | `(a b c)` proper; `(a . b)` dotted (cons cells) |
| vector | `#(1 2 3)` |
| procedure | `(lambda (x) ...)`, builtins, macros |
| error | result of `(error ...)` |

### Reader syntax

- `'x` → `(quote x)`; `` `x `` → `(quasiquote x)`; `,x` → `(unquote x)`; `,@x` → `(unquote-splicing x)`
- `;` line comments; `#| ... |#` block comments
- `#t` `#f` `#(...)` vectors

### Special forms

`quote` `quasiquote` `if` `define` `set!` `lambda` `let` `let*` `letrec`
`begin` `cond` `case` `and` `or` `when` `unless` `while` `defmacro` `try`*

`*` — `try`/`catch` is stretch; shipped in v0 only if it lands cleanly.

### Core builtins (the complete v0 set — nothing else ships)

- **Arithmetic** `+ - * / mod quo rem abs min max` `< <= > >= =`
- **Equality** `eq? eqv? equal? not`
- **Lists** `cons car cdr list append reverse length nth map filter fold-left apply`
- **Predicates** `null? pair? list? symbol? string? number? boolean? procedure? vector? error? atom?`
- **Symbols & strings** `symbol->string string->symbol string-append string-length substring`
- **Vectors** `vector vector-ref vector-set! vector-length list->vector vector->list`
- **Agent layer** `parse emit eval macroexpand-1 macroexpand procedure-src error`

### Semantics decisions

- **Truthiness**: only `#f` is false; everything else (including `()` and `0`) is true.
- **Scoping**: lexical, closures capture their environment.
- **Tail calls**: proper TCO via trampolined eval loop.
- **Macros**: `defmacro` (unhygienic, apply-time). No `syntax-rules` in v0.
- **Quasiquote**: full support incl. nested and splicing — the codegen workhorse.
- **Errors**: propagate as Go errors to the host; `(error ...)` raises;
  REPL prints them structurally.

## Non-goals (v0)

- No I/O, filesystem, network, time, or os primitives.
- No stdlib modules, packages, or namespace system.
- No bignums, rationals, or a numeric tower.
- No continuations, dynamic-wind, hygiene, or reader extensibility.
- No self-hosting.

## Architecture

```
lark/
  cmd/lark/          # CLI entry: repl | run | parse | emit | -e
  internal/sexp/     # shared data model: Value types, reader.go, printer.go
  internal/eval/     # env.go, eval.go (TCO loop), special.go, macro.go, builtins*.go
  internal/cli/      # subcommands; REPL
  docs/PLAN.md       # this file
  docs/SPEC.md       # full language reference (agent-loadable)
  testdata/*.lark    # golden programs
```

Dependency rule: `sexp` imports nothing from lark; `eval` imports `sexp`;
`cli` imports both. Reader/printer never depend on evaluation.

## Phases

Each phase is PR-sized, lands green, and ends with tests.

### Phase 0 — Scaffolding
Go module `github.com/tacoda/lark`, README, LICENSE (pick one), `.gitignore`,
lightweight `AGENTS.md` (build/test commands only — lark is *not*
keystone-chartered), GitHub Actions: `go vet`, `go test ./...`.
**Done when:** CI runs green on an empty skeleton.

### Phase 1 — Data model, reader, printer (`internal/sexp`)
Value types for every type above; reader for all syntax incl. quote-sugar and
block comments; printer with a canonical pretty mode (fixed indentation
rules) and a `write` mode (lossless).
**Done when:** property tests pass — `parse(emit(x)) == x` for random
datums, and pretty output is a fixed point: `emit(emit(x)) == emit(x)`.

### Phase 2 — Core evaluator (`internal/eval`)
Environments, all special forms except `defmacro`/`try`, closures, TCO,
arithmetic/equality/list/predicate builtins.
**Done when:** golden suite (~50 programs: recursion, closures, shadowing,
higher-order use) passes; a `(loop 1000000)` tail call does not grow the stack.

### Phase 3 — Macros & quasiquote
`defmacro`, quasiquote/unquote/splicing (incl. nesting), `macroexpand-1` /
`macroexpand`.
**Done when:** lark-defined `unless`, `while`, and threading macros pass
golden tests; expansion is inspectable from the REPL.

### Phase 4 — Agent layer
`parse` `emit` `eval` `procedure-src` builtins; deterministic pretty-printer
tuned for diff stability.
**Done when:** a lark program can: read a source string, parse it, walk the
AST renaming symbols, emit it, and eval the result — all from lark itself
(checked as a golden test).

### Phase 5 — CLI & spec
`lark` (REPL), `lark run f.lark`, `lark -e '(+ 1 2)'`, `lark parse f.lark`
(dump AST), `lark emit f.lark` (normalize). `docs/SPEC.md`: every form and
builtin, one example each, sized to fit comfortably in a context window.
**Done when:** fresh clone → `go install ./cmd/lark` → REPL runs all golden
programs; SPEC.md covers 100% of shipped forms.

## Testing strategy

- Table-driven Go tests per package; no test file exceeds the code it tests.
- Property tests (round-trip, printer fixed-point) with `testing/quick` or a
  custom random-datum generator.
- Golden `.lark` programs in `testdata/`, each expected to run or to fail
  with a specific error shape.
- `lark parse`/`lark emit` diffs against checked-in expectations.

## Open questions (resolve at phase start)

1. **Cons cells vs Go slices** for lists — plan says cons cells (dotted pairs
   stay expressible); revisit if Phase 1 drags.
2. **Integers** — int64 only in v0; `math/big` deferred to a later minor.
3. **`try`/`catch`** — include only if it falls out of error propagation
   naturally; otherwise v0.1.
4. **Module path** — `github.com/tacoda/lark` (default) unless renamed.
