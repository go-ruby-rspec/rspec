// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"fmt"
	"math/big"
)

// The block-form matchers observe the execution of an example block, which the
// host runs. `expect { … }.to change { … }` and `expect { … }.to raise_error`
// pass their *observations* — the before/after value of the change probe, or the
// error the block raised — as the "actual" this package evaluates. The eval of
// the block and the probe is the rbgo host seam.

// Change is the observation `change { probe }` collected around a block: the
// probe value before and after the host ran the example block. exprName is the
// source of the probe (e.g. "a[0]") used in the message.
type Change struct {
	ExprName string
	Before   any
	After    any
}

// changeMatcher — change { probe }[.from(x)][.to(y)][.by(n)][.by_at_least(n)]…
type changeMatcher struct {
	obs     Change
	mode    changeMode
	from    any
	to      any
	by      any
	hasFrom bool
}

type changeMode int

const (
	changeAny       changeMode = iota // just "have changed"
	changeTo                          // .to(y)
	changeBy                          // .by(n)
	changeByAtLeast                   // .by_at_least(n)
	changeByAtMost                    // .by_at_most(n)
)

// ChangeObserved builds a change matcher over a host-collected observation.
// Chain From/To/By/ByAtLeast/ByAtMost to constrain it.
func ChangeObserved(obs Change) *changeMatcher { return &changeMatcher{obs: obs} }

func (m *changeMatcher) From(x any) *changeMatcher { m.from = x; m.hasFrom = true; return m }
func (m *changeMatcher) To(y any) *changeMatcher   { m.to = y; m.mode = changeTo; return m }
func (m *changeMatcher) By(n any) *changeMatcher   { m.by = n; m.mode = changeBy; return m }
func (m *changeMatcher) ByAtLeast(n any) *changeMatcher {
	m.by = n
	m.mode = changeByAtLeast
	return m
}
func (m *changeMatcher) ByAtMost(n any) *changeMatcher {
	m.by = n
	m.mode = changeByAtMost
	return m
}

func (m *changeMatcher) isBlockMatcher() {}

// Matches ignores its argument (the observation is already captured) and
// evaluates the change constraints.
func (m *changeMatcher) Matches(any) bool {
	changed := !rubyEqual(m.obs.Before, m.obs.After)
	if m.hasFrom && !rubyEqual(m.obs.Before, m.from) {
		return false
	}
	switch m.mode {
	case changeAny:
		return changed
	case changeTo:
		return changed && rubyEqual(m.obs.After, m.to)
	case changeBy:
		d, ok := numericDelta(m.obs.Before, m.obs.After)
		return ok && rubyEqual(d, m.by)
	case changeByAtLeast:
		d, ok := numericDelta(m.obs.Before, m.obs.After)
		return ok && cmpNum(d, m.by) >= 0
	case changeByAtMost:
		d, ok := numericDelta(m.obs.Before, m.obs.After)
		return ok && cmpNum(d, m.by) <= 0
	}
	return false
}

func (m *changeMatcher) FailureMessage() string {
	name := m.obs.ExprName
	switch m.mode {
	case changeTo:
		if !rubyEqual(m.obs.Before, m.obs.After) {
			return fmt.Sprintf("expected `%s` to have changed to %s, but is now %s",
				name, Inspect(m.to), Inspect(m.obs.After))
		}
		return fmt.Sprintf("expected `%s` to have changed to %s, but is still %s",
			name, Inspect(m.to), Inspect(m.obs.After))
	case changeBy:
		d, _ := numericDelta(m.obs.Before, m.obs.After)
		return fmt.Sprintf("expected `%s` to have changed by %s, but was changed by %s",
			name, Inspect(m.by), Inspect(d))
	case changeByAtLeast:
		d, _ := numericDelta(m.obs.Before, m.obs.After)
		return fmt.Sprintf("expected `%s` to have changed by at least %s, but was changed by %s",
			name, Inspect(m.by), Inspect(d))
	case changeByAtMost:
		d, _ := numericDelta(m.obs.Before, m.obs.After)
		return fmt.Sprintf("expected `%s` to have changed by at most %s, but was changed by %s",
			name, Inspect(m.by), Inspect(d))
	default:
		return fmt.Sprintf("expected `%s` to have changed, but is still %s",
			name, Inspect(m.obs.After))
	}
}

