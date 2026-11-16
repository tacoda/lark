package eval

import (
	"fmt"

	"github.com/tacoda/lark/internal/sexp"
)

// Env is a lexical environment: one frame of bindings plus a parent pointer.
// Lookup walks outward; Define always binds in the current frame.
type Env struct {
	vars   map[string]sexp.Value
	parent *Env
}

// NewEnv returns a child of parent (nil for the root).
func NewEnv(parent *Env) *Env {
	return &Env{vars: map[string]sexp.Value{}, parent: parent}
}

// Lookup finds the nearest binding of name.
func (e *Env) Lookup(name string) (sexp.Value, bool) {
	for cur := e; cur != nil; cur = cur.parent {
		if v, ok := cur.vars[name]; ok {
			return v, true
		}
	}
	return nil, false
}

// Define binds name in the current frame, shadowing any outer binding.
func (e *Env) Define(name string, v sexp.Value) { e.vars[name] = v }

// Set mutates the nearest existing binding of name; error if unbound.
func (e *Env) Set(name string, v sexp.Value) error {
	for cur := e; cur != nil; cur = cur.parent {
		if _, ok := cur.vars[name]; ok {
			cur.vars[name] = v
			return nil
		}
	}
	return fmt.Errorf("set!: unbound symbol: %s", name)
}
