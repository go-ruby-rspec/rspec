// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"math"
	"math/big"
	"testing"
)

func TestInspectScalars(t *testing.T) {
	cases := []struct {
		v any
		w string
	}{
		{nil, "nil"},
		{true, "true"},
		{false, "false"},
		{"hi", `"hi"`},
		{Symbol("sym"), ":sym"},
		{42, "42"},
		{int8(1), "1"},
		{int16(2), "2"},
		{int32(3), "3"},
		{int64(-7), "-7"},
		{uint(8), "8"},
		{uint64(9), "9"},
		{bigOf("123456789012345678901234567890"), "123456789012345678901234567890"},
		{float32(1.5), "1.5"},
		{1.0, "1.0"},
		{1.5, "1.5"},
		{100.0, "100.0"},
		{1e20, "1.0e+20"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
		{math.NaN(), "NaN"},
		{Class("Foo"), "Foo"},
		{Module("Bar"), "Bar"},
		{&Regexp{Source: "a.c", Flags: "i"}, "/a.c/i"},
		{&Range{Begin: 1, End: 5}, "1..5"},
		{&Range{Begin: 1, End: 5, Exclusive: true}, "1...5"},
		{[]any{1, "x", Symbol("y")}, `[1, "x", :y]`},
	}
	for _, c := range cases {
		if g := Inspect(c.v); g != c.w {
			t.Errorf("Inspect(%v): got %q want %q", c.v, g, c.w)
		}
	}
}

func TestInspectStringEscapes(t *testing.T) {
	in := "a\nb\tc\r\x00\\\"'\a\b\f\v\x1b\x02é"
	want := `"a\nb\tc\r\0\\\"'\a\b\f\v\e\x02é"`
	if g := Inspect(in); g != want {
		t.Errorf("got %q want %q", g, want)
	}
}

func TestInspectHash(t *testing.T) {
	if g := Inspect(NewHash()); g != "{}" {
		t.Errorf("empty %q", g)
	}
	h := NewHash()
	h.Set(Symbol("a"), 1)
	h.Set("b", 2)
	if g := Inspect(h); g != `{a: 1, "b" => 2}` {
		t.Errorf("mixed %q", g)
	}
	// map[string]any sorts keys.
	m := map[string]any{"z": 1, "a": 2}
	if g := Inspect(m); g != `{"a" => 2, "z" => 1}` {
		t.Errorf("gomap %q", g)
	}
}

func TestInspectObject(t *testing.T) {
	if g := Inspect(&Object{Class: "Foo", ID: 0x10}); g != "#<Foo:0x0000000000000010>" {
		t.Errorf("no-ivar %q", g)
	}
	o := &Object{Class: "Foo", IVars: map[string]any{"@a": 1, "@b": 2}, Order: []string{"@b", "@a"}}
	if g := Inspect(o); g != "#<Foo:0x0000000000000000 @b=2, @a=1>" {
		t.Errorf("ordered %q", g)
	}
	// Unordered falls back to sorted keys.
	o2 := &Object{Class: "Foo", IVars: map[string]any{"@z": 1, "@a": 2}}
	if g := Inspect(o2); g != "#<Foo:0x0000000000000000 @a=2, @z=1>" {
		t.Errorf("sorted %q", g)
	}
}

func TestInspectStringerAndDefault(t *testing.T) {
	// fmt.Stringer path.
	if g := Inspect(stringerT{}); g != "STR" {
		t.Errorf("stringer %q", g)
	}
	// default path for an unmodelled type.
	if g := Inspect(struct{ X int }{5}); g == "" {
		t.Errorf("default %q", g)
	}
}

type stringerT struct{}

func (stringerT) String() string { return "STR" }

func TestHashOps(t *testing.T) {
	h := NewHash()
	h.Set("a", 1)
	h.Set("a", 2) // update
	if v, ok := h.Get("a"); !ok || v != 2 {
		t.Fatal("update")
	}
	if _, ok := h.Get("missing"); ok {
		t.Fatal("missing")
	}
	if h.Len() != 1 {
		t.Fatal("len")
	}
	if len(h.Pairs()) != 1 {
		t.Fatal("pairs")
	}
	// Set on a zero-value Hash (nil index).
	var z Hash
	z.Set("x", 1)
	if v, _ := z.Get("x"); v != 1 {
		t.Fatal("zero hash set")
	}
}

func TestInspectFloatNegZero(t *testing.T) {
	if g := inspectFloat(math.Copysign(0, -1)); g != "-0.0" {
		t.Errorf("negzero %q", g)
	}
	if g := inspectFloat(2.5e-10); g != "2.5e-10" {
		t.Errorf("small exp %q", g)
	}
	// big.Int inspect via Inspect already tested; direct number.
	if Inspect(big.NewInt(5)) != "5" {
		t.Fatal("bignum small")
	}
}
