// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"fmt"
	"strings"
)

// includeMatcher — include(items...). For arrays it tests membership; for
// hashes each item is a key (or a single-pair hash key=>val). For strings it
// tests substring.
type includeMatcher struct {
	expected []any
	actual   any
	missing  []any
}

// Include matches when actual includes every argument.
func Include(items ...any) Matcher { return &includeMatcher{expected: items} }

func (m *includeMatcher) Matches(a any) bool {
	m.actual = a
	m.missing = nil
	for _, want := range m.expected {
		if !includesOne(a, want) {
			m.missing = append(m.missing, want)
		}
	}
	return len(m.missing) == 0
}

func includesOne(a, want any) bool {
	switch coll := a.(type) {
	case []any:
		for _, e := range coll {
			if rubyEqual(e, want) {
				return true
			}
		}
		return false
	case string:
		if s, ok := want.(string); ok {
			return strings.Contains(coll, s)
		}
		return false
	case *Hash:
		if h, ok := want.(*Hash); ok {
			// include(k => v): every pair must be present with equal value.
			for _, p := range h.pairs {
				bv, found := coll.Get(p.Key)
				if !found || !rubyEqual(bv, p.Val) {
					return false
				}
			}
			return true
		}
		_, found := coll.Get(want)
		return found
	}
	return false
}

func (m *includeMatcher) FailureMessage() string {
	return fmt.Sprintf("expected %s to include %s", Inspect(m.actual), andList(m.missing))
}
func (m *includeMatcher) FailureMessageNegated() string {
	// When negated, RSpec lists the items that WERE included.
	return fmt.Sprintf("expected %s not to include %s", Inspect(m.actual), andList(m.expected))
}

// startEndMatcher — start_with / end_with.
type startEndMatcher struct {
	start    bool
	expected []any
	actual   any
}

// StartWith matches when actual starts with the given prefix elements/substring.
func StartWith(items ...any) Matcher { return &startEndMatcher{start: true, expected: items} }

// EndWith matches when actual ends with the given suffix elements/substring.
func EndWith(items ...any) Matcher { return &startEndMatcher{start: false, expected: items} }

func (m *startEndMatcher) Matches(a any) bool {
	m.actual = a
	switch coll := a.(type) {
	case string:
		if len(m.expected) != 1 {
			return false
		}
		s, ok := m.expected[0].(string)
		if !ok {
			return false
		}
		if m.start {
			return strings.HasPrefix(coll, s)
		}
		return strings.HasSuffix(coll, s)
	case []any:
		if len(m.expected) > len(coll) {
			return false
		}
		if m.start {
			for i, want := range m.expected {
				if !rubyEqual(coll[i], want) {
					return false
				}
			}
		} else {
			off := len(coll) - len(m.expected)
			for i, want := range m.expected {
				if !rubyEqual(coll[off+i], want) {
					return false
				}
			}
		}
		return true
	}
	return false
}

func (m *startEndMatcher) verb() string {
	if m.start {
		return "start with"
	}
	return "end with"
}
func (m *startEndMatcher) FailureMessage() string {
	return fmt.Sprintf("expected %s to %s %s", Inspect(m.actual), m.verb(), andList(m.expected))
}
func (m *startEndMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected %s not to %s %s", Inspect(m.actual), m.verb(), andList(m.expected))
}

// containExactlyMatcher — contain_exactly / match_array (order-independent
// multiset equality with missing/extra diagnostics).
type containExactlyMatcher struct {
	expected []any
	actual   []any
	missing  []any
	extra    []any
	arrayArg bool // match_array takes a single array; contain_exactly takes varargs
}

// ContainExactly matches when actual holds exactly the given elements, any order.
func ContainExactly(items ...any) Matcher { return &containExactlyMatcher{expected: items} }

// MatchArray is contain_exactly with a single array argument.
func MatchArray(items []any) Matcher {
	return &containExactlyMatcher{expected: items, arrayArg: true}
}

func (m *containExactlyMatcher) Matches(a any) bool {
	arr, ok := a.([]any)
	if !ok {
		m.actual = nil
		return false
	}
	m.actual = arr
	m.missing, m.extra = multisetDiff(m.expected, arr)
	return len(m.missing) == 0 && len(m.extra) == 0
}

// multisetDiff returns the elements of expected missing from actual and the
// elements of actual not accounted for by expected (multiset semantics).
func multisetDiff(expected, actual []any) (missing, extra []any) {
	used := make([]bool, len(actual))
	for _, want := range expected {
		found := false
		for i, got := range actual {
			if !used[i] && rubyEqual(got, want) {
				used[i] = true
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, want)
		}
	}
	for i, got := range actual {
		if !used[i] {
			extra = append(extra, got)
		}
	}
	return missing, extra
}

func (m *containExactlyMatcher) FailureMessage() string {
	var b strings.Builder
	fmt.Fprintf(&b, "expected collection contained:  %s\n", Inspect(sliceAny(m.expected)))
	fmt.Fprintf(&b, "actual collection contained:    %s\n", Inspect(sliceAny(m.actual)))
	if len(m.missing) > 0 {
		fmt.Fprintf(&b, "the missing elements were:      %s\n", Inspect(sliceAny(m.missing)))
	}
	if len(m.extra) > 0 {
		fmt.Fprintf(&b, "the extra elements were:        %s\n", Inspect(sliceAny(m.extra)))
	}
	return b.String()
}

func (m *containExactlyMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected %s not to contain exactly %s",
		Inspect(sliceAny(m.actual)), andList(m.expected))
}

// sliceAny normalises a nil []any to an empty one so Inspect prints "[]".
func sliceAny(s []any) []any {
	if s == nil {
		return []any{}
	}
	return s
}
