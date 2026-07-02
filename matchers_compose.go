// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import "strings"

// andMatcher — matcher.and(other): both must match; on failure it renders the
// first failing sub-matcher's message (RSpec short-circuits AND messages).
type andMatcher struct {
	left, right Matcher
	failed      Matcher
}

// And composes two matchers so both must match (RSpec `matcher.and(other)`).
func And(left, right Matcher) Matcher { return &andMatcher{left: left, right: right} }

func (m *andMatcher) Matches(a any) bool {
	if !m.left.Matches(a) {
		m.failed = m.left
		return false
	}
	if !m.right.Matches(a) {
		m.failed = m.right
		return false
	}
	return true
}

func (m *andMatcher) Description() string {
	return innerDescription(m.left) + " and " + innerDescription(m.right)
}
func (m *andMatcher) FailureMessage() string {
	if m.failed != nil {
		return m.failed.FailureMessage()
	}
	return m.left.FailureMessage()
}
func (m *andMatcher) FailureMessageNegated() string {
	return "expected not to " + m.Description()
}

// orMatcher — matcher.or(other): either must match; on failure RSpec renders
// both sub-messages joined by "...or:".
type orMatcher struct {
	left, right Matcher
}

// Or composes two matchers so either must match (RSpec `matcher.or(other)`).
func Or(left, right Matcher) Matcher { return &orMatcher{left: left, right: right} }

func (m *orMatcher) Matches(a any) bool {
	return m.left.Matches(a) || m.right.Matches(a)
}

func (m *orMatcher) Description() string {
	return innerDescription(m.left) + " or " + innerDescription(m.right)
}

func (m *orMatcher) FailureMessage() string {
	// RSpec indents each sub-message by three spaces and joins with "\n\n...or:\n\n".
	l := indentEachLine(m.left.FailureMessage(), "   ")
	r := indentEachLine(m.right.FailureMessage(), "   ")
	return l + "\n...or:\n" + r
}
func (m *orMatcher) FailureMessageNegated() string {
	return "expected not to " + m.Description()
}

// indentEachLine prefixes every non-empty line with pad (preserving the leading
// and trailing blank lines the equality messages carry).
func indentEachLine(s, pad string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l == "" {
			continue
		}
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}
