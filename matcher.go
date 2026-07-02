// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package rspec is a pure-Go (no cgo) reimplementation of the deterministic,
// interpreter-independent core of Ruby's RSpec — the rspec-expectations
// matchers (their match logic and byte-faithful failure messages) and the
// rspec-core example-group structure model and formatter output.
//
// Running the bodies of `it` examples and before/after/let/subject hooks is the
// host's job (rbgo evaluates the Ruby); this library provides the parts that are
// pure functions of values and results: does a matcher match, what message does
// it produce, and how does the formatter render a set of results. It is the
// RSpec backend for go-embedded-ruby, but a standalone, reusable module with no
// dependency on a Ruby runtime.
package rspec

// Matcher is the RSpec matcher protocol as this package models it: a predicate
// on the actual value plus the two failure messages RSpec renders for `to` and
// `not_to`. It mirrors rspec-expectations' `matches?` / `failure_message` /
// `failure_message_when_negated`.
type Matcher interface {
	// Matches reports whether actual satisfies the matcher (RSpec `matches?`).
	Matches(actual any) bool
	// FailureMessage is rendered when a positive expectation (`to`) fails.
	FailureMessage() string
	// FailureMessageNegated is rendered when a negative expectation (`not_to`)
	// fails (RSpec `failure_message_when_negated`).
	FailureMessageNegated() string
}

// Describer is the optional protocol for a matcher's `description` (used by
// composed matchers and `all`, and by the documentation formatter's generated
// example names). Matchers that do not implement it fall back to a generic
// description.
type Describer interface {
	Description() string
}

// innerDescription returns m's RSpec description, falling back to a generic form.
func innerDescription(m Matcher) string {
	if d, ok := m.(Describer); ok {
		return d.Description()
	}
	return "match"
}

// blockMatcher is implemented by matchers whose actual is a block whose
// execution (and any state change or raised error it causes) is observed —
// change, raise_error. The host runs the block; these matchers consume the
// observation. A plain Matcher operates on an already-evaluated value.
type blockMatcher interface {
	Matcher
	// MatchesBlock is like Matches but for the block-form matchers; run reports
	// what the host observed when it executed the example block.
	isBlockMatcher()
}

// Expect models `expect(actual).to(matcher)` / `.not_to(matcher)`. It returns
// the failure message (or "" on success) so the host can record a result
// without this package raising. Positive is the `to` direction.
func Expect(actual any, m Matcher, positive bool) (ok bool, message string) {
	matched := m.Matches(actual)
	if positive {
		if matched {
			return true, ""
		}
		return false, m.FailureMessage()
	}
	// not_to
	if !matched {
		return true, ""
	}
	return false, m.FailureMessageNegated()
}
