// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"fmt"
	"strings"
)

// The rspec-core formatters, reproducing RSpec's output byte-faithfully for a
// given set of results. The progress and documentation formatters render the
// live per-example characters/tree; the summary block (pending list, failures
// list, totals line, failed-examples rerun list) is shared. Timing is
// environment-specific, so a formatter renders it from a caller-supplied
// Duration string (the host measures wall-clock); everything else is a pure
// function of the results.

// Reporter accumulates example results and renders the formatter output. It is
// driven by the host, which calls Example for each finished example in run
// order, then Summary once at the end.
type Reporter struct {
	examples []*Example
}

// NewReporter returns an empty reporter.
func NewReporter() *Reporter { return &Reporter{} }

// Record appends a finished example (with its Result set) in run order.
func (r *Reporter) Record(e *Example) { r.examples = append(r.examples, e) }

// progressChar is the per-example character in the progress formatter.
func progressChar(s Status) string {
	switch s {
	case Failed:
		return "F"
	case Pending:
		return "*"
	default:
		return "."
	}
}

// Progress renders the progress-formatter body (the run of `.`/`F`/`*` chars).
func (r *Reporter) Progress() string {
	var b strings.Builder
	for _, e := range r.examples {
		b.WriteString(progressChar(e.Result.Status))
	}
	return b.String()
}

// Documentation renders the documentation-formatter tree: each group heading
// indented by depth, each example labelled with its outcome suffix.
func (r *Reporter) Documentation() string {
	var b strings.Builder
	var lastChain []*ExampleGroup
	for _, e := range r.examples {
		chain := groupChain(e.Group)
		// Print any group headings not shared with the previous example.
		common := commonPrefixLen(lastChain, chain)
		for depth := common; depth < len(chain); depth++ {
			b.WriteString(strings.Repeat("  ", depth) + chain[depth].Description + "\n")
		}
		lastChain = chain
		indent := strings.Repeat("  ", len(chain))
		b.WriteString(indent + e.Description + docSuffix(e) + "\n")
	}
	return b.String()
}

// docSuffix is the documentation label's outcome marker.
func docSuffix(e *Example) string {
	switch e.Result.Status {
	case Failed:
		return " (FAILED - " + itoa(e.failureOrdinal) + ")"
	case Pending:
		reason := e.Result.PendingReason
		if reason == "" {
			reason = "No reason given"
		}
		return " (PENDING: " + reason + ")"
	default:
		return ""
	}
}

func groupChain(g *ExampleGroup) []*ExampleGroup {
	var chain []*ExampleGroup
	for cur := g; cur != nil; cur = cur.Parent {
		chain = append([]*ExampleGroup{cur}, chain...)
	}
	return chain
}

func commonPrefixLen(a, b []*ExampleGroup) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

// failureOrdinal is assigned when Summary numbers the failures; it feeds the
// documentation "(FAILED - N)" marker, so Documentation must be called after
// Summary (or via Render, which orders them). We store it on the example.
func (r *Reporter) numberOutcomes() (failures, pendings []*Example) {
	for _, e := range r.examples {
		switch e.Result.Status {
		case Failed:
			failures = append(failures, e)
			e.failureOrdinal = len(failures)
		case Pending:
			pendings = append(pendings, e)
			e.pendingOrdinal = len(pendings)
		}
	}
	return failures, pendings
}

// Summary renders the pending section, failures section, totals line, and
// failed-examples rerun list, given a Timing to place in the "Finished in …"
// line. Timing is caller-supplied so the output is deterministic in tests.
func (r *Reporter) Summary(timing Timing) string {
	failures, pendings := r.numberOutcomes()
	var b strings.Builder

	if len(pendings) > 0 {
		b.WriteString("\nPending: (Failures listed here are expected and do not affect your suite's status)\n\n")
		for i, e := range pendings {
			b.WriteString(pendingBlock(i+1, e))
			if i < len(pendings)-1 {
				b.WriteString("\n")
			}
		}
	}

	if len(failures) > 0 {
		b.WriteString("\nFailures:\n\n")
		for i, e := range failures {
			b.WriteString(failureBlock(i+1, e))
			if i < len(failures)-1 {
				b.WriteString("\n")
			}
		}
	}

	fmt.Fprintf(&b, "\nFinished in %s\n", timing.String())
	b.WriteString(TotalsLine(len(r.examples), len(failures), len(pendings), 0) + "\n")

	if len(failures) > 0 {
		b.WriteString("\nFailed examples:\n\n")
		for _, e := range failures {
			fmt.Fprintf(&b, "rspec %s # %s\n", e.Location, e.FullDescription())
		}
	}
	return b.String()
}

