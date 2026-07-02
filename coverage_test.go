// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"math"
	"testing"
)

// TestCoverageCorners exercises the remaining error/edge branches so the whole
// deterministic core is covered without a Ruby runtime.
func TestCoverageCorners(t *testing.T) {
	// sliceAny nil branch -> "[]".
	if Inspect(sliceAny(nil)) != "[]" {
		t.Fatal("sliceAny nil")
	}
	if len(sliceAny([]any{1})) != 1 {
		t.Fatal("sliceAny non-nil")
	}

	// isImmutableForEqual default (non-immutable) branch.
	if isImmutableForEqual("string") {
		t.Fatal("string not immutable")
	}
	if !isImmutableForEqual(nil) || !isImmutableForEqual(true) || !isImmutableForEqual(Symbol("a")) || !isImmutableForEqual(1) {
		t.Fatal("immutable set")
	}

	// andList empty and single.
	if andList(nil) != "" {
		t.Fatal("andList empty")
	}
	if andList([]any{1}) != "1" {
		t.Fatal("andList single")
	}

	// indentLines with an internal blank line.
	if indentLines("a\n\nb", "  ") != "  a\n\n  b" {
		t.Fatalf("indentLines blank: %q", indentLines("a\n\nb", "  "))
	}

	// be != Matches path (op "!=").
	if !BeNotEqualOp(5).Matches(6) || BeNotEqualOp(5).Matches(5) {
		t.Fatal("be !=")
	}

	// rangeCovers: value == begin (lo==0), and hi exactly at end inclusive.
	if !rangeCovers(&Range{Begin: 1, End: 3}, 1) {
		t.Fatal("cover begin")
	}

	// inspectFloat exponent mantissa already has a dot (e.g. 1.5e-10).
	if inspectFloat(1.5e-10) != "1.5e-10" {
		t.Fatalf("exp with dot: %q", inspectFloat(1.5e-10))
	}
	// inspectFloat plain integral value gets ".0".
	if inspectFloat(7) != "7.0" {
		t.Fatal("integral")
	}
	_ = math.Inf

	// expectedDescription default (class only, no message) is reached via a
	// non-raising observation with a class and nil message.
	m := RaiseErrorObserved(RaisedError{Raised: true, Class: "E", Message: "m"}, "E", nil).(*raiseErrorMatcher)
	if m.expectedDescription() != "E" {
		t.Fatal("expectedDescription default")
	}

	// The block matchers satisfy the blockMatcher interface; call the marker.
	var cbm blockMatcher = ChangeObserved(Change{})
	cbm.isBlockMatcher()
	var rbm blockMatcher = RaiseErrorObserved(RaisedError{}, "", nil).(*raiseErrorMatcher)
	rbm.isBlockMatcher()

	// be Matches: the <= true branch and the final default (unknown op) return.
	if !BeLessOrEqual(5).Matches(3) {
		t.Fatal("be <= 3")
	}
	if (&beComparisonMatcher{op: "??", operand: 1}).Matches(1) {
		t.Fatal("be unknown op")
	}

	// rangeCovers: value below begin (lo<0) short-circuits.
	if rangeCovers(&Range{Begin: 5, End: 9}, 1) {
		t.Fatal("below begin")
	}

	// andList exactly two items.
	if andList([]any{1, 2}) != "1 and 2" {
		t.Fatal("andList two")
	}

	// rubyEqual through *Object identity branch.
	o := &Object{Class: "F", ID: 3}
	if !rubyEqual(o, o) {
		t.Fatal("object equal self")
	}

	// changeMatcher unknown mode -> final `return false`.
	if (&changeMatcher{mode: changeMode(99)}).Matches(nil) {
		t.Fatal("change unknown mode")
	}

	// rangeCovers: value comparable to begin but not to end (mixed-type end).
	if rangeCovers(&Range{Begin: 1, End: "z"}, 5) {
		t.Fatal("end incomparable")
	}

	// expectedDescription with empty class + string and + regexp messages.
	es := RaiseErrorObserved(RaisedError{Raised: true, Class: "E", Message: "m"}, "", "msg").(*raiseErrorMatcher)
	if es.expectedDescription() != `Exception with "msg"` {
		t.Fatalf("exc string: %q", es.expectedDescription())
	}
	er := RaiseErrorObserved(RaisedError{Raised: true, Class: "E", Message: "m"}, "", &Regexp{Source: "p"}).(*raiseErrorMatcher)
	if er.expectedDescription() != "Exception with message matching /p/" {
		t.Fatalf("exc regex: %q", er.expectedDescription())
	}

	// inspectFloat: negative value and a value whose 'g' form already carries a
	// dotted mantissa in exponent form.
	if inspectFloat(-3.5) != "-3.5" {
		t.Fatal("neg float")
	}
	if inspectFloat(1e-7) != "1.0e-07" {
		t.Fatalf("tiny: %q", inspectFloat(1e-7))
	}
	// positive zero branch.
	if inspectFloat(0.0) != "0.0" {
		t.Fatal("pos zero")
	}
	// exponent form whose mantissa already carries a dot (skip appending .0).
	if inspectFloat(1.5e-10) != "1.5e-10" {
		t.Fatalf("dotted exp: %q", inspectFloat(1.5e-10))
	}
}

// TestSummaryFailuresAndPendingOnly covers the Summary branches where only
// failures (no pending) and only pending (no failures) are present, plus the
// numberOutcomes pending-ordinal path.
func TestSummaryVariants(t *testing.T) {
	// Failures only.
	root := NewRootGroup("G")
	f1 := root.It("f1")
	f1.Location = "s:1"
	f1.Result = Result{Status: Failed, FailureExpression: "x", FailureMessage: "m"}
	f2 := root.It("f2")
	f2.Location = "s:2"
	f2.Result = Result{Status: Failed, FailureExpression: "y", FailureMessage: "n"}
	r := NewReporter()
	r.Record(f1)
	r.Record(f2)
	out := r.Summary(Timing{Text: "T"})
	if !contains(out, "2 examples, 2 failures") || !contains(out, "Failed examples:") {
		t.Errorf("failures-only: %q", out)
	}

	// Pending only, multiple -> pending ordinals and blank-line separators.
	root2 := NewRootGroup("H")
	p1 := root2.It("p1")
	p1.Result = Result{Status: Pending, PendingReason: "r1"}
	p2 := root2.It("p2")
	p2.Result = Result{Status: Pending, PendingReason: "r2"}
	r2 := NewReporter()
	r2.Record(p1)
	r2.Record(p2)
	out2 := r2.Summary(Timing{Text: "T"})
	if !contains(out2, "0 failures, 2 pending") {
		t.Errorf("pending-only: %q", out2)
	}
	if p1.pendingOrdinal != 1 || p2.pendingOrdinal != 2 {
		t.Fatal("pending ordinals")
	}
}

// TestRubyEqualDefaultPath covers the final `return a == b` fallthrough for a
// comparable, unmodelled, non-numeric type.
func TestRubyEqualDefaultPath(t *testing.T) {
	type token struct{ n int }
	if !rubyEqual(token{1}, token{1}) || rubyEqual(token{1}, token{2}) {
		t.Fatal("default equal")
	}
}
