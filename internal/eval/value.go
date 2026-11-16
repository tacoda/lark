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

// Builtin is a Go-implemented procedure.
type Builtin struct {
	Name string
	Fn   func(args []sexp.Value) (sexp.Value, error)
}

func (*Builtin) LarkValue()       {}
func (b *Builtin) String() string { return "#<builtin " + b.Name + ">" }

// Error is a lark error value, created by the error builtin.
type Error struct {
	Msg  string
	Data sexp.Value // proper list of extra arguments
}

func (*Error) LarkValue()       {}
func (e *Error) String() string { return "#<error " + e.Msg + ">" }
