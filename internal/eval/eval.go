// Package eval implements lark's evaluator: special forms, closures,
// application, and proper tail calls.
package eval

import (
	"fmt"

	"github.com/tacoda/lark/internal/sexp"
)

// Run parses and evaluates every top-level form in src, returning the value
// of the last one (NilVal for empty input).
func Run(src string, env *Env) (sexp.Value, error) {
	forms, err := sexp.ParseAll(src)
	if err != nil {
		return nil, err
	}
	if len(forms) == 0 {
		return sexp.NilVal, nil
	}
	var last sexp.Value
	for _, f := range forms {
		last, err = Eval(f, env)
		if err != nil {
			return nil, err
		}
	}
	return last, nil
}

// formResult tells Eval what a special-form handler decided.
type formResult int

const (
	notAForm  formResult = iota // head is an ordinary application
	formValue                   // *vp holds the final value
	formNext                    // *vp/*envp hold the next tail expression
)

// Eval evaluates v in env. Tail positions — lambda application, if branches,
// begin/cond/case bodies, and/or last operands — continue the loop instead of
// recursing, so (loop 1000000) does not grow the Go stack.
//
// Truthiness: only #f is false. Special-form names are reserved words.
func Eval(v sexp.Value, env *Env) (sexp.Value, error) {
	for {
		switch x := v.(type) {
		case sexp.Sym:
			val, ok := env.Lookup(string(x))
			if !ok {
				return nil, fmt.Errorf("unbound symbol: %s", x)
			}
			return val, nil
		case *sexp.Pair:
			if hs, ok := x.Car.(sexp.Sym); ok {
				kind, err := evalForm(string(hs), x, env, &v, &env)
				if err != nil {
					return nil, err
				}
				switch kind {
				case formValue:
					return v, nil
				case formNext:
					continue
				}
			}
			// application
			fn, err := Eval(x.Car, env)
			if err != nil {
				return nil, err
			}
			args, err := evalArgs(x.Cdr, env)
			if err != nil {
				return nil, err
			}
			if lam, ok := fn.(*Lambda); ok {
				callEnv, err := lam.bind(args)
				if err != nil {
					return nil, err
				}
				env = callEnv
				v = &sexp.Pair{Car: sexp.Sym("begin"), Cdr: lam.Body}
				continue
			}
			return call(fn, args)
		default:
			return v, nil
		}
	}
}

// call applies fn to args without tail-call optimization (used for builtins
// and for apply/map/filter, whose caller is already a Go function).
func call(fn sexp.Value, args []sexp.Value) (sexp.Value, error) {
	switch f := fn.(type) {
	case *Builtin:
		return f.Fn(args)
	case *Lambda:
		env, err := f.bind(args)
		if err != nil {
			return nil, err
		}
		return Eval(&sexp.Pair{Car: sexp.Sym("begin"), Cdr: f.Body}, env)
	default:
		return nil, fmt.Errorf("not a procedure: %s", sexp.Write(fn))
	}
}

func evalArgs(v sexp.Value, env *Env) ([]sexp.Value, error) {
	var args []sexp.Value
	for {
		p, ok := v.(*sexp.Pair)
		if !ok {
			if _, isNil := v.(sexp.Nil); !isNil {
				return nil, fmt.Errorf("improper argument list: %s", sexp.Write(v))
			}
			return args, nil
		}
		a, err := Eval(p.Car, env)
		if err != nil {
			return nil, err
		}
		args = append(args, a)
		v = p.Cdr
	}
}

// beginOf wraps a slice of forms in a begin so the eval loop runs them in
// order, with the last in tail position.
func beginOf(forms []sexp.Value) sexp.Value {
	return &sexp.Pair{Car: sexp.Sym("begin"), Cdr: sexp.ListTail(sexp.NilVal, forms...)}
}

// isFalse: only #f is false in lark.
func isFalse(v sexp.Value) bool {
	b, ok := v.(sexp.Bool)
	return ok && !bool(b)
}

