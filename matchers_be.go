// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"fmt"
	"strings"
)

// truthy reports Ruby truthiness: everything but nil and false is truthy.
func truthy(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

// article returns "an" before a vowel-initial word, else "a" — RSpec's
// EnglishPhrasing.
func article(word string) string {
	if word == "" {
		return "a"
	}
	switch strings.ToLower(word[:1]) {
	case "a", "e", "i", "o", "u":
		return "an"
	}
	return "a"
}

// beTruthyMatcher — be_truthy.
type beTruthyMatcher struct{ actual any }

// BeTruthy matches any truthy value.
func BeTruthy() Matcher { return &beTruthyMatcher{} }

func (m *beTruthyMatcher) Matches(a any) bool { m.actual = a; return truthy(a) }
func (m *beTruthyMatcher) FailureMessage() string {
	return "expected: truthy value\n     got: " + Inspect(m.actual)
}
func (m *beTruthyMatcher) FailureMessageNegated() string {
	return "expected: falsey value\n     got: " + Inspect(m.actual)
}

// beFalseyMatcher — be_falsey.
type beFalseyMatcher struct{ actual any }

// BeFalsey matches nil or false.
func BeFalsey() Matcher { return &beFalseyMatcher{} }

func (m *beFalseyMatcher) Matches(a any) bool { m.actual = a; return !truthy(a) }
func (m *beFalseyMatcher) FailureMessage() string {
	return "expected: falsey value\n     got: " + Inspect(m.actual)
}
func (m *beFalseyMatcher) FailureMessageNegated() string {
	return "expected: truthy value\n     got: " + Inspect(m.actual)
}

// beNilMatcher — be_nil.
type beNilMatcher struct{ actual any }

// BeNil matches nil.
func BeNil() Matcher { return &beNilMatcher{} }

func (m *beNilMatcher) Matches(a any) bool { m.actual = a; return a == nil }
func (m *beNilMatcher) FailureMessage() string {
	return "expected: nil\n     got: " + Inspect(m.actual)
}
func (m *beNilMatcher) FailureMessageNegated() string {
	return "expected: not nil\n     got: " + Inspect(m.actual)
}

// beComparisonMatcher — be > x, be >= x, be < x, be <= x, be == x, be != x.
type beComparisonMatcher struct {
	op      string
	operand any
	actual  any
}

// Be returns a comparison builder; combine with a comparison operator via the
// Be* constructors below. Bare `be` (no operator) asserts truthiness like
// be_truthy but with its own message; RSpec's bare `be` is rarely used, so the
// operator forms are the common path.
func BeGreaterThan(x any) Matcher    { return &beComparisonMatcher{op: ">", operand: x} }
func BeGreaterOrEqual(x any) Matcher { return &beComparisonMatcher{op: ">=", operand: x} }
func BeLessThan(x any) Matcher       { return &beComparisonMatcher{op: "<", operand: x} }
func BeLessOrEqual(x any) Matcher    { return &beComparisonMatcher{op: "<=", operand: x} }
func BeEqualOp(x any) Matcher        { return &beComparisonMatcher{op: "==", operand: x} }
func BeNotEqualOp(x any) Matcher     { return &beComparisonMatcher{op: "!=", operand: x} }

func (m *beComparisonMatcher) Matches(a any) bool {
	m.actual = a
	if m.op == "==" {
		return rubyEqual(a, m.operand)
	}
	if m.op == "!=" {
		return !rubyEqual(a, m.operand)
	}
	c, ok := rubyCompare(a, m.operand)
	if !ok {
		return false
	}
	switch m.op {
	case ">":
		return c > 0
	case ">=":
		return c >= 0
	case "<":
		return c < 0
	case "<=":
		return c <= 0
	}
	return false
}

func (m *beComparisonMatcher) Description() string {
	return "be " + m.op + " " + Inspect(m.operand)
}

func (m *beComparisonMatcher) FailureMessage() string {
	// RSpec aligns "got" under "expected" by padding to the operator width.
	exp := m.op + " " + Inspect(m.operand)
	pad := strings.Repeat(" ", len(m.op)+1)
	return fmt.Sprintf("expected: %s\n     got: %s%s", exp, pad, Inspect(m.actual))
}

func (m *beComparisonMatcher) FailureMessageNegated() string {
	exp := m.op + " " + Inspect(m.operand)
	pad := strings.Repeat(" ", len(m.op)+1)
	return fmt.Sprintf("expected: not %s\n     got:     %s%s", exp, pad, Inspect(m.actual))
}

// beKindOfMatcher — be_a / be_kind_of / be_an.
type beKindOfMatcher struct {
	class  string
	actual any
}

// BeKindOf matches when actual is a kind of the named class (be_a / be_kind_of).
func BeKindOf(class string) Matcher { return &beKindOfMatcher{class: class} }

func (m *beKindOfMatcher) Matches(a any) bool { m.actual = a; return isKindOf(a, m.class) }
func (m *beKindOfMatcher) Description() string {
	return "be " + article(m.class) + " kind of " + m.class
}
func (m *beKindOfMatcher) FailureMessage() string {
	return fmt.Sprintf("expected %s to be %s kind of %s",
		Inspect(m.actual), article(m.class), m.class)
}
func (m *beKindOfMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected %s not to be %s kind of %s",
		Inspect(m.actual), article(m.class), m.class)
}

// beInstanceOfMatcher — be_instance_of / be_an_instance_of.
type beInstanceOfMatcher struct {
	class  string
	actual any
}

// BeInstanceOf matches when actual's exact class is the named class.
func BeInstanceOf(class string) Matcher { return &beInstanceOfMatcher{class: class} }

func (m *beInstanceOfMatcher) Matches(a any) bool {
	m.actual = a
	return className(a) == m.class
}
func (m *beInstanceOfMatcher) FailureMessage() string {
	return fmt.Sprintf("expected %s to be an instance of %s",
		Inspect(m.actual), m.class)
}
func (m *beInstanceOfMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected %s not to be an instance of %s",
		Inspect(m.actual), m.class)
}