// pendingBlock renders one entry of the pending section.
func pendingBlock(n int, e *Example) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  %d) %s\n", n, e.FullDescription())
	reason := e.Result.PendingReason
	if reason == "" {
		reason = "No reason given"
	}
	fmt.Fprintf(&b, "     # %s\n", reason)
	// A pending example that fails as expected also shows its Failure/Error body.
	if e.Result.FailureExpression != "" || e.Result.FailureMessage != "" {
		b.WriteString(failureBody(e))
	}
	return b.String()
}

// failureBlock renders one entry of the Failures section.
func failureBlock(n int, e *Example) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  %d) %s\n", n, e.FullDescription())
	b.WriteString(failureBody(e))
	return b.String()
}

// failureBody renders the "Failure/Error:" line, the indented matcher message,
// and the source location — the shared core of a failure/pending entry.
func failureBody(e *Example) string {
	var b strings.Builder
	fmt.Fprintf(&b, "     Failure/Error: %s\n", e.Result.FailureExpression)
	if e.Result.FailureMessage != "" {
		b.WriteString("\n")
		b.WriteString(indentMessage(e.Result.FailureMessage, "       "))
		b.WriteString("\n")
	}
	if e.Location != "" {
		fmt.Fprintf(&b, "     # %s\n", e.Location)
	}
	return b.String()
}

// indentMessage indents each non-empty line of a matcher message by pad,
// matching how RSpec renders the message under "Failure/Error:".
func indentMessage(msg, pad string) string {
	lines := strings.Split(strings.Trim(msg, "\n"), "\n")
	for i, l := range lines {
		if l == "" {
			continue
		}
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

// Render produces the full progress-formatter report (progress line, blank
// line, then the summary block) for the given timing — the common one-shot
// entry point mirroring `rspec`'s default output.
func (r *Reporter) Render(timing Timing) string {
	return r.Progress() + "\n" + r.Summary(timing)
}

// RenderDoc produces the full documentation-formatter report.
func (r *Reporter) RenderDoc(timing Timing) string {
	// Number outcomes first so the doc "(FAILED - N)" markers are assigned.
	r.numberOutcomes()
	// The documentation formatter opens with a blank line before the tree.
	return "\n" + r.Documentation() + r.Summary(timing)
}

// TotalsLine reproduces RSpec's SummaryNotification#totals_line, e.g.
// "4 examples, 1 failure, 1 pending" with correct pluralisation, and an
// "N error(s) occurred outside of examples" clause when errors > 0.
func TotalsLine(examples, failures, pending, errors int) string {
	parts := []string{
		pluralize(examples, "example"),
		pluralize(failures, "failure"),
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", pending))
	}
	line := strings.Join(parts, ", ")
	if errors > 0 {
		noun := "error"
		if errors != 1 {
			noun = "errors"
		}
		line += fmt.Sprintf(", %d %s occurred outside of examples", errors, noun)
	}
	return line
}

func pluralize(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// Timing is the caller-supplied timing for the "Finished in …" line. RSpec
// formats it as "Finished in X seconds (files took Y seconds to load)"; the
// host measures the durations, so a formatter renders whatever the host reports,
// keeping the rest of the output deterministic.
type Timing struct {
	// Text, when set, is used verbatim (the whole "X seconds (files took …)"
	// clause). Otherwise Run/Load format the standard phrasing.
	Text string
	Run  string // e.g. "0.01 seconds"
	Load string // e.g. "0.03 seconds"
}

func (t Timing) String() string {
	if t.Text != "" {
		return t.Text
	}
	return fmt.Sprintf("%s (files took %s to load)", t.Run, t.Load)
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
