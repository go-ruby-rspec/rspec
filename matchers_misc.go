// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

// regexpCompile compiles a Ruby-source regexp to a Go regexp. Shared by match
// and raise_error's message-matching form.
func regexpCompile(src string) (*regexp.Regexp, error) { return regexp.Compile(src) }

// matchMatcher — match(/re/) or match(pattern). A Regexp argument tests the
// pattern; a string argument tests literal equality-as-pattern (RSpec renders
// it quoted); composability with a nested matcher is out of scope here.
type matchMatcher struct {
	pattern any
	actual  any
}

// Match matches when actual matches the given Regexp (or string pattern).
func Match(pattern any) Matcher { return &matchMatcher{pattern: pattern} }

func (m *matchMatcher) Matches(a any) bool {
	m.actual = a
	s, ok := a.(string)
	if !ok {
		return false
	}
	switch p := m.pattern.(type) {
	case *Regexp:
		re, err := regexp.Compile(p.Source)
		if err != nil {
			return false
		}
		return re.MatchString(s)
	case string:
		re, err := regexp.Compile(p)
		if err != nil {
			return s == p
		}
		return re.MatchString(s)
	}
	return false
}

func (m *matchMatcher) Description() string { return "match " + Inspect(m.pattern) }

func (m *matchMatcher) FailureMessage() string {
	return fmt.Sprintf("expected %s to match %s", Inspect(m.actual), Inspect(m.pattern))
}
func (m *matchMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected %s not to match %s", Inspect(m.actual), Inspect(m.pattern))
}

// haveAttributesMatcher — have_attributes(name: value, …). Keys are rendered
// alphabetically in the message (RSpec sorts them).
type haveAttributesMatcher struct {
	expected []Pair
	actual   any
	had      []Pair // actual values for the expected keys
}

// HaveAttributes matches when actual's attributes equal the expected pairs.
func HaveAttributes(attrs *Hash) Matcher {
	// Sort keys alphabetically by their Symbol/string name for message parity.
	pairs := append([]Pair(nil), attrs.pairs...)
	sort.SliceStable(pairs, func(i, j int) bool {
		return attrKey(pairs[i].Key) < attrKey(pairs[j].Key)
	})
	return &haveAttributesMatcher{expected: pairs}
}

func attrKey(k any) string {
	if s, ok := k.(Symbol); ok {
		return string(s)
	}
	if s, ok := k.(string); ok {
		return s
	}
	return Inspect(k)
}

func (m *haveAttributesMatcher) Matches(a any) bool {
	m.actual = a
	m.had = nil
	ok := true
	for _, p := range m.expected {
		got, present := AttrReader(a, attrKey(p.Key))
		if !present {
			ok = false
			m.had = append(m.had, Pair{p.Key, Symbol("no method")})
			continue
		}
		m.had = append(m.had, Pair{p.Key, got})
		if !rubyEqual(got, p.Val) {
			ok = false
		}
	}
	return ok
}

func (m *haveAttributesMatcher) FailureMessage() string {
	return fmt.Sprintf("expected %s to have attributes %s but had attributes %s",
		Inspect(m.actual), symKeyHash(m.expected), symKeyHash(m.had))
}
func (m *haveAttributesMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected %s not to have attributes %s",
		Inspect(m.actual), symKeyHash(m.expected))
}

// symKeyHash renders a pair list as a `{key: val}` hash literal (symbol keys).
func symKeyHash(pairs []Pair) string {
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = attrKey(p.Key) + ": " + Inspect(p.Val)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// allMatcher — all(matcher): every element must satisfy the inner matcher.
type allMatcher struct {
	inner     Matcher
	actual    []any
	failIndex int
	failMsg   string
}

// All matches when every element of actual satisfies inner.
func All(inner Matcher) Matcher { return &allMatcher{inner: inner, failIndex: -1} }

func (m *allMatcher) Matches(a any) bool {
	m.failIndex = -1
	arr, ok := a.([]any)
	if !ok {
		m.actual = nil
		return false
	}
	m.actual = arr
	for i, e := range arr {
		if !m.inner.Matches(e) {
			m.failIndex = i
			m.failMsg = m.inner.FailureMessage()
			return false
		}
	}
	return true
}

func (m *allMatcher) FailureMessage() string {
	indent := indentLines(m.failMsg, "      ")
	return fmt.Sprintf("expected %s to all %s\n\n   object at index %d failed to match:\n%s",
		Inspect(sliceAny(m.actual)), innerDescription(m.inner), m.failIndex, indent)
}
func (m *allMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected %s not to all %s",
		Inspect(sliceAny(m.actual)), innerDescription(m.inner))
}

// indentLines prefixes every non-empty line of s with pad, preserving each
// line's own internal alignment (used by all's nested message).
func indentLines(s, pad string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		if l == "" {
			continue
		}
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

// respondToMatcher — respond_to(:m).with(n).arguments.
type respondToMatcher struct {
	methods  []Symbol
	argCount int // -1 = unspecified
	actual   any
	missing  []Symbol
}

// RespondTo matches when actual responds to every named method.
func RespondTo(methods ...Symbol) *respondToMatcher {
	return &respondToMatcher{methods: methods, argCount: -1}
}

// With sets the expected argument count (respond_to(:m).with(n).arguments).
func (m *respondToMatcher) With(n int) *respondToMatcher { m.argCount = n; return m }

func (m *respondToMatcher) Matches(a any) bool {
	m.actual = a
	m.missing = nil
	for _, name := range m.methods {
		if !Responder(a, string(name)) {
			m.missing = append(m.missing, name)
		}
	}
	return len(m.missing) == 0
}

func (m *respondToMatcher) names(list []Symbol) string {
	items := make([]any, len(list))
	for i, s := range list {
		items[i] = s
	}
	return andList(items)
}

func (m *respondToMatcher) argSuffix() string {
	if m.argCount < 0 {
		return ""
	}
	unit := "arguments"
	if m.argCount == 1 {
		unit = "argument"
	}
	return fmt.Sprintf(" with %d %s", m.argCount, unit)
}

func (m *respondToMatcher) FailureMessage() string {
	return fmt.Sprintf("expected %s to respond to %s%s",
		Inspect(m.actual), m.names(m.missing), m.argSuffix())
}
func (m *respondToMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected %s not to respond to %s%s",
		Inspect(m.actual), m.names(m.methods), m.argSuffix())
}

