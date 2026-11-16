package sexp

import (
	"math/rand"
	"testing"
)

// randValue builds a random datum. Strings deliberately include characters
// that need escaping so the round-trip tests exercise writeString.
func randValue(r *rand.Rand, depth int) Value {
	atom := func() Value {
		switch r.Intn(7) {
		case 0:
			return NilVal
		case 1:
			return Bool(r.Intn(2) == 0)
		case 2:
			return Int(r.Int63n(2000) - 1000)
		case 3:
			return Float(r.NormFloat64() * 100)
		case 4:
			return Float(float64(r.Intn(50))) // integral floats: "42.0" case
		case 5:
			return Str(randStr(r))
		default:
			return Sym(randSym(r))
		}
	}
	if depth <= 0 {
		return atom()
	}
	switch r.Intn(3) {
	case 0: // list, sometimes dotted
		n := r.Intn(6)
		elems := make([]Value, n)
		for i := range elems {
			elems[i] = randValue(r, depth-1)
		}
		tail := Value(NilVal)
		if r.Intn(4) == 0 {
			tail = randValue(r, 0)
		}
		return ListTail(tail, elems...)
	case 1: // vector
		n := r.Intn(5)
		elems := make([]Value, n)
		for i := range elems {
			elems[i] = randValue(r, depth-1)
		}
		return &Vec{Elems: elems}
	default:
		return atom()
	}
}

func randStr(r *rand.Rand) string {
	const chars = `ab " \n\t\ 01`
	n := r.Intn(12)
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return string(b)
}

func randSym(r *rand.Rand) string {
	const chars = "abcXYZ-+*/<>=!?"
	n := 1 + r.Intn(10)
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return string(b)
}

// TestRoundTrip is the plan's core property: Parse(Write(v)) == v for any
// datum, so what an agent writes is exactly what runs.
func TestRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 5000; i++ {
		v := randValue(r, 4)
		s := Write(v)
		back, err := Parse(s)
		if err != nil {
			t.Fatalf("iter %d: Parse(Write(v)) error for %q: %v", i, s, err)
		}
		if !Equal(back, v) {
			t.Fatalf("iter %d: round-trip mismatch:\n got %s\nwant %s", i, s, Write(v))
		}
	}
}

// TestWriteIdempotent: Write(Parse(Write(v))) == Write(v).
func TestWriteIdempotent(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for i := 0; i < 5000; i++ {
		v := randValue(r, 4)
		s1 := Write(v)
		back, err := Parse(s1)
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if s2 := Write(back); s2 != s1 {
			t.Fatalf("iter %d: not idempotent:\n%s\n%s", i, s1, s2)
		}
	}
}

// TestPrettyFixedPoint is the plan's other property: the pretty printer is a
// fixed point, so emitted code is stable across runs and diffs cleanly.
func TestPrettyFixedPoint(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	for i := 0; i < 5000; i++ {
		v := randValue(r, 4)
		p1 := Pretty(v)
		back, err := Parse(p1)
		if err != nil {
			t.Fatalf("iter %d: Pretty output does not re-parse: %v\n%s", i, err, p1)
		}
		if p2 := Pretty(back); p2 != p1 {
			t.Fatalf("iter %d: pretty not a fixed point:\n--- first ---\n%s\n--- second ---\n%s", i, p1, p2)
		}
	}
}
