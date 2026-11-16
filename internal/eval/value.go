package eval

import (
	"fmt"

	"github.com/tacoda/lark/internal/sexp"
)

// Lambda is a user-defined procedure: parameters, body forms, and the
// captured defining environment (a closure).
type Lambda struct {
	Params sexp.Value // proper/dotted list of syms, or a single sym
	Body   sexp.Value // proper list of forms
	Env    *Env
}

func (*Lambda) LarkValue()       {}
func (l *Lambda) String() string { return "#<lambda>" }

// bind creates the call environment for args.
func (l *Lambda) bind(args []sexp.Value) (*Env, error) {
	env := NewEnv(l.Env)
	if s, ok := l.Params.(sexp.Sym); ok {
		// (lambda args body...) — bind the whole arg list
		env.Define(string(s), sexp.List(args...))
		return env, nil
	}
	elems, tail := sexp.Split(l.Params)
	i := 0
	for _, e := range elems {
		s, ok := e.(sexp.Sym)
		if !ok {
			return nil, fmt.Errorf("lambda: bad parameter: %s", sexp.Write(e))
		}
		if i >= len(args) {
			return nil, fmt.Errorf("lambda: too few args: want %d, got %d", len(elems), len(args))
		}
		env.Define(string(s), args[i])
		i++
	}
	switch t := tail.(type) {
	case sexp.Nil:
		if i != len(args) {
			return nil, fmt.Errorf("lambda: too many args: want %d, got %d", i, len(args))
		}
	case sexp.Sym:
		env.Define(string(t), sexp.List(args[i:]...))
	default:
		return nil, fmt.Errorf("lambda: bad rest parameter: %s", sexp.Write(t))
	}
	return env, nil
}

// Macro is a defmacro-defined transformer: raw (unevaluated) args in,
// code out. The expansion is evaluated in the caller's environment.
type Macro struct {
	Name string
	Lam  *Lambda // params + body, evaluated with raw args bound
}

func (*Macro) LarkValue()       {}
func (m *Macro) String() string { return "#<macro " + m.Name + ">" }

// expand runs the macro body with raw args bound and returns the expansion.
func (m *Macro) expand(args []sexp.Value) (sexp.Value, error) {
	env, err := m.Lam.bind(args)
	if err != nil {
		return nil, err
	}
	return Eval(&sexp.Pair{Car: sexp.Sym("begin"), Cdr: m.Lam.Body}, env)
}

// expandRaw expands a call form (m arg...) with unevaluated args.
func expandRaw(m *Macro, form *sexp.Pair) (sexp.Value, error) {
	elems, tail := sexp.Split(form)
	if _, ok := tail.(sexp.Nil); !ok {
		return nil, fmt.Errorf("macro %s: improper argument list: %s", m.Name, sexp.Write(form))
	}
	return m.expand(elems[1:])
}

// Builtin is a Go-implemented procedure.
type Builtin struct {
	Name string
	Fn   func(args []sexp.Value) (sexp.Value, error)
}

// EnvBuiltin is a Go-implemented procedure that needs the caller's env
// (macroexpand, map, apply, ...).
type EnvBuiltin struct {
	Name string
	Fn   func(args []sexp.Value, env *Env) (sexp.Value, error)
}

func (*EnvBuiltin) LarkValue()       {}
func (b *EnvBuiltin) String() string { return "#<builtin " + b.Name + ">" }

func (*Builtin) LarkValue()       {}
func (b *Builtin) String() string { return "#<builtin " + b.Name + ">" }

// Error is a lark error value, created by the error builtin.
type Error struct {
	Msg  string
	Data sexp.Value // proper list of extra arguments
}

func (*Error) LarkValue()       {}
func (e *Error) String() string { return "#<error " + e.Msg + ">" }
