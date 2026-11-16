# Lark

Lisp for agents: a small Scheme in Go optimized for agentic writing via
AST-as-data. See `docs/PLAN.md` for the plan and `docs/SPEC.md` for the full
language reference.

## Build & test

```sh
go build ./...
go vet ./...
go test ./...
```

No external dependencies in v0 — standard library only.

## Layout rules

- `internal/sexp` (data model, reader, printer) must not import `internal/eval`.
- Every builtin and special form in v0 is listed in `docs/PLAN.md`; do not add
  stdlib modules — primitives only.
- Golden test programs live in `testdata/`.
