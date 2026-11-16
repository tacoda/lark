package eval

import (
	"fmt"

	"github.com/tacoda/lark/internal/sexp"
)

// eqv compares atoms by type and value; pairs/vectors by identity.
func eqv(a, b sexp.Value) bool {
	switch av := a.(type) {
	case sexp.Nil:
		_, ok := b.(sexp.Nil)
		return ok
	case sexp.Bool:
		bv, ok := b.(sexp.Bool)
		return ok && av == bv
	case sexp.Int:
		bv, ok := b.(sexp.Int)
		return ok && av == bv
	case sexp.Float:
		bv, ok := b.(sexp.Float)
		return ok && av == bv
	case sexp.Str:
		bv, ok := b.(sexp.Str)
		return ok && av == bv
	case sexp.Sym:
		bv, ok := b.(sexp.Sym)
		return ok && av == bv
	default:
		return a == b // *Pair, *Vec, and host types: identity
	}
}

func defMisc() {
	one := func(name string, pred func(sexp.Value) bool) {
		def(name, func(args []sexp.Value) (sexp.Value, error) {
			if err := argErr(name, args, 1, 1); err != nil {
				return nil, err
			}
			return sexp.Bool(pred(args[0])), nil
		})
	}

	// equality
	def("eq?", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("eq?", args, 2, 2); err != nil {
			return nil, err
		}
		a, b := args[0], args[1]
		switch a.(type) {
		case *sexp.Pair, *sexp.Vec:
			return sexp.Bool(a == b), nil // identity
		}
		return sexp.Bool(eqv(a, b)), nil
	})
	def("eqv?", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("eqv?", args, 2, 2); err != nil {
			return nil, err
		}
		return sexp.Bool(eqv(args[0], args[1])), nil
	})
	def("equal?", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("equal?", args, 2, 2); err != nil {
			return nil, err
		}
		return sexp.Bool(sexp.Equal(args[0], args[1])), nil
	})
	def("not", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("not", args, 1, 1); err != nil {
			return nil, err
		}
		return sexp.Bool(isFalse(args[0])), nil
	})

	// predicates
	one("null?", func(v sexp.Value) bool { _, ok := v.(sexp.Nil); return ok })
	one("pair?", func(v sexp.Value) bool { _, ok := v.(*sexp.Pair); return ok })
	one("list?", sexp.IsProperList)
	one("symbol?", func(v sexp.Value) bool { _, ok := v.(sexp.Sym); return ok })
	one("string?", func(v sexp.Value) bool { _, ok := v.(sexp.Str); return ok })
	one("number?", func(v sexp.Value) bool { _, ok := numF(v); return ok })
	one("boolean?", func(v sexp.Value) bool { _, ok := v.(sexp.Bool); return ok })
	one("procedure?", func(v sexp.Value) bool {
		switch v.(type) {
		case *Lambda, *Builtin:
			return true
		}
		return false
	})
	one("vector?", func(v sexp.Value) bool { _, ok := v.(*sexp.Vec); return ok })
	one("error?", func(v sexp.Value) bool { _, ok := v.(*Error); return ok })
	one("atom?", func(v sexp.Value) bool { _, ok := v.(*sexp.Pair); return !ok })

	// symbols & strings
	def("symbol->string", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("symbol->string", args, 1, 1); err != nil {
			return nil, err
		}
		s, ok := asSym(args[0])
		if !ok {
			return nil, fmt.Errorf("symbol->string: not a symbol: %s", sexp.Write(args[0]))
		}
		return sexp.Str(s), nil
	})
	def("string->symbol", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("string->symbol", args, 1, 1); err != nil {
			return nil, err
		}
		s, ok := asStr(args[0])
		if !ok {
			return nil, fmt.Errorf("string->symbol: not a string: %s", sexp.Write(args[0]))
		}
		return sexp.Sym(s), nil
	})
	def("string-append", func(args []sexp.Value) (sexp.Value, error) {
		out := ""
		for _, a := range args {
			s, ok := asStr(a)
			if !ok {
				return nil, fmt.Errorf("string-append: not a string: %s", sexp.Write(a))
			}
			out += s
		}
		return sexp.Str(out), nil
	})
	def("string-length", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("string-length", args, 1, 1); err != nil {
			return nil, err
		}
		s, ok := asStr(args[0])
		if !ok {
			return nil, fmt.Errorf("string-length: not a string: %s", sexp.Write(args[0]))
		}
		return sexp.Int(int64(len(s))), nil // bytes
	})
	def("substring", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("substring", args, 2, 3); err != nil {
			return nil, err
		}
		s, ok := asStr(args[0])
		if !ok {
			return nil, fmt.Errorf("substring: not a string: %s", sexp.Write(args[0]))
		}
		start, ok1 := asInt(args[1])
		end := int64(len(s))
		if len(args) == 3 {
			var ok2 bool
			end, ok2 = asInt(args[2])
			if !ok2 {
				return nil, fmt.Errorf("substring: not an integer: %s", sexp.Write(args[2]))
			}
		}
		if !ok1 || start < 0 || end < start || end > int64(len(s)) {
			return nil, fmt.Errorf("substring: bad range %s..%s for length %d", sexp.Write(args[1]), sexp.Write(args[len(args)-1]), len(s))
		}
		return sexp.Str(s[start:end]), nil
	})

	// vectors
	def("vector", func(args []sexp.Value) (sexp.Value, error) {
		return &sexp.Vec{Elems: append([]sexp.Value(nil), args...)}, nil
	})
	def("vector-ref", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("vector-ref", args, 2, 2); err != nil {
			return nil, err
		}
		vec, ok := asVec(args[0])
		if !ok {
			return nil, fmt.Errorf("vector-ref: not a vector: %s", sexp.Write(args[0]))
		}
		i, ok := asInt(args[1])
		if !ok || i < 0 || i >= int64(len(vec.Elems)) {
			return nil, fmt.Errorf("vector-ref: index out of range: %s", sexp.Write(args[1]))
		}
		return vec.Elems[i], nil
	})
	def("vector-set!", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("vector-set!", args, 3, 3); err != nil {
			return nil, err
		}
		vec, ok := asVec(args[0])
		if !ok {
			return nil, fmt.Errorf("vector-set!: not a vector: %s", sexp.Write(args[0]))
		}
		i, ok := asInt(args[1])
		if !ok || i < 0 || i >= int64(len(vec.Elems)) {
			return nil, fmt.Errorf("vector-set!: index out of range: %s", sexp.Write(args[1]))
		}
		vec.Elems[i] = args[2]
		return sexp.NilVal, nil
	})
	def("vector-length", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("vector-length", args, 1, 1); err != nil {
			return nil, err
		}
		vec, ok := asVec(args[0])
		if !ok {
			return nil, fmt.Errorf("vector-length: not a vector: %s", sexp.Write(args[0]))
		}
		return sexp.Int(int64(len(vec.Elems))), nil
	})
	def("list->vector", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("list->vector", args, 1, 1); err != nil {
			return nil, err
		}
		elems, ok := sexp.Slice(args[0])
		if !ok {
			return nil, fmt.Errorf("list->vector: not a list: %s", sexp.Write(args[0]))
		}
		return &sexp.Vec{Elems: append([]sexp.Value(nil), elems...)}, nil
	})
	def("vector->list", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("vector->list", args, 1, 1); err != nil {
			return nil, err
		}
		vec, ok := asVec(args[0])
		if !ok {
			return nil, fmt.Errorf("vector->list: not a vector: %s", sexp.Write(args[0]))
		}
		return sexp.List(vec.Elems...), nil
	})

	// errors as data
	def("error", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("error", args, 1, -1); err != nil {
			return nil, err
		}
		msg, ok := asStr(args[0])
		if !ok {
			return nil, fmt.Errorf("error: want a string message, got %s", sexp.Write(args[0]))
		}
		return &Error{Msg: msg, Data: sexp.List(args[1:]...)}, nil
	})
}
