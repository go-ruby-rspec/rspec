// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import "testing"

func TestDefaultAttrReader(t *testing.T) {
	if v, ok := AttrReader("héllo", "size"); !ok || v != 5 {
		t.Errorf("string size %v", v)
	}
	if v, ok := AttrReader("ab", "length"); !ok || v != 2 {
		t.Errorf("length %v", v)
	}
	if v, ok := AttrReader("ab", "bytesize"); !ok || v != 2 {
		t.Errorf("bytesize %v", v)
	}
	if v, ok := AttrReader("aB", "upcase"); !ok || v != "AB" {
		t.Errorf("upcase %v", v)
	}
	if v, ok := AttrReader("aB", "downcase"); !ok || v != "ab" {
		t.Errorf("downcase %v", v)
	}
	if _, ok := AttrReader("x", "nope"); ok {
		t.Fatal("string unknown")
	}
	// arrays
	arr := []any{1, 2, 3}
	if v, _ := AttrReader(arr, "size"); v != 3 {
		t.Fatal("array size")
	}
	if v, _ := AttrReader(arr, "first"); v != 1 {
		t.Fatal("first")
	}
	if v, _ := AttrReader(arr, "last"); v != 3 {
		t.Fatal("last")
	}
	if v, ok := AttrReader([]any{}, "first"); !ok || v != nil {
		t.Fatal("first empty")
	}
	if v, ok := AttrReader([]any{}, "last"); !ok || v != nil {
		t.Fatal("last empty")
	}
	if _, ok := AttrReader(arr, "nope"); ok {
		t.Fatal("array unknown")
	}
	// hash
	if v, _ := AttrReader(hashOf("a", 1), "size"); v != 1 {
		t.Fatal("hash size")
	}
	if _, ok := AttrReader(hashOf(), "nope"); ok {
		t.Fatal("hash unknown")
	}
	// object ivar with @ and without
	o := &Object{Class: "F", IVars: map[string]any{"@name": "n", "age": 5}}
	if v, _ := AttrReader(o, "name"); v != "n" {
		t.Fatal("obj @name")
	}
	if v, _ := AttrReader(o, "age"); v != 5 {
		t.Fatal("obj bare")
	}
	if _, ok := AttrReader(o, "missing"); ok {
		t.Fatal("obj missing")
	}
	// unmodelled actual
	if _, ok := AttrReader(42, "size"); ok {
		t.Fatal("int attr")
	}
}

func TestDefaultResponder(t *testing.T) {
	if !Responder("x", "size") || Responder("x", "nope") {
		t.Fatal("string")
	}
	if !Responder([]any{}, "map") || !Responder(hashOf(), "keys") {
		t.Fatal("array/hash")
	}
	if Responder(42, "x") {
		t.Fatal("int")
	}
	o := &Object{Class: "F", RespondsTo: []string{"foo"}, IVars: map[string]any{"@bar": 1}}
	if !Responder(o, "foo") || !Responder(o, "bar") || Responder(o, "baz") {
		t.Fatal("object")
	}
}

func TestDefaultPredicate(t *testing.T) {
	if r, ok := Predicate("", "empty"); !ok || !r {
		t.Fatal("string empty")
	}
	if r, ok := Predicate("x", "empty"); !ok || r {
		t.Fatal("string non-empty")
	}
	if r, ok := Predicate([]any{}, "empty"); !ok || !r {
		t.Fatal("array empty")
	}
	if r, ok := Predicate(hashOf(), "empty"); !ok || !r {
		t.Fatal("hash empty")
	}
	if _, ok := Predicate("x", "weird"); ok {
		t.Fatal("string unknown pred")
	}
	// object stored predicate
	o := &Object{Class: "F", IVars: map[string]any{"?valid": true}}
	if r, ok := Predicate(o, "valid"); !ok || !r {
		t.Fatal("obj pred stored")
	}
	// object responds to pred? but no stored value
	od := &Object{Class: "F", RespondsTo: []string{"ready?"}}
	if r, ok := Predicate(od, "ready"); !ok || r {
		t.Fatal("obj pred declared")
	}
	if _, ok := Predicate(od, "unknown"); ok {
		t.Fatal("obj pred unknown")
	}
	// unmodelled
	if _, ok := Predicate(42, "x"); ok {
		t.Fatal("int pred")
	}
}

func TestUpDownEdge(t *testing.T) {
	if upcase("aZ9") != "AZ9" || downcase("Az9") != "az9" {
		t.Fatal("case fns")
	}
	if len(builtinMethods(42)) != 0 {
		t.Fatal("builtin none")
	}
}
