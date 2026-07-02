// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"math/big"
)

// rubyEqual models Ruby `==` over the value model. Integers compare across
// int/int64/*big.Int and numerically against floats; strings, symbols, bools,
// nil compare by value; arrays and hashes compare element-wise; objects compare
// by identity unless the host marks them equal via matching ID.
func rubyEqual(a, b any) bool {
	// Numeric cross-type equality (Integer == Float, big == small).
	if an, aok := asBigFloat(a); aok {
		if bn, bok := asBigFloat(b); bok {
			return an.Cmp(bn) == 0
		}
		return false
	}
	switch x := a.(type) {
	case nil:
		return b == nil
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	case Symbol:
		y, ok := b.(Symbol)
		return ok && x == y
	case Class:
		y, ok := b.(Class)
		return ok && x == y
	case Module:
		y, ok := b.(Module)
		return ok && x == y
	case *Regexp:
		y, ok := b.(*Regexp)
		return ok && x.Source == y.Source && x.Flags == y.Flags
	case *Range:
		y, ok := b.(*Range)
		return ok && x.Exclusive == y.Exclusive && rubyEqual(x.Begin, y.Begin) && rubyEqual(x.End, y.End)
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !rubyEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	case *Hash:
		y, ok := b.(*Hash)
		if !ok || x.Len() != y.Len() {
			return false
		}
		for _, p := range x.pairs {
			bv, found := y.Get(p.Key)
			if !found || !rubyEqual(p.Val, bv) {
				return false
			}
		}
		return true
	case *Object:
		return rubyIdentical(a, b)
	}
	return a == b
}

// rubyEql models Ruby `eql?`: like `==` but Integer and Float are never eql
// (1.eql?(1.0) is false), and only same-type numbers are eql.
func rubyEql(a, b any) bool {
	aInt := isRubyInteger(a)
	bInt := isRubyInteger(b)
	aFloat := isRubyFloat(a)
	bFloat := isRubyFloat(b)
	if (aInt || aFloat) || (bInt || bFloat) {
		if aInt && bInt {
			ai, _ := asBig(a)
			bi, _ := asBig(b)
			return ai.Cmp(bi) == 0
		}
		if aFloat && bFloat {
			return toFloat(a) == toFloat(b)
		}
		return false // mixed Integer/Float or number vs non-number
	}
	return rubyEqual(a, b)
}

// rubyIdentical models Ruby `equal?` (object identity). Value types are
// identical when equal; objects are identical when they carry the same non-zero
// ID, else when the same pointer.
func rubyIdentical(a, b any) bool {
	oa, aok := a.(*Object)
	ob, bok := b.(*Object)
	if aok && bok {
		if oa.ID != 0 || ob.ID != 0 {
			return oa.ID == ob.ID
		}
		return oa == ob
	}
	if aok != bok {
		return false
	}
	// For immutable value types (nil, bool, small int, symbol) Ruby equal? is
	// value identity. Strings/arrays/hashes are distinct objects unless the same
	// pointer; the host distinguishes them via *Object, so here treat plain
	// strings/arrays as identical only when the underlying values are the same.
	switch a.(type) {
	case string, []any, *Hash, *Regexp, *Range:
		// Distinct mutable objects: identical only if truly the same value; the
		// host passes *Object when it needs pointer identity, so equal-by-value
		// here would over-match. RSpec's equal on two equal-but-distinct strings
		// fails, so report false unless they are the very same interface value.
		return false
	}
	return rubyEqual(a, b)
}

func isRubyInteger(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint64, *big.Int:
		return true
	}
	return false
}

func isRubyFloat(v any) bool {
	switch v.(type) {
	case float32, float64:
		return true
	}
	return false
}

// asBig returns v as a *big.Int when it is an integer.
func asBig(v any) (*big.Int, bool) {
	switch x := v.(type) {
	case int:
		return big.NewInt(int64(x)), true
	case int8:
		return big.NewInt(int64(x)), true
	case int16:
		return big.NewInt(int64(x)), true
	case int32:
		return big.NewInt(int64(x)), true
	case int64:
		return big.NewInt(x), true
	case uint:
		return new(big.Int).SetUint64(uint64(x)), true
	case uint64:
		return new(big.Int).SetUint64(x), true
	case *big.Int:
		return x, true
	}
	return nil, false
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float32:
		return float64(x)
	case float64:
		return x
	}
	return 0
}

// asBigFloat returns v as a big.Float when it is any Ruby numeric, for
// cross-type == and comparison.
func asBigFloat(v any) (*big.Float, bool) {
	if bi, ok := asBig(v); ok {
		return new(big.Float).SetInt(bi), true
	}
	if isRubyFloat(v) {
		return big.NewFloat(toFloat(v)), true
	}
	return nil, false
}

// rubyCompare returns -1, 0, 1 for a<=>b over numerics and strings, and a
// second bool reporting whether the two are comparable at all (Ruby <=> returns
// nil for incomparable operands).
func rubyCompare(a, b any) (int, bool) {
	if af, aok := asBigFloat(a); aok {
		if bf, bok := asBigFloat(b); bok {
			return af.Cmp(bf), true
		}
		return 0, false
	}
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		switch {
		case as < bs:
			return -1, true
		case as > bs:
			return 1, true
		default:
			return 0, true
		}
	}
	return 0, false
}
