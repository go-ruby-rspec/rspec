// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

// The reflection seam. have_attributes, respond_to, predicate matchers, and
// change/satisfy all invoke methods on the actual value. For the built-in model
// (String/Array/Hash) this package answers common queries directly; for host
// objects the host supplies the behaviour via *Object metadata (RespondsTo) or,
// when a real method call is needed, via these overridable hooks so rbgo can
// route the call through the interpreter.

// AttrReader is called by have_attributes to read attribute `name` off actual.
// The default handles the built-in model; a host may replace it to dispatch to
// the interpreter. It returns (value, true) when the attribute exists.
var AttrReader = defaultAttrReader

// Responder is called by respond_to and predicate matchers to test whether
// actual answers to `method`. The default handles the built-in model and
// *Object.RespondsTo; a host may replace it.
var Responder = defaultResponder

// Predicate is called by predicate matchers (be_empty → empty?) to evaluate the
// predicate method on actual, returning (result, ok) where ok is false when the
// object does not respond to the predicate. The default handles the built-in
// model; a host may replace it.
var Predicate = defaultPredicate

func defaultAttrReader(actual any, name string) (any, bool) {
	switch x := actual.(type) {
	case string:
		switch name {
		case "size", "length":
			return len([]rune(x)), true
		case "bytesize":
			return len(x), true
		case "upcase":
			return upcase(x), true
		case "downcase":
			return downcase(x), true
		}
	case []any:
		switch name {
		case "size", "length", "count":
			return len(x), true
		case "first":
			if len(x) > 0 {
				return x[0], true
			}
			return nil, true
		case "last":
			if len(x) > 0 {
				return x[len(x)-1], true
			}
			return nil, true
		}
	case *Hash:
		switch name {
		case "size", "length", "count":
			return x.Len(), true
		}
	case *Object:
		if v, ok := x.IVars["@"+name]; ok {
			return v, true
		}
		if v, ok := x.IVars[name]; ok {
			return v, true
		}
	}
	return nil, false
}

func defaultResponder(actual any, method string) bool {
	if o, ok := actual.(*Object); ok {
		for _, m := range o.RespondsTo {
			if m == method {
				return true
			}
		}
		if _, ok := o.IVars["@"+method]; ok {
			return true
		}
	}
	for _, m := range builtinMethods(actual) {
		if m == method {
			return true
		}
	}
	return false
}

func defaultPredicate(actual any, method string) (bool, bool) {
	// method is the bare predicate name without the trailing '?'.
	switch x := actual.(type) {
	case string:
		switch method {
		case "empty":
			return len(x) == 0, true
		}
	case []any:
		switch method {
		case "empty":
			return len(x) == 0, true
		}
	case *Hash:
		switch method {
		case "empty":
			return x.Len() == 0, true
		}
	case *Object:
		if v, ok := x.IVars["?"+method]; ok {
			return truthy(v), true
		}
		for _, m := range x.RespondsTo {
			if m == method+"?" {
				// Host declared it responds but gave no stored value; treat as
				// unknown so the host hook can be swapped in. Default to false.
				return false, true
			}
		}
	}
	return false, false
}

// builtinMethods lists the method names the built-in value model answers to,
// enough for respond_to over the common cases.
func builtinMethods(actual any) []string {
	switch actual.(type) {
	case string:
		return []string{"size", "length", "bytesize", "upcase", "downcase",
			"empty?", "to_s", "to_str", "chars", "split", "include?", "start_with?", "end_with?"}
	case []any:
		return []string{"size", "length", "count", "first", "last", "empty?",
			"each", "map", "include?", "to_a", "push", "pop"}
	case *Hash:
		return []string{"size", "length", "count", "empty?", "keys", "values",
			"each", "fetch", "include?", "key?", "has_key?"}
	}
	return nil
}

func upcase(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 32
		}
	}
	return string(b)
}
func downcase(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	return string(b)
}