// evalForm handles one special form. formValue: *vp is the final value.
// formNext: *vp/*envp are the next tail expression. notAForm: the head is an
// ordinary application. Any non-nil error aborts evaluation.
func evalForm(name string, x *sexp.Pair, env *Env, vp *sexp.Value, envp **Env) (formResult, error) {
	elems, _ := sexp.Split(x)
	switch name {
	case "quote":
		if len(elems) != 2 {
			return formValue, fmt.Errorf("quote: bad form: %s", sexp.Write(x))
		}
		*vp = elems[1]
		return formValue, nil
	case "quasiquote":
		if len(elems) != 2 {
			return formValue, fmt.Errorf("quasiquote: bad form: %s", sexp.Write(x))
		}
		v, err := qq(elems[1], env, 1)
		if err != nil {
			return formValue, err
		}
		*vp = v
		return formValue, nil
	case "if":
		if len(elems) != 3 && len(elems) != 4 {
			return formValue, fmt.Errorf("if: bad form: %s", sexp.Write(x))
		}
		t, err := Eval(elems[1], env)
		if err != nil {
			return formValue, err
		}
		if isFalse(t) {
			if len(elems) == 4 {
				*vp = elems[3]
				return formNext, nil
			}
			*vp = sexp.NilVal
			return formValue, nil
		}
		*vp = elems[2]
		return formNext, nil
	case "define":
		if len(elems) < 3 {
			return formValue, fmt.Errorf("define: bad form: %s", sexp.Write(x))
		}
		switch t := elems[1].(type) {
		case sexp.Sym:
			if len(elems) != 3 {
				return formValue, fmt.Errorf("define: bad form: %s", sexp.Write(x))
			}
			val, err := Eval(elems[2], env)
			if err != nil {
				return formValue, err
			}
			env.Define(string(t), val)
		case *sexp.Pair:
			// (define (f args...) body...)
			fname, ok := t.Car.(sexp.Sym)
			if !ok {
				return formValue, fmt.Errorf("define: bad name: %s", sexp.Write(t.Car))
			}
			env.Define(string(fname), &Lambda{Params: t.Cdr, Body: sexp.ListTail(sexp.NilVal, elems[2:]...), Env: env})
		default:
			return formValue, fmt.Errorf("define: bad target: %s", sexp.Write(elems[1]))
		}
		*vp = sexp.NilVal
		return formValue, nil
	case "set!":
		if len(elems) != 3 {
			return formValue, fmt.Errorf("set!: bad form: %s", sexp.Write(x))
		}
		name, ok := elems[1].(sexp.Sym)
		if !ok {
			return formValue, fmt.Errorf("set!: bad name: %s", sexp.Write(elems[1]))
		}
		val, err := Eval(elems[2], env)
		if err != nil {
			return formValue, err
		}
		if err := env.Set(string(name), val); err != nil {
			return formValue, err
		}
		*vp = sexp.NilVal
		return formValue, nil
	case "lambda":
		if len(elems) < 3 {
			return formValue, fmt.Errorf("lambda: bad form: %s", sexp.Write(x))
		}
		*vp = &Lambda{Params: elems[1], Body: sexp.ListTail(sexp.NilVal, elems[2:]...), Env: env}
		return formValue, nil
	case "let", "let*", "letrec":
		if len(elems) < 3 {
			return formValue, fmt.Errorf("%s: bad form: %s", name, sexp.Write(x))
		}
		bindings, ok := sexp.Slice(elems[1])
		if !ok {
			return formValue, fmt.Errorf("%s: bad bindings: %s", name, sexp.Write(elems[1]))
		}
		newEnv := NewEnv(env)
		parsed := make([][2]sexp.Value, 0, len(bindings))
		for _, b := range bindings {
			be, ok := sexp.Slice(b)
			if !ok || len(be) != 2 {
				return formValue, fmt.Errorf("%s: bad binding: %s", name, sexp.Write(b))
			}
			s, ok := be[0].(sexp.Sym)
			if !ok {
				return formValue, fmt.Errorf("%s: bad binding name: %s", name, sexp.Write(be[0]))
			}
			parsed = append(parsed, [2]sexp.Value{s, be[1]})
		}
		switch name {
		case "let":
			for _, p := range parsed {
				val, err := Eval(p[1], env) // all inits see the outer env
				if err != nil {
					return formValue, err
				}
				newEnv.Define(string(p[0].(sexp.Sym)), val)
			}
		case "let*":
			for _, p := range parsed {
				val, err := Eval(p[1], newEnv) // each sees the previous
				if err != nil {
					return formValue, err
				}
				newEnv.Define(string(p[0].(sexp.Sym)), val)
			}
		case "letrec":
			for _, p := range parsed {
				newEnv.Define(string(p[0].(sexp.Sym)), sexp.NilVal)
			}
			for _, p := range parsed {
				val, err := Eval(p[1], newEnv)
				if err != nil {
					return formValue, err
				}
				newEnv.Define(string(p[0].(sexp.Sym)), val)
			}
		}
		*envp = newEnv
		*vp = beginOf(elems[2:])
		return formNext, nil
	case "begin":
		if len(elems) == 1 {
			*vp = sexp.NilVal
			return formValue, nil
		}
		for _, e := range elems[1 : len(elems)-1] {
			if _, err := Eval(e, env); err != nil {
				return formValue, err
			}
		}
		*vp = elems[len(elems)-1]
		return formNext, nil
	case "cond":
		for _, clause := range elems[1:] {
			ce, ok := sexp.Slice(clause)
			if !ok || len(ce) == 0 {
				return formValue, fmt.Errorf("cond: bad clause: %s", sexp.Write(clause))
			}
			isElse := false
			if s, ok := ce[0].(sexp.Sym); ok && string(s) == "else" {
				isElse = true
			}
			var tv sexp.Value = sexp.Bool(true)
			if !isElse {
				var err error
				tv, err = Eval(ce[0], env)
				if err != nil {
					return formValue, err
				}
			}
			if isElse || !isFalse(tv) {
				if len(ce) == 1 {
					*vp = tv
					return formValue, nil
				}
				*vp = beginOf(ce[1:])
				return formNext, nil
			}
		}
		*vp = sexp.NilVal
		return formValue, nil
	case "case":
		if len(elems) < 2 {
			return formValue, fmt.Errorf("case: bad form: %s", sexp.Write(x))
		}
		key, err := Eval(elems[1], env)
		if err != nil {
			return formValue, err
		}
		for _, clause := range elems[2:] {
			ce, ok := sexp.Slice(clause)
			if !ok || len(ce) == 0 {
				return formValue, fmt.Errorf("case: bad clause: %s", sexp.Write(clause))
			}
			isElse := false
			if s, ok := ce[0].(sexp.Sym); ok && string(s) == "else" {
				isElse = true
			}
			hit := isElse
			if !isElse {
				dlist, ok := sexp.Slice(ce[0])
				if !ok {
					return formValue, fmt.Errorf("case: bad datums: %s", sexp.Write(ce[0]))
				}
				for _, d := range dlist {
					if sexp.Equal(key, d) {
						hit = true
						break
					}
				}
			}
			if hit {
				if len(ce) == 1 {
					*vp = sexp.NilVal
					return formValue, nil
				}
				*vp = beginOf(ce[1:])
				return formNext, nil
			}
		}
		*vp = sexp.NilVal
		return formValue, nil
	case "and":
		if len(elems) == 1 {
			*vp = sexp.Bool(true)
			return formValue, nil
		}
		for _, e := range elems[1 : len(elems)-1] {
			val, err := Eval(e, env)
			if err != nil {
				return formValue, err
			}
			if isFalse(val) {
				*vp = val
				return formValue, nil
			}
		}
		*vp = elems[len(elems)-1]
		return formNext, nil
	case "or":
		if len(elems) == 1 {
			*vp = sexp.Bool(false)
			return formValue, nil
		}
		for _, e := range elems[1 : len(elems)-1] {
			val, err := Eval(e, env)
			if err != nil {
				return formValue, err
			}
			if !isFalse(val) {
				*vp = val
				return formValue, nil
			}
		}
		*vp = elems[len(elems)-1]
		return formNext, nil
	case "when", "unless":
		if len(elems) < 3 {
			return formValue, fmt.Errorf("%s: bad form: %s", name, sexp.Write(x))
		}
		t, err := Eval(elems[1], env)
		if err != nil {
			return formValue, err
		}
		fire := !isFalse(t)
		if name == "unless" {
			fire = !fire
		}
		if !fire {
			*vp = sexp.NilVal
			return formValue, nil
		}
		*vp = beginOf(elems[2:])
		return formNext, nil
	case "while":
		if len(elems) < 2 {
			return formValue, fmt.Errorf("while: bad form: %s", sexp.Write(x))
		}
		for {
			t, err := Eval(elems[1], env)
			if err != nil {
				return formValue, err
			}
			if isFalse(t) {
				*vp = sexp.NilVal
				return formValue, nil
			}
			for _, e := range elems[2:] {
				if _, err := Eval(e, env); err != nil {
					return formValue, err
				}
			}
		}
	default:
		return notAForm, nil
	}
}

