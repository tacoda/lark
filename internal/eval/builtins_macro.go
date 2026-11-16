package eval

import (
	"fmt"

	"github.com/tacoda/lark/internal/sexp"
)

// macroexpandCap bounds full macroexpansion so a macro that expands to
// itself errors instead of looping.
const macroexpandCap = 1000

func defMacroBuiltins() {
	defEnv("macroexpand-1", func(args []sexp.Value, env *Env) (sexp.Value, error) {
		if err := argErr("macroexpand-1", args, 1, 1); err != nil {
			return nil, err
		}
		v, _, err := expand1(args[0], env)
		return v, err
	})
	defEnv("macroexpand", func(args []sexp.Value, env *Env) (sexp.Value, error) {
		if err := argErr("macroexpand", args, 1, 1); err != nil {
			return nil, err
		}
		cur := args[0]
		for i := 0; i < macroexpandCap; i++ {
			next, expanded, err := expand1(cur, env)
			if err != nil {
				return nil, err
			}
			if !expanded {
				return cur, nil
			}
			cur = next
		}
		return nil, fmt.Errorf("macroexpand: exceeded %d expansions (recursive macro?)", macroexpandCap)
	})
}
