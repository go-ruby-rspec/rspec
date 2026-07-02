// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import "fmt"

// eqMatcher is `eq(expected)` — Ruby `==`.
type eqMatcher struct {
	expected any
	actual   any
}

// Eq matches when actual == expected (Ruby `==`).
func Eq(expected any) Matcher { return &eqMatcher{expected: expected} }

func (m *eqMatcher) Matches(actual any) bool {
	m.actual = actual
	return rubyEqual(actual, m.expected)
}

func (m *eqMatcher) Description() string { return "eq " + Inspect(m.expected) }

func (m *eqMatcher) FailureMessage() string {
	return fmt.Sprintf("\nexpected: %s\n     got: %s\n\n(compared using ==)\n",
		Inspect(m.expected), Inspect(m.actual))
}

func (m *eqMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("\nexpected: value != %s\n     got: %s\n\n(compared using ==)\n",
		Inspect(m.expected), Inspect(m.actual))
}

// eqlMatcher is `eql(expected)` — Ruby `eql?`.
type eqlMatcher struct {
	expected any
	actual   any
}

// Eql matches when actual.eql?(expected) (type-strict equality).
func Eql(expected any) Matcher { return &eqlMatcher{expected: expected} }

func (m *eqlMatcher) Matches(actual any) bool {
	m.actual = actual
	return rubyEql(actual, m.expected)
}

func (m *eqlMatcher) FailureMessage() string {
	return fmt.Sprintf("\nexpected: %s\n     got: %s\n\n(compared using eql?)\n",
		Inspect(m.expected), Inspect(m.actual))
}

func (m *eqlMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("\nexpected: value != %s\n     got: %s\n\n(compared using eql?)\n",
		Inspect(m.expected), Inspect(m.actual))
}

// equalMatcher is `equal(expected)` — Ruby `equal?` (object identity).
type equalMatcher struct {
	expected any
	actual   any
}

// Equal matches when actual.equal?(expected) (same object identity).
func Equal(expected any) Matcher { return &equalMatcher{expected: expected} }

func (m *equalMatcher) Matches(actual any) bool {
	m.actual = actual
	return rubyIdentical(actual, m.expected)
}

func (m *equalMatcher) FailureMessage() string {
	// RSpec renders a short form for immutable values and the object-identity
	// explanation for reference types. We reproduce the reference form, which is
	// what a host observes when comparing two equal-but-distinct objects.
	if isImmutableForEqual(m.expected) && isImmutableForEqual(m.actual) {
		return fmt.Sprintf("\nexpected: %s\n     got: %s\n\n(compared using equal?)\n",
			Inspect(m.expected), Inspect(m.actual))
	}
	return fmt.Sprintf(
		"\nexpected %s => %s\n     got %s => %s\n\n"+
			"Compared using equal?, which compares object identity,\n"+
			"but expected and actual are not the same object. Use\n"+
			"`expect(actual).to eq(expected)` if you don't care about\n"+
			"object identity in this example.\n\n",
		objectIDRef(m.expected), Inspect(m.expected),
		objectIDRef(m.actual), Inspect(m.actual))
}

func (m *equalMatcher) FailureMessageNegated() string {
	return fmt.Sprintf(
		"\nexpected not %s => %s\n     got %s => %s\n\n"+
			"Compared using equal?, which compares object identity.\n\n",
		objectIDRef(m.expected), Inspect(m.expected),
		objectIDRef(m.actual), Inspect(m.actual))
}

func isImmutableForEqual(v any) bool {
	switch v.(type) {
	case nil, bool, Symbol:
		return true
	}
	return isRubyInteger(v) || isRubyFloat(v)
}

// objectIDRef reproduces RSpec's `#<Class:id>` short reference used in the
// equal? failure message. For strings RSpec prints `#<String:NN>` with the
// object_id; the host supplies IDs via *Object, otherwise we render the class.
func objectIDRef(v any) string {
	switch x := v.(type) {
	case string:
		return "#<String>"
	case []any:
		return "#<Array>"
	case *Hash:
		return "#<Hash>"
	case *Object:
		return fmt.Sprintf("#<%s:%d>", x.Class, x.ID)
	}
	return "#<" + className(v) + ">"
}