// qq evaluates a quasiquote template at the given nesting depth.
func qq(v sexp.Value, env *Env, depth int) (sexp.Value, error) {
	p, ok := v.(*sexp.Pair)
	if !ok {
		return v, nil // atoms are data inside quasiquote
	}
	if h, ok := p.Car.(sexp.Sym); ok {
		elems, _ := sexp.Split(v)
		switch string(h) {
		case "unquote":
			if len(elems) != 2 {
				return nil, fmt.Errorf("unquote: bad form: %s", sexp.Write(v))
			}
			if depth == 1 {
				return Eval(elems[1], env)
			}
			inner, err := qq(elems[1], env, depth-1)
			if err != nil {
				return nil, err
			}
			return sexp.List(sexp.Sym("unquote"), inner), nil
		case "quasiquote":
			if len(elems) != 2 {
				return nil, fmt.Errorf("quasiquote: bad form: %s", sexp.Write(v))
			}
			inner, err := qq(elems[1], env, depth+1)
			if err != nil {
				return nil, err
			}
			return sexp.List(sexp.Sym("quasiquote"), inner), nil
		case "unquote-splicing":
			return nil, fmt.Errorf("unquote-splicing outside a list: %s", sexp.Write(v))
		}
	}
	var out []sexp.Value
	cur := sexp.Value(p)
	tail := sexp.Value(sexp.NilVal)
	for {
		pp, ok := cur.(*sexp.Pair)
		if !ok {
			tail = cur
			break
		}
		elem := pp.Car
		if ep, ok := elem.(*sexp.Pair); ok {
			if h, ok := ep.Car.(sexp.Sym); ok && string(h) == "unquote-splicing" && depth == 1 {
				ee, _ := sexp.Split(elem)
				if len(ee) != 2 {
					return nil, fmt.Errorf("unquote-splicing: bad form: %s", sexp.Write(elem))
				}
				spliced, err := Eval(ee[1], env)
				if err != nil {
					return nil, err
				}
				se, st := sexp.Split(spliced)
				if _, ok := st.(sexp.Nil); !ok {
					return nil, fmt.Errorf("unquote-splicing: not a list: %s", sexp.Write(spliced))
				}
				out = append(out, se...)
				cur = pp.Cdr
				continue
			}
		}
		e, err := qq(elem, env, depth)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
		cur = pp.Cdr
	}
	if _, ok := tail.(sexp.Nil); !ok {
		t, err := qq(tail, env, depth)
		if err != nil {
			return nil, err
		}
		tail = t
	}
	return sexp.ListTail(tail, out...), nil
}