// satisfyMatcher — satisfy("description") { |x| … }. The predicate is a Go
// func supplied by the host (the block eval seam); the description defaults to
// RSpec's `expression \`…\“ form when absent.
type satisfyMatcher struct {
	desc   string
	expr   string // source snippet for the default description
	pred   func(any) bool
	actual any
}

// Satisfy matches when pred(actual) is true. desc is the human description used
// in the message (RSpec's optional string argument); when empty, expr provides
// the `satisfy expression \`expr\“ default.
func Satisfy(desc, expr string, pred func(any) bool) Matcher {
	return &satisfyMatcher{desc: desc, expr: expr, pred: pred}
}

func (m *satisfyMatcher) Matches(a any) bool {
	m.actual = a
	return m.pred(a)
}

func (m *satisfyMatcher) description() string {
	if m.desc != "" {
		return m.desc
	}
	return "satisfy expression `" + m.expr + "`"
}
func (m *satisfyMatcher) FailureMessage() string {
	return fmt.Sprintf("expected %s to %s", Inspect(m.actual), m.description())
}
func (m *satisfyMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected %s not to %s", Inspect(m.actual), m.description())
}

// beWithinMatcher — be_within(delta).of(expected).
type beWithinMatcher struct {
	delta    float64
	expected float64
	set      bool
	actual   any
}

// BeWithin builds be_within(delta); call Of to set the centre.
func BeWithin(delta float64) *beWithinMatcher { return &beWithinMatcher{delta: delta} }

// Of sets the expected centre value (be_within(d).of(x)).
func (m *beWithinMatcher) Of(expected float64) *beWithinMatcher {
	m.expected = expected
	m.set = true
	return m
}

func (m *beWithinMatcher) Matches(a any) bool {
	m.actual = a
	f, ok := asBigFloat(a)
	if !ok {
		return false
	}
	af, _ := f.Float64()
	return math.Abs(af-m.expected) <= m.delta
}

func (m *beWithinMatcher) FailureMessage() string {
	return fmt.Sprintf("expected %s to be within %s of %s",
		Inspect(m.actual), inspectFloat(m.delta), Inspect(numFromFloat(m.expected)))
}
func (m *beWithinMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected %s not to be within %s of %s",
		Inspect(m.actual), inspectFloat(m.delta), Inspect(numFromFloat(m.expected)))
}

// coverMatcher — cover(items…) for a Range actual.
type coverMatcher struct {
	values []any
	actual any
}

// Cover matches when a Range actual covers every argument.
func Cover(values ...any) Matcher { return &coverMatcher{values: values} }

func (m *coverMatcher) Matches(a any) bool {
	m.actual = a
	r, ok := a.(*Range)
	if !ok {
		return false
	}
	for _, v := range m.values {
		if !rangeCovers(r, v) {
			return false
		}
	}
	return true
}

func rangeCovers(r *Range, v any) bool {
	lo, ok := rubyCompare(v, r.Begin)
	if !ok || lo < 0 {
		return false
	}
	hi, ok := rubyCompare(v, r.End)
	if !ok {
		return false
	}
	if r.Exclusive {
		return hi < 0
	}
	return hi <= 0
}

func (m *coverMatcher) FailureMessage() string {
	return fmt.Sprintf("expected %s to cover %s", Inspect(m.actual), andList(m.values))
}
func (m *coverMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected %s not to cover %s", Inspect(m.actual), andList(m.values))
}

// predicateMatcher — be_empty → empty?, be_valid → valid?, etc.
type predicateMatcher struct {
	predicate string // bare name, e.g. "empty"
	actual    any
	responds  bool
}

// BePredicate builds a predicate matcher for `predicate?` (predicate given
// without the trailing '?'), e.g. BePredicate("empty") is be_empty.
func BePredicate(predicate string) Matcher {
	return &predicateMatcher{predicate: predicate}
}

func (m *predicateMatcher) Matches(a any) bool {
	m.actual = a
	res, ok := Predicate(a, m.predicate)
	m.responds = ok
	return ok && res
}

func (m *predicateMatcher) FailureMessage() string {
	if !m.responds {
		return fmt.Sprintf("expected %s to respond to `%s?`", Inspect(m.actual), m.predicate)
	}
	return fmt.Sprintf("expected `%s.%s?` to be truthy, got false",
		Inspect(m.actual), m.predicate)
}
func (m *predicateMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected `%s.%s?` to be falsey, got true",
		Inspect(m.actual), m.predicate)
}

// numFromFloat returns an int when f is integral (so Inspect prints "10" not
// "10.0"), matching RSpec's rendering of the integer centre in be_within.
func numFromFloat(f float64) any {
	if f == math.Trunc(f) && !math.IsInf(f, 0) {
		return int64(f)
	}
	return f
}
