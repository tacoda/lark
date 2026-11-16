# lark

Lisp for agents. A small Scheme written in Go, designed for *agentic writing*:
programs are data, and the AST is the interface. Agents construct, inspect,
transform, and evaluate code without a parser/codegen mismatch — what you
write is what runs.

## Status

v0 planning — see [docs/PLAN.md](docs/PLAN.md). No standard library beyond
the language core; primitives only.

## Usage (once built)

```sh
lark                 # REPL
lark run prog.lark   # run a file
lark -e '(+ 1 2)'    # evaluate a string
lark parse prog.lark # dump AST
lark emit  prog.lark # normalize/print canonically
```

## Layout

- `internal/sexp` — data model, reader, printer (no deps on eval)
- `internal/eval` — evaluator, special forms, macros, builtins
- `cmd/lark` — CLI
- `docs/SPEC.md` — language reference (agent-loadable)
- `docs/PLAN.md` — implementation plan
- `testdata/` — golden programs
