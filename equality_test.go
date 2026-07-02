// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import "testing"

func TestRubyEqual(t *testing.T) {
	if !rubyEqual(nil, nil) || rubyEqual(nil, 1) {
		t.Fatal("nil")
	}
	if !rubyEqual(true, true) || rubyEqual(true, false) || rubyEqual(true, 1) {
		t.Fatal("bool")
	}
	if !rubyEqual("a", "a") || rubyEqual("a", "b") || rubyEqual("a", 1) {
		t.Fatal("string")
	}
	if !rubyEqual(Symbol("a"), Symbol("a")) || rubyEqual(Symbol("a"), Symbol("b")) {
		t.Fatal("symbol")
	}
	if !rubyEqual(Class("C"), Class("C")) || rubyEqual(Class("C"), Class("D")) {
		t.Fatal("class")
	}
	if !rubyEqual(Module("M"), Module("M")) || rubyEqual(Module("M"), 1) {
		t.Fatal("module")
	}
	if !rubyEqual(&Regexp{Source: "x"}, &Regexp{Source: "x"}) || rubyEqual(&Regexp{Source: "x"}, &Regexp{Source: "y"}) {
		t.Fatal("regexp")
	}
	if rubyEqual(&Regexp{Source: "x"}, 1) {
		t.Fatal("regexp type")
	}
	if !rubyEqual(&Range{Begin: 1, End: 3}, &Range{Begin: 1, End: 3}) {
		t.Fatal("range eq")
	}
	if rubyEqual(&Range{Begin: 1, End: 3}, &Range{Begin: 1, End: 4}) || rubyEqual(&Range{Begin: 1, End: 3}, 1) {
		t.Fatal("range ne")
	}
	if !rubyEqual([]any{1, 2}, []any{1, 2}) || rubyEqual([]any{1}, []any{1, 2}) || rubyEqual([]any{1}, []any{2}) || rubyEqual([]any{1}, 1) {
		t.Fatal("array")
	}
	h1 := hashOf(Symbol("a"), 1)
	h2 := hashOf(Symbol("a"), 1)
	if !rubyEqual(h1, h2) {
		t.Fatal("hash eq")
	}
	if rubyEqual(h1, hashOf(Symbol("a"), 2)) || rubyEqual(h1, hashOf(Symbol("b"), 1)) || rubyEqual(h1, hashOf()) || rubyEqual(h1, 1) {
		t.Fatal("hash ne")
	}
	// numeric cross-type
	if !rubyEqual(1, 1.0) || !rubyEqual(bigOf("5"), 5) || rubyEqual(1, "x") {
		t.Fatal("numeric")
	}
	// default path: identical unmodelled values.
	if !rubyEqual(uintptr(1), uintptr(1)) {
		t.Fatal("default eq")
	}
}

func TestRubyEql(t *testing.T) {
	if !rubyEql(1, 1) || rubyEql(1, 1.0) || rubyEql(1.0, 1) {
		t.Fatal("int/float")
	}
	if !rubyEql(1.5, 1.5) || rubyEql(1.5, 2.5) {
		t.Fatal("float")
	}
	if rubyEql(1, "x") {
		t.Fatal("num vs non-num")
	}
	if !rubyEql("a", "a") || rubyEql("a", "b") {
		t.Fatal("string")
	}
	if !rubyEql(bigOf("5"), 5) {
		t.Fatal("big int eql")
	}
}