func (m *changeMatcher) FailureMessageNegated() string {
	return fmt.Sprintf("expected `%s` not to have changed, but did change from %s to %s",
		m.obs.ExprName, Inspect(m.obs.Before), Inspect(m.obs.After))
}

// numericDelta returns after-before for numeric probes.
func numericDelta(before, after any) (any, bool) {
	b, bok := asBig(before)
	a, aok := asBig(after)
	if bok && aok {
		return normBig(new(big.Int).Sub(a, b)), true
	}
	if isRubyNumber(before) && isRubyNumber(after) {
		return toFloat(after) - toFloat(before), true
	}
	return nil, false
}

func isRubyNumber(v any) bool { return isRubyInteger(v) || isRubyFloat(v) }

// normBig returns an int for small big.Ints so Inspect renders "3" not via big.
func normBig(x *big.Int) any {
	if x.IsInt64() {
		return int(x.Int64())
	}
	return x
}

func cmpNum(a, b any) int {
	af, _ := asBigFloat(a)
	bf, _ := asBigFloat(b)
	if af == nil || bf == nil {
		return 0
	}
	return af.Cmp(bf)
}

// RaisedError is the host's observation of what an example block raised: the
// Ruby class name of the exception and its message, or Raised=false when the
// block completed normally.
type RaisedError struct {
	Raised  bool
	Class   string
	Message string
}

// raiseErrorMatcher — raise_error[(Class[, message|/re/])].
type raiseErrorMatcher struct {
	obs         RaisedError
	wantClass   string // "" = any StandardError
	wantMessage any    // string, *Regexp, or nil
}

// RaiseErrorObserved builds a raise_error matcher over a host observation.
// class "" matches any error; message may be a string, *Regexp, or nil.
func RaiseErrorObserved(obs RaisedError, class string, message any) Matcher {
	return &raiseErrorMatcher{obs: obs, wantClass: class, wantMessage: message}
}

func (m *raiseErrorMatcher) isBlockMatcher() {}

func (m *raiseErrorMatcher) Matches(any) bool {
	if !m.obs.Raised {
		return false
	}
	if m.wantClass != "" && m.obs.Class != m.wantClass {
		return false
	}
	switch msg := m.wantMessage.(type) {
	case nil:
		return true
	case string:
		return m.obs.Message == msg
	case *Regexp:
		re, err := regexpCompile(msg.Source)
		if err != nil {
			return false
		}
		return re.MatchString(m.obs.Message)
	}
	return false
}

// expectedDescription renders the "expected X" clause (class + optional message
// constraint) shared by the failure messages.
func (m *raiseErrorMatcher) expectedDescription() string {
	cls := m.wantClass
	if cls == "" {
		cls = "Exception"
	}
	switch msg := m.wantMessage.(type) {
	case string:
		return fmt.Sprintf("%s with %s", cls, Inspect(msg))
	case *Regexp:
		return fmt.Sprintf("%s with message matching /%s/", cls, msg.Source)
	}
	return cls
}

func (m *raiseErrorMatcher) FailureMessage() string {
	if !m.obs.Raised {
		if m.wantClass == "" && m.wantMessage == nil {
			return "expected Exception but nothing was raised"
		}
		return fmt.Sprintf("expected %s but nothing was raised", m.expectedDescription())
	}
	return fmt.Sprintf("expected %s, got #<%s: %s>",
		m.expectedDescription(), m.obs.Class, m.obs.Message)
}

func (m *raiseErrorMatcher) FailureMessageNegated() string {
	cls := m.wantClass
	if cls == "" {
		cls = "Exception"
	}
	return fmt.Sprintf("expected no %s, got #<%s: %s>", cls, m.obs.Class, m.obs.Message)
}
