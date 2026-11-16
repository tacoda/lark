package eval

import (
	"fmt"

	"github.com/tacoda/lark/internal/sexp"
)

// GlobalEnv returns a fresh root environment with every core builtin defined.
func GlobalEnv() *Env {
	env := NewEnv(nil)
	for name, b := range builtins {
		env.Define(name, b)
	}
	return env
}

var builtins = map[string]*Builtin{}

func def(name string, fn func(args []sexp.Value) (sexp.Value, error)) {
	builtins[name] = &Builtin{Name: name, Fn: fn}
}

func init() {
	defNumeric()
	defListOps()
	defMisc()
}

// --- shared helpers ---

func argErr(name string, args []sexp.Value, min, max int) error {
	if len(args) < min || (max >= 0 && len(args) > max) {
		return fmt.Errorf("%s: want between %d and %d args, got %d", name, min, max, len(args))
	}
	return nil
}

func asInt(v sexp.Value) (int64, bool) {
	n, ok := v.(sexp.Int)
	return int64(n), ok
}

func asFloat(v sexp.Value) (float64, bool) {
	f, ok := v.(sexp.Float)
	return float64(f), ok
}

// numF returns the numeric value of v as a float64.
func numF(v sexp.Value) (float64, bool) {
	if i, ok := asInt(v); ok {
		return float64(i), true
	}
	return asFloat(v)
}

func asStr(v sexp.Value) (string, bool) {
	s, ok := v.(sexp.Str)
	return string(s), ok
}

func asSym(v sexp.Value) (string, bool) {
	s, ok := v.(sexp.Sym)
	return string(s), ok
}

func asVec(v sexp.Value) (*sexp.Vec, bool) {
	x, ok := v.(*sexp.Vec)
	return x, ok
}

func notNum(name string, v sexp.Value) error {
	return fmt.Errorf("%s: not a number: %s", name, sexp.Write(v))
}

var errDivZero = fmt.Errorf("division by zero")