func TestRubyIdentical(t *testing.T) {
	if !rubyIdentical(5, 5) || !rubyIdentical(Symbol("a"), Symbol("a")) {
		t.Fatal("immutable")
	}
	if rubyIdentical("a", "a") || rubyIdentical([]any{1}, []any{1}) {
		t.Fatal("mutable distinct")
	}
	if rubyIdentical(hashOf(), hashOf()) || rubyIdentical(&Regexp{Source: "x"}, &Regexp{Source: "x"}) {
		t.Fatal("mutable hash/regexp")
	}
	if rubyIdentical(&Range{Begin: 1, End: 2}, &Range{Begin: 1, End: 2}) {
		t.Fatal("mutable range")
	}
	// object identity mismatch of arity: *Object vs non-object.
	if rubyIdentical(&Object{ID: 1}, 5) {
		t.Fatal("obj vs non-obj")
	}
	// two pointer-equal *Objects with ID 0.
	o := &Object{Class: "F"}
	if !rubyIdentical(o, o) {
		t.Fatal("same pointer")
	}
	if rubyIdentical(&Object{Class: "F"}, &Object{Class: "F"}) {
		t.Fatal("distinct zero-id objects")
	}
}

func TestNumericHelpers(t *testing.T) {
	for _, v := range []any{int8(1), int16(1), int32(1), int64(1), uint(1), uint64(1), bigOf("1")} {
		if _, ok := asBig(v); !ok {
			t.Errorf("asBig %T", v)
		}
	}
	if _, ok := asBig("x"); ok {
		t.Fatal("asBig non-int")
	}
	if toFloat(float32(1.5)) != 1.5 || toFloat(2.5) != 2.5 || toFloat("x") != 0 {
		t.Fatal("toFloat")
	}
	if _, ok := asBigFloat("x"); ok {
		t.Fatal("asBigFloat non-num")
	}
	if _, ok := asBigFloat(1.5); !ok {
		t.Fatal("asBigFloat float")
	}
}

func TestRubyCompare(t *testing.T) {
	if c, ok := rubyCompare(1, 2); !ok || c != -1 {
		t.Fatal("num <")
	}
	if c, ok := rubyCompare(2, 2); !ok || c != 0 {
		t.Fatal("num ==")
	}
	if c, ok := rubyCompare(3, 2); !ok || c != 1 {
		t.Fatal("num >")
	}
	if c, ok := rubyCompare("a", "b"); !ok || c != -1 {
		t.Fatal("str <")
	}
	if c, ok := rubyCompare("b", "a"); !ok || c != 1 {
		t.Fatal("str >")
	}
	if c, ok := rubyCompare("a", "a"); !ok || c != 0 {
		t.Fatal("str ==")
	}
	if _, ok := rubyCompare(1, "x"); ok {
		t.Fatal("num vs str")
	}
	if _, ok := rubyCompare("x", 1); ok {
		t.Fatal("str vs num")
	}
	if _, ok := rubyCompare(nil, nil); ok {
		t.Fatal("incomparable")
	}
}

func TestClassNameAndAncestors(t *testing.T) {
	cases := []struct {
		v any
		w string
	}{
		{nil, "NilClass"}, {true, "TrueClass"}, {false, "FalseClass"},
		{"s", "String"}, {Symbol("a"), "Symbol"}, {1, "Integer"}, {1.5, "Float"},
		{[]any{}, "Array"}, {NewHash(), "Hash"}, {&Range{}, "Range"},
		{&Regexp{}, "Regexp"}, {Class("C"), "Class"}, {Module("M"), "Module"},
		{&Object{Class: "Foo"}, "Foo"}, {uintptr(1), "Object"},
	}
	for _, c := range cases {
		if g := className(c.v); g != c.w {
			t.Errorf("className(%v): got %q want %q", c.v, g, c.w)
		}
	}
	// ancestors for each kind exercises the whole switch.
	for _, v := range []any{nil, true, false, "s", Symbol("a"), 1, 1.5, []any{}, NewHash(), &Range{}, &Regexp{}, Class("C"), Module("M"), &Object{Class: "F"}, uintptr(1)} {
		if len(ancestors(v)) == 0 {
			t.Errorf("ancestors(%v) empty", v)
		}
	}
	if !isKindOf(1, "Numeric") || isKindOf(1, "String") {
		t.Fatal("isKindOf")
	}
}
