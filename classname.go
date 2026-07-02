// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import "math/big"

// className returns the Ruby class name of a value in the model, e.g. "Integer",
// "String", "Array". Used by be_a / be_instance_of and predicate reflection.
func className(v any) string {
	switch v.(type) {
	case nil:
		return "NilClass"
	case bool:
		if v.(bool) {
			return "TrueClass"
		}
		return "FalseClass"
	case string:
		return "String"
	case Symbol:
		return "Symbol"
	case int, int8, int16, int32, int64, uint, uint64, *big.Int:
		return "Integer"
	case float32, float64:
		return "Float"
	case []any:
		return "Array"
	case *Hash:
		return "Hash"
	case *Range:
		return "Range"
	case *Regexp:
		return "Regexp"
	case Class:
		return "Class"
	case Module:
		return "Module"
	case *Object:
		return v.(*Object).Class
	}
	return "Object"
}

// ancestors returns the Ruby ancestor chain of a value's class (names only),
// enough for be_a / be_kind_of to test module/superclass membership over the
// model. The host can supply richer chains via *Object; for built-ins we encode
// MRI's hierarchy for the classes matchers commonly test.
func ancestors(v any) []string {
	switch v.(type) {
	case nil:
		return []string{"NilClass", "Object", "Kernel", "BasicObject"}
	case bool:
		if v.(bool) {
			return []string{"TrueClass", "Object", "Kernel", "BasicObject"}
		}
		return []string{"FalseClass", "Object", "Kernel", "BasicObject"}
	case string:
		return []string{"String", "Comparable", "Object", "Kernel", "BasicObject"}
	case Symbol:
		return []string{"Symbol", "Comparable", "Object", "Kernel", "BasicObject"}
	case int, int8, int16, int32, int64, uint, uint64, *big.Int:
		return []string{"Integer", "Numeric", "Comparable", "Object", "Kernel", "BasicObject"}
	case float32, float64:
		return []string{"Float", "Numeric", "Comparable", "Object", "Kernel", "BasicObject"}
	case []any:
		return []string{"Array", "Enumerable", "Object", "Kernel", "BasicObject"}
	case *Hash:
		return []string{"Hash", "Enumerable", "Object", "Kernel", "BasicObject"}
	case *Range:
		return []string{"Range", "Enumerable", "Object", "Kernel", "BasicObject"}
	case *Regexp:
		return []string{"Regexp", "Object", "Kernel", "BasicObject"}
	case Class:
		return []string{"Class", "Module", "Object", "Kernel", "BasicObject"}
	case Module:
		return []string{"Module", "Object", "Kernel", "BasicObject"}
	case *Object:
		o := v.(*Object)
		chain := []string{o.Class}
		return append(chain, "Object", "Kernel", "BasicObject")
	}
	return []string{"Object", "Kernel", "BasicObject"}
}

// isKindOf reports whether v is an instance of class name (walking ancestors).
func isKindOf(v any, name string) bool {
	for _, a := range ancestors(v) {
		if a == name {
			return true
		}
	}
	// *Object may declare extra ancestor modules via RespondsTo-style metadata;
	// its Class already matched above.
	return false
}
