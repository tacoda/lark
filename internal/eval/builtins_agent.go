package eval

import (
	"fmt"

	"github.com/tacoda/lark/internal/sexp"
)

// The agent layer: these builtins expose the reader, printer, and evaluator
// to lark code itself, so a program can parse source, transform the AST as
// plain data, re-emit it, and eval it — the core agentic loop.
//
// parse/eval return an Error value (not a Go error) on failure, keeping
// "errors are data" inside the language.
func defAgent() {
	defEnv("parse", func(args []sexp.Value, env *Env) (sexp.Value, error) {
		if err := argErr("parse", args, 1, 1); err != nil {
			return nil, err
		}
		s, ok := asStr(args[0])
		if !ok {
			return nil, fmt.Errorf("parse: not a string: %s", sexp.Write(args[0]))
		}
		v, err := sexp.Parse(s)
		if err != nil {
			return &Error{Msg: err.Error(), Data: sexp.List(args[0])}, nil
		}
		return v, nil
	})

	defEnv("emit", func(args []sexp.Value, env *Env) (sexp.Value, error) {
		if err := argErr("emit", args, 1, 2); err != nil {
			return nil, err
		}
		if len(args) == 2 && !isFalse(args[1]) {
			return sexp.Str(sexp.Pretty(args[0])), nil
		}
		return sexp.Str(sexp.Write(args[0])), nil
	})

	defEnv("eval", func(args []sexp.Value, env *Env) (sexp.Value, error) {
		if err := argErr("eval", args, 1, 1); err != nil {
			return nil, err
		}
		v, err := Eval(args[0], env)
		if err != nil {
			return &Error{Msg: err.Error(), Data: sexp.List(args[0])}, nil
		}
		return v, nil
	})

	defEnv("procedure-src", func(args []sexp.Value, env *Env) (sexp.Value, error) {
		if err := argErr("procedure-src", args, 1, 1); err != nil {
			return nil, err
		}
		switch p := args[0].(type) {
		case *Lambda:
			body, _ := sexp.Slice(p.Body)
			forms := append([]sexp.Value{sexp.Sym("lambda"), p.Params}, body...)
			return sexp.List(forms...), nil
		case *Macro:
			body, _ := sexp.Slice(p.Lam.Body)
			forms := append([]sexp.Value{sexp.Sym("defmacro"), sexp.Sym(p.Name), p.Lam.Params}, body...)
			return sexp.List(forms...), nil
		default:
			return &Error{Msg: "procedure-src: no source for " + sexp.Write(args[0]), Data: sexp.List(args[0])}, nil
		}
	})
}
