package eval

import (
	"fmt"

	"github.com/tacoda/lark/internal/sexp"
)

func defListOps() {
	def("cons", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("cons", args, 2, 2); err != nil {
			return nil, err
		}
		return &sexp.Pair{Car: args[0], Cdr: args[1]}, nil
	})
	def("car", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("car", args, 1, 1); err != nil {
			return nil, err
		}
		p, ok := args[0].(*sexp.Pair)
		if !ok {
			return nil, fmt.Errorf("car: not a pair: %s", sexp.Write(args[0]))
		}
		return p.Car, nil
	})
	def("cdr", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("cdr", args, 1, 1); err != nil {
			return nil, err
		}
		p, ok := args[0].(*sexp.Pair)
		if !ok {
			return nil, fmt.Errorf("cdr: not a pair: %s", sexp.Write(args[0]))
		}
		return p.Cdr, nil
	})
	def("list", func(args []sexp.Value) (sexp.Value, error) {
		return sexp.List(args...), nil
	})
	def("append", func(args []sexp.Value) (sexp.Value, error) {
		if len(args) == 0 {
			return sexp.NilVal, nil
		}
		var out []sexp.Value
		for _, a := range args[:len(args)-1] {
			e, ok := sexp.Slice(a)
			if !ok {
				return nil, fmt.Errorf("append: not a list: %s", sexp.Write(a))
			}
			out = append(out, e...)
		}
		// the last arg becomes the tail as-is (may be improper)
		return sexp.ListTail(args[len(args)-1], out...), nil
	})
	def("reverse", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("reverse", args, 1, 1); err != nil {
			return nil, err
		}
		elems, ok := sexp.Slice(args[0])
		if !ok {
			return nil, fmt.Errorf("reverse: not a list: %s", sexp.Write(args[0]))
		}
		var out sexp.Value = sexp.NilVal
		for _, e := range elems {
			out = &sexp.Pair{Car: e, Cdr: out}
		}
		return out, nil
	})
	def("length", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("length", args, 1, 1); err != nil {
			return nil, err
		}
		elems, ok := sexp.Slice(args[0])
		if !ok {
			return nil, fmt.Errorf("length: not a list: %s", sexp.Write(args[0]))
		}
		return sexp.Int(int64(len(elems))), nil
	})
	def("nth", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("nth", args, 2, 2); err != nil {
			return nil, err
		}
		elems, ok := sexp.Slice(args[0])
		if !ok {
			return nil, fmt.Errorf("nth: not a list: %s", sexp.Write(args[0]))
		}
		i, ok := asInt(args[1])
		if !ok || i < 0 || i >= int64(len(elems)) {
			return nil, fmt.Errorf("nth: index out of range: %s", sexp.Write(args[1]))
		}
		return elems[i], nil
	})
	def("map", func(args []sexp.Value) (sexp.Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("map: want a procedure and at least one list, got %d args", len(args))
		}
		fn := args[0]
		var lists [][]sexp.Value
		min := -1
		for _, l := range args[1:] {
			e, ok := sexp.Slice(l)
			if !ok {
				return nil, fmt.Errorf("map: not a list: %s", sexp.Write(l))
			}
			if min < 0 || len(e) < min {
				min = len(e)
			}
			lists = append(lists, e)
		}
		out := make([]sexp.Value, min)
		for i := 0; i < min; i++ {
			callArgs := make([]sexp.Value, len(lists))
			for j, e := range lists {
				callArgs[j] = e[i]
			}
			v, err := call(fn, callArgs)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return sexp.List(out...), nil
	})
	def("filter", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("filter", args, 2, 2); err != nil {
			return nil, err
		}
		elems, ok := sexp.Slice(args[1])
		if !ok {
			return nil, fmt.Errorf("filter: not a list: %s", sexp.Write(args[1]))
		}
		var out []sexp.Value
		for _, e := range elems {
			v, err := call(args[0], []sexp.Value{e})
			if err != nil {
				return nil, err
			}
			if !isFalse(v) {
				out = append(out, e)
			}
		}
		return sexp.List(out...), nil
	})
	def("fold-left", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("fold-left", args, 3, 3); err != nil {
			return nil, err
		}
		elems, ok := sexp.Slice(args[2])
		if !ok {
			return nil, fmt.Errorf("fold-left: not a list: %s", sexp.Write(args[2]))
		}
		acc := args[1]
		for _, e := range elems {
			v, err := call(args[0], []sexp.Value{acc, e})
			if err != nil {
				return nil, err
			}
			acc = v
		}
		return acc, nil
	})
	def("apply", func(args []sexp.Value) (sexp.Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("apply: want a procedure and an argument list, got %d args", len(args))
		}
		last, ok := sexp.Slice(args[len(args)-1])
		if !ok {
			return nil, fmt.Errorf("apply: last arg not a list: %s", sexp.Write(args[len(args)-1]))
		}
		callArgs := append(append([]sexp.Value{}, args[1:len(args)-1]...), last...)
		return call(args[0], callArgs)
	})
}
