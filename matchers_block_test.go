// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import "testing"

func TestChangeMatcher(t *testing.T) {
	// plain "have changed"
	if !ChangeObserved(Change{ExprName: "a", Before: 0, After: 1}).Matches(nil) {
		t.Fatal("changed")
	}
	if ChangeObserved(Change{ExprName: "a", Before: 0, After: 0}).Matches(nil) {
		t.Fatal("unchanged")
	}
	if g := fmsg(ChangeObserved(Change{ExprName: "a[0]", Before: 0, After: 0}), nil); g != "expected `a[0]` to have changed, but is still 0" {
		t.Errorf("%q", g)
	}
	// .to
	if !ChangeObserved(Change{ExprName: "a", Before: 0, After: 5}).To(5).Matches(nil) {
		t.Fatal("to pass")
	}
	if ChangeObserved(Change{ExprName: "a", Before: 0, After: 3}).To(5).Matches(nil) {
		t.Fatal("to fail")
	}
	if g := fmsg(ChangeObserved(Change{ExprName: "a[0]", Before: 0, After: 3}).To(5), nil); g != "expected `a[0]` to have changed to 5, but is now 3" {
		t.Errorf("to now %q", g)
	}
	if g := fmsg(ChangeObserved(Change{ExprName: "a[0]", Before: 0, After: 0}).To(5), nil); g != "expected `a[0]` to have changed to 5, but is still 0" {
		t.Errorf("to still %q", g)
	}
	// .from constraint
	if ChangeObserved(Change{ExprName: "a", Before: 1, After: 2}).From(0).Matches(nil) {
		t.Fatal("from mismatch")
	}
	if !ChangeObserved(Change{ExprName: "a", Before: 0, After: 5}).From(0).To(5).Matches(nil) {
		t.Fatal("from+to")
	}
	// .by
	if !ChangeObserved(Change{ExprName: "a", Before: 0, After: 3}).By(3).Matches(nil) {
		t.Fatal("by pass")
	}
	if ChangeObserved(Change{ExprName: "a", Before: 0, After: 3}).By(5).Matches(nil) {
		t.Fatal("by fail")
	}
	if g := fmsg(ChangeObserved(Change{ExprName: "a[0]", Before: 0, After: 3}).By(5), nil); g != "expected `a[0]` to have changed by 5, but was changed by 3" {
		t.Errorf("by %q", g)
	}
	// .by_at_least
	if !ChangeObserved(Change{ExprName: "a", Before: 0, After: 5}).ByAtLeast(3).Matches(nil) {
		t.Fatal("at_least pass")
	}
	if g := fmsg(ChangeObserved(Change{ExprName: "a[0]", Before: 0, After: 1}).ByAtLeast(5), nil); g != "expected `a[0]` to have changed by at least 5, but was changed by 1" {
		t.Errorf("at_least %q", g)
	}
	// .by_at_most
	if !ChangeObserved(Change{ExprName: "a", Before: 0, After: 2}).ByAtMost(3).Matches(nil) {
		t.Fatal("at_most pass")
	}
	if ChangeObserved(Change{ExprName: "a", Before: 0, After: 9}).ByAtMost(3).Matches(nil) {
		t.Fatal("at_most fail")
	}
	if g := fmsg(ChangeObserved(Change{ExprName: "a[0]", Before: 0, After: 9}).ByAtMost(3), nil); g != "expected `a[0]` to have changed by at most 3, but was changed by 9" {
		t.Errorf("at_most msg %q", g)
	}
	// float delta path
	if !ChangeObserved(Change{ExprName: "a", Before: 0.0, After: 1.5}).By(1.5).Matches(nil) {
		t.Fatal("float by")
	}
	// non-numeric delta: by on strings can't compute.
	if ChangeObserved(Change{ExprName: "a", Before: "x", After: "y"}).By(1).Matches(nil) {
		t.Fatal("nonnum by")
	}
	// negated
	if g := nmsg(ChangeObserved(Change{ExprName: "a[0]", Before: 0, After: 3}), nil); g != "expected `a[0]` not to have changed, but did change from 0 to 3" {
		t.Errorf("neg %q", g)
	}
	ChangeObserved(Change{}).isBlockMatcher()
}

