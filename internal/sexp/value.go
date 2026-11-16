// Package sexp defines lark's data model: the values that code is made of.
// Code and data share one representation, so the reader, printer, and
// evaluator all operate on Value — an agent holding a Value can print it,
// walk it, rebuild it, and evaluate it with no parser/codegen gap.
package sexp

import (
	"math"
	"reflect"
)

// Value is any lark datum. The data types (Nil, Bool, Int, Float, Str, Sym,
// Pair, Vec) live in this package; the evaluator adds procedures and errors.
// Any Value must be printable via Write.
type Value interface {
	LarkValue()
}

// Nil is the empty list and the sole nil value: ().
type Nil struct{}

func (Nil) LarkValue() {}

// NilVal is the singleton nil value.
var NilVal = Nil{}

// Bool is #t or #f.
type Bool bool

func (Bool) LarkValue() {}

// Int is a 64-bit integer.
type Int int64

func (Int) LarkValue() {}

// Float is a 64-bit IEEE float.
type Float float64

func (Float) LarkValue() {}

// Str is a string.
type Str string

func (Str) LarkValue() {}

// Sym is a symbol.
type Sym string

func (Sym) LarkValue() {}

// Pair is a cons cell. A proper list is a chain of Pairs ending in NilVal.
type Pair struct {
	Car Value
	Cdr Value
}

func (*Pair) LarkValue() {}

// Vec is a vector: #(a b c).
type Vec struct {
	Elems []Value
}

func (*Vec) LarkValue() {}

// List builds a proper list from elems.
func List(elems ...Value) Value { return ListTail(NilVal, elems...) }

// ListTail builds a list of elems with the given tail (NilVal for proper).
func ListTail(tail Value, elems ...Value) Value {
	v := tail
	for i := len(elems) - 1; i >= 0; i-- {
		v = &Pair{Car: elems[i], Cdr: v}
	}
	return v
}

// Split decomposes a pair chain into its elements and its final non-pair
// tail (NilVal for a proper list).
func Split(v Value) (elems []Value, tail Value) {
	tail = v
	for {
		p, ok := tail.(*Pair)
		if !ok {
			return elems, tail
		}
		elems = append(elems, p.Car)
		tail = p.Cdr
	}
}

// IsProperList reports whether v is a proper list (ends in NilVal).
func IsProperList(v Value) bool {
	_, tail := Split(v)
	_, ok := tail.(Nil)
	return ok
}

// Slice returns the elements of a proper list, or ok=false.
func Slice(v Value) (elems []Value, ok bool) {
	elems, tail := Split(v)
	if _, ok := tail.(Nil); !ok {
		return nil, false
	}
	return elems, true
}

// Equal is deep structural equality (lark's equal?). NaN equals NaN.
func Equal(a, b Value) bool {
	switch av := a.(type) {
	case Nil:
		_, ok := b.(Nil)
		return ok
	case Bool:
		bv, ok := b.(Bool)
		return ok && av == bv
	case Int:
		bv, ok := b.(Int)
		return ok && av == bv
	case Float:
		bv, ok := b.(Float)
		return ok && (av == bv || (math.IsNaN(float64(av)) && math.IsNaN(float64(bv))))
	case Str:
		bv, ok := b.(Str)
		return ok && av == bv
	case Sym:
		bv, ok := b.(Sym)
		return ok && av == bv
	case *Pair:
		bv, ok := b.(*Pair)
		return ok && Equal(av.Car, bv.Car) && Equal(av.Cdr, bv.Cdr)
	case *Vec:
		bv, ok := b.(*Vec)
		if !ok || len(av.Elems) != len(bv.Elems) {
			return false
		}
		for i := range av.Elems {
			if !Equal(av.Elems[i], bv.Elems[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a, b)
	}
}
