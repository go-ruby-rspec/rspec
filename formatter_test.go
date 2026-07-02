// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import "testing"

// demoReporter reconstructs the canonical spec_demo.rb result set used to pin
// the formatter output byte-for-byte against the real `rspec` run.
func demoReporter() *Reporter {
	root := NewRootGroup("Calc")
	adds := root.It("adds")
	adds.Result = Result{Status: Passed}
	fails := root.It("fails sub")
	fails.Location = "./spec_demo.rb:3"
	fails.Result = Result{Status: Failed,
		FailureExpression: `it "fails sub" do expect(5-2).to eq(4) end`,
		FailureMessage:    fmsg(Eq(4), 3)}
	nested := root.Describe("nested")
	muls := nested.It("muls")
	muls.Result = Result{Status: Passed}
	pend := nested.It("is pending")
	pend.Location = "./spec_demo.rb:6"
	pend.Result = Result{Status: Pending, PendingReason: "later",
		FailureExpression: `it "is pending" do pending("later"); expect(1).to eq(2) end`,
		FailureMessage:    fmsg(Eq(2), 1)}
	r := NewReporter()
	r.Record(adds)
	r.Record(fails)
	r.Record(muls)
	r.Record(pend)
	return r
}

const demoTiming = "T seconds (files took T seconds to load)"

func TestProgressRender(t *testing.T) {
	got := demoReporter().Render(Timing{Text: demoTiming})
	want := ".F.*\n\nPending: (Failures listed here are expected and do not affect your suite's status)\n\n  1) Calc nested is pending\n     # later\n     Failure/Error: it \"is pending\" do pending(\"later\"); expect(1).to eq(2) end\n\n       expected: 2\n            got: 1\n\n       (compared using ==)\n     # ./spec_demo.rb:6\n\nFailures:\n\n  1) Calc fails sub\n     Failure/Error: it \"fails sub\" do expect(5-2).to eq(4) end\n\n       expected: 4\n            got: 3\n\n       (compared using ==)\n     # ./spec_demo.rb:3\n\nFinished in T seconds (files took T seconds to load)\n4 examples, 1 failure, 1 pending\n\nFailed examples:\n\nrspec ./spec_demo.rb:3 # Calc fails sub\n"
	if got != want {
		t.Errorf("\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestDocRender(t *testing.T) {
	got := demoReporter().RenderDoc(Timing{Text: demoTiming})
	want := "\nCalc\n  adds\n  fails sub (FAILED - 1)\n  nested\n    muls\n    is pending (PENDING: later)\n\nPending: (Failures listed here are expected and do not affect your suite's status)\n\n  1) Calc nested is pending\n     # later\n     Failure/Error: it \"is pending\" do pending(\"later\"); expect(1).to eq(2) end\n\n       expected: 2\n            got: 1\n\n       (compared using ==)\n     # ./spec_demo.rb:6\n\nFailures:\n\n  1) Calc fails sub\n     Failure/Error: it \"fails sub\" do expect(5-2).to eq(4) end\n\n       expected: 4\n            got: 3\n\n       (compared using ==)\n     # ./spec_demo.rb:3\n\nFinished in T seconds (files took T seconds to load)\n4 examples, 1 failure, 1 pending\n\nFailed examples:\n\nrspec ./spec_demo.rb:3 # Calc fails sub\n"
	if got != want {
		t.Errorf("\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestAllPassRender(t *testing.T) {
	root := NewRootGroup("P")
	a := root.It("a")
	a.Result = Result{Status: Passed}
	b := root.It("b")
	b.Result = Result{Status: Passed}
	r := NewReporter()
	r.Record(a)
	r.Record(b)
	got := r.Render(Timing{Text: demoTiming})
	want := "..\n\nFinished in T seconds (files took T seconds to load)\n2 examples, 0 failures\n"
	if got != want {
		t.Errorf("\n got %q", got)
	}
}

func TestPendingNoReasonAndTiming(t *testing.T) {
	root := NewRootGroup("G")
	e := root.It("pends")
	e.Result = Result{Status: Pending} // no reason, no failure body
	r := NewReporter()
	r.Record(e)
	out := r.RenderDoc(Timing{Run: "0.01 seconds", Load: "0.03 seconds"})
	// doc suffix uses "No reason given"; timing uses the Run/Load phrasing.
	if !contains(out, "pends (PENDING: No reason given)") {
		t.Errorf("doc suffix: %q", out)
	}
	if !contains(out, "# No reason given") {
		t.Errorf("pending reason: %q", out)
	}
	if !contains(out, "Finished in 0.01 seconds (files took 0.03 seconds to load)") {
		t.Errorf("timing: %q", out)
	}
}

func TestTotalsLine(t *testing.T) {
	cases := []struct {
		e, f, p, err int
		w            string
	}{
		{1, 0, 0, 0, "1 example, 0 failures"},
		{2, 0, 0, 0, "2 examples, 0 failures"},
		{4, 1, 1, 0, "4 examples, 1 failure, 1 pending"},
		{4, 2, 0, 0, "4 examples, 2 failures"},
		{1, 1, 0, 0, "1 example, 1 failure"},
		{3, 0, 2, 0, "3 examples, 0 failures, 2 pending"},
		{0, 0, 0, 0, "0 examples, 0 failures"},
		{5, 2, 1, 1, "5 examples, 2 failures, 1 pending, 1 error occurred outside of examples"},
		{5, 2, 0, 2, "5 examples, 2 failures, 2 errors occurred outside of examples"},
	}
	for _, c := range cases {
		if g := TotalsLine(c.e, c.f, c.p, c.err); g != c.w {
			t.Errorf("TotalsLine(%d,%d,%d,%d): got %q want %q", c.e, c.f, c.p, c.err, g, c.w)
		}
	}
}

func TestProgressCharsAndDocSuffix(t *testing.T) {
	if progressChar(Passed) != "." || progressChar(Failed) != "F" || progressChar(Pending) != "*" || progressChar(StatusUnknown) != "." {
		t.Fatal("progress chars")
	}
	e := &Example{Result: Result{Status: Passed}}
	if docSuffix(e) != "" {
		t.Fatal("pass suffix")
	}
}

func TestFailureBodyNoLocation(t *testing.T) {
	root := NewRootGroup("G")
	e := root.It("x")
	e.Result = Result{Status: Failed, FailureExpression: "expect(1).to eq(2)", FailureMessage: fmsg(Eq(2), 1)}
	r := NewReporter()
	r.Record(e)
	out := r.Summary(Timing{Text: demoTiming})
	// No Location set -> no "# path" line, but still numbered failure.
	if !contains(out, "1) G x") {
		t.Errorf("no-loc failure: %q", out)
	}
}

func TestTimingText(t *testing.T) {
	if (Timing{Text: "X"}).String() != "X" {
		t.Fatal("text")
	}
	if (Timing{Run: "a", Load: "b"}).String() != "a (files took b to load)" {
		t.Fatal("run/load")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