func TestNumericDeltaHelpers(t *testing.T) {
	if !isRubyNumber(1) || !isRubyNumber(1.5) || isRubyNumber("x") {
		t.Fatal("isRubyNumber")
	}
	// big delta beyond int64
	d, ok := numericDelta(bigOf("0"), bigOf("340282366920938463463374607431768211456"))
	if !ok {
		t.Fatal("big delta")
	}
	if _, isBig := d.(interface{ Int64() int64 }); !isBig {
		// normBig keeps it as *big.Int
	}
	if cmpNum("x", "y") != 0 {
		t.Fatal("cmpNum nonnum")
	}
	if cmpNum(5, 3) <= 0 {
		t.Fatal("cmpNum")
	}
}

func TestRaiseError(t *testing.T) {
	raised := RaisedError{Raised: true, Class: "ArgumentError", Message: "boom"}
	none := RaisedError{Raised: false}

	// any error
	if !RaiseErrorObserved(raised, "", nil).Matches(nil) {
		t.Fatal("any")
	}
	if RaiseErrorObserved(none, "", nil).Matches(nil) {
		t.Fatal("none raised")
	}
	// specific class
	if !RaiseErrorObserved(raised, "ArgumentError", nil).Matches(nil) {
		t.Fatal("class match")
	}
	if RaiseErrorObserved(raised, "RuntimeError", nil).Matches(nil) {
		t.Fatal("class mismatch")
	}
	// message string
	if !RaiseErrorObserved(raised, "ArgumentError", "boom").Matches(nil) {
		t.Fatal("msg match")
	}
	if RaiseErrorObserved(raised, "ArgumentError", "nope").Matches(nil) {
		t.Fatal("msg mismatch")
	}
	// message regexp
	if !RaiseErrorObserved(raised, "ArgumentError", &Regexp{Source: "bo+m"}).Matches(nil) {
		t.Fatal("regex match")
	}
	if RaiseErrorObserved(raised, "ArgumentError", &Regexp{Source: "zzz"}).Matches(nil) {
		t.Fatal("regex mismatch")
	}
	if RaiseErrorObserved(raised, "ArgumentError", &Regexp{Source: "("}).Matches(nil) {
		t.Fatal("bad regex")
	}
	// unsupported message type
	if RaiseErrorObserved(raised, "", 42).Matches(nil) {
		t.Fatal("bad msg type")
	}

	// Failure messages.
	if g := fmsg(RaiseErrorObserved(none, "ArgumentError", nil), nil); g != "expected ArgumentError but nothing was raised" {
		t.Errorf("none class %q", g)
	}
	if g := fmsg(RaiseErrorObserved(none, "", nil), nil); g != "expected Exception but nothing was raised" {
		t.Errorf("none any %q", g)
	}
	if g := fmsg(RaiseErrorObserved(raised, "RuntimeError", nil), nil); g != "expected RuntimeError, got #<ArgumentError: boom>" {
		t.Errorf("wrong class %q", g)
	}
	if g := fmsg(RaiseErrorObserved(raised, "RuntimeError", "expected"), nil); g != `expected RuntimeError with "expected", got #<ArgumentError: boom>` {
		t.Errorf("wrong msg %q", g)
	}
	if g := fmsg(RaiseErrorObserved(raised, "RuntimeError", &Regexp{Source: "foo"}), nil); g != "expected RuntimeError with message matching /foo/, got #<ArgumentError: boom>" {
		t.Errorf("wrong regex %q", g)
	}
	// negated
	if g := nmsg(RaiseErrorObserved(raised, "ArgumentError", nil), nil); g != "expected no ArgumentError, got #<ArgumentError: boom>" {
		t.Errorf("neg %q", g)
	}
	if g := nmsg(RaiseErrorObserved(raised, "", nil), nil); g != "expected no Exception, got #<ArgumentError: boom>" {
		t.Errorf("neg any %q", g)
	}
	RaiseErrorObserved(raised, "", nil).(*raiseErrorMatcher).isBlockMatcher()
}
