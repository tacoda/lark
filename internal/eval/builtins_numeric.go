package eval

import (
	"fmt"

	"github.com/tacoda/lark/internal/sexp"
)

// foldNum folds numeric args left to right. If every arg is an Int the
// result is an Int (int64 arithmetic); if any arg is a Float the result is a
// Float. identity is the result of the zero-arg case.
func foldNum(name string, args []sexp.Value, identity int64,
	intOp func(a, b int64) int64, floatOp func(a, b float64) float64) (sexp.Value, error) {
	if len(args) == 0 {
		return sexp.Int(identity), nil
	}
	allInt := true
	ii := int64(identity)
	ff := float64(identity)
	for k, a := range args {
		i, iok := asInt(a)
		f, fok := asFloat(a)
		if !iok && !fok {
			return nil, notNum(name, a)
		}
		v := float64(i)
		if fok {
			v = f
			allInt = false
		}
		if k == 0 {
			if iok {
				ii = i
			}
			ff = v
			continue
		}
		if iok && allInt {
			ii = intOp(ii, i)
		}
		ff = floatOp(ff, v)
	}
	if allInt {
		return sexp.Int(ii), nil
	}
	return sexp.Float(ff), nil
}

func defNumeric() {
	def("+", func(args []sexp.Value) (sexp.Value, error) {
		return foldNum("+", args, 0,
			func(a, b int64) int64 { return a + b },
			func(a, b float64) float64 { return a + b })
	})
	def("*", func(args []sexp.Value) (sexp.Value, error) {
		return foldNum("*", args, 1,
			func(a, b int64) int64 { return a * b },
			func(a, b float64) float64 { return a * b })
	})
	def("-", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("-", args, 1, -1); err != nil {
			return nil, err
		}
		if len(args) == 1 {
			if i, ok := asInt(args[0]); ok {
				return sexp.Int(-i), nil
			}
			if f, ok := asFloat(args[0]); ok {
				return sexp.Float(-f), nil
			}
			return nil, notNum("-", args[0])
		}
		return foldNum("-", args, 0,
			func(a, b int64) int64 { return a - b },
			func(a, b float64) float64 { return a - b })
	})
	// / on all-int args is truncating integer division; any float makes the
	// whole chain float. (/ x) is 1/x.
	def("/", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("/", args, 1, -1); err != nil {
			return nil, err
		}
		if len(args) == 1 {
			if i, ok := asInt(args[0]); ok {
				if i == 0 {
					return nil, errDivZero
				}
				return sexp.Int(1 / i), nil
			}
			if f, ok := asFloat(args[0]); ok {
				return sexp.Float(1 / f), nil
			}
			return nil, notNum("/", args[0])
		}
		allInt := true
		var ii int64
		var ff float64
		for k, a := range args {
			i, iok := asInt(a)
			f, fok := asFloat(a)
			if !iok && !fok {
				return nil, notNum("/", a)
			}
			v := float64(i)
			if fok {
				v = f
				allInt = false
			}
			if k == 0 {
				if iok {
					ii = i
				}
				ff = v
				continue
			}
			if v == 0 {
				return nil, errDivZero
			}
			if iok && allInt {
				ii /= i
			}
			ff /= v
		}
		if allInt {
			return sexp.Int(ii), nil
		}
		return sexp.Float(ff), nil
	})

	cmp := func(name string, keep func(a, b float64) bool) func([]sexp.Value) (sexp.Value, error) {
		return func(args []sexp.Value) (sexp.Value, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("%s: want at least 2 args, got %d", name, len(args))
			}
			for i := 0; i+1 < len(args); i++ {
				a, ok1 := numF(args[i])
				b, ok2 := numF(args[i+1])
				if !ok1 {
					return nil, notNum(name, args[i])
				}
				if !ok2 {
					return nil, notNum(name, args[i+1])
				}
				if !keep(a, b) {
					return sexp.Bool(false), nil
				}
			}
			return sexp.Bool(true), nil
		}
	}
	def("<", cmp("<", func(a, b float64) bool { return a < b }))
	def("<=", cmp("<=", func(a, b float64) bool { return a <= b }))
	def(">", cmp(">", func(a, b float64) bool { return a > b }))
	def(">=", cmp(">=", func(a, b float64) bool { return a >= b }))
	def("=", cmp("=", func(a, b float64) bool { return a == b }))

	intPair := func(name string, op func(a, b int64) int64) func([]sexp.Value) (sexp.Value, error) {
		return func(args []sexp.Value) (sexp.Value, error) {
			if err := argErr(name, args, 2, 2); err != nil {
				return nil, err
			}
			a, ok1 := asInt(args[0])
			b, ok2 := asInt(args[1])
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("%s: want integers, got %s %s", name, sexp.Write(args[0]), sexp.Write(args[1]))
			}
			if b == 0 {
				return nil, errDivZero
			}
			return sexp.Int(op(a, b)), nil
		}
	}
	def("quo", intPair("quo", func(a, b int64) int64 { return a / b })) // truncated
	def("rem", intPair("rem", func(a, b int64) int64 { return a % b })) // truncated
	def("mod", intPair("mod", func(a, b int64) int64 {                  // floored
		r := a % b
		if r != 0 && (r < 0) != (b < 0) {
			r += b
		}
		return r
	}))

	def("abs", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("abs", args, 1, 1); err != nil {
			return nil, err
		}
		if i, ok := asInt(args[0]); ok {
			if i < 0 {
				return sexp.Int(-i), nil
			}
			return sexp.Int(i), nil
		}
		if f, ok := asFloat(args[0]); ok {
			if f < 0 {
				return sexp.Float(-f), nil
			}
			return sexp.Float(f), nil
		}
		return nil, notNum("abs", args[0])
	})
	def("min", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("min", args, 1, -1); err != nil {
			return nil, err
		}
		return foldNum("min", args, 0, func(a, b int64) int64 {
			if b < a {
				return b
			}
			return a
		},
			func(a, b float64) float64 {
				if b < a {
					return b
				}
				return a
			})
	})
	def("max", func(args []sexp.Value) (sexp.Value, error) {
		if err := argErr("max", args, 1, -1); err != nil {
			return nil, err
		}
		return foldNum("max", args, 0, func(a, b int64) int64 {
			if b > a {
				return b
			}
			return a
		},
			func(a, b float64) float64 {
				if b > a {
					return b
				}
				return a
			})
	})
}
