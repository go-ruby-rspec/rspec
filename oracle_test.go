// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// The differential RSpec oracle. Each matcher is exercised on both sides — this
// package and the real rspec-expectations gem — on a pass and a fail actual, and
// the boolean match result plus the two failure-message strings are compared
// byte-for-byte. The formatter's totals-line grammar is likewise checked against
// rspec-core's SummaryNotification. The deterministic, ruby-free golden-vector
// tests hold coverage at 100% on their own, so these oracle checks skip
// themselves where ruby / the rspec gems are absent (the qemu cross-arch lanes
// and Windows), and gate on RUBY_VERSION >= "4.0" per the org convention.

// rubyGate locates a usable ruby with the rspec gems and a new-enough version.
// It skips the test otherwise so the ruby-free lanes still pass.
func rubyGate(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping RSpec oracle")
	}
	out, err := exec.Command(bin, "-e", `
		exit 2 if RUBY_VERSION < "4.0"
		begin
			require "rspec/expectations"; require "rspec/core"
		rescue LoadError
			exit 3
		end
		print "ok"
	`).CombinedOutput()
	if err != nil || string(out) != "ok" {
		t.Skipf("ruby unusable for RSpec oracle (RUBY_VERSION gate / rspec gems): %s", strings.TrimSpace(string(out)))
	}
	return bin
}

// oracleRow is one matcher case emitted by the Ruby side.
type oracleRow struct {
	Name    string `json:"name"`
	Matched bool   `json:"matched"`
	Msg     string `json:"msg"`  // failure_message
	MsgN    string `json:"msgn"` // failure_message_when_negated
}

// rubyOracle runs the RSpec side over the shared case table and returns the
// per-case results keyed by name.
func rubyOracle(t *testing.T, bin string) map[string]oracleRow {
	t.Helper()
	script := `
require "rspec/expectations"
require "json"
M = Object.new
M.extend RSpec::Matchers
def row(name, m, actual)
  matched = (m.matches?(actual) == true)
  msg  = matched ? "" : (m.failure_message rescue "")
  msgn = (m.matches?(actual) ? (m.failure_message_when_negated rescue "") : "")
  {name: name, matched: matched, msg: msg, msgn: msgn}
end
rows = []
rows << row("eq_fail",   M.eq(1), 2)
rows << row("eq_pass",   M.eq(1), 1)
rows << row("eql_fail",  M.eql(1), 1.0)
rows << row("be_truthy", M.be_truthy, nil)
rows << row("be_nil",    M.be_nil, 1)
rows << row("be_gt",     M.be > 5, 3)
rows << row("be_ge",     M.be >= 5, 3)
rows << row("be_a",      M.be_a(String), 1)
rows << row("be_inst",   M.be_instance_of(Integer), 1.0)
rows << row("match",     M.match(/foo/), "bar")
rows << row("include",   M.include(3), [1,2])
rows << row("start",     M.start_with("foo"), "bar")
rows << row("endw",      M.end_with("foo"), "bar")
rows << row("contain",   M.contain_exactly(1,2), [1,2,3])
rows << row("respond",   M.respond_to(:foo), "x")
rows << row("within",    M.be_within(0.5).of(10), 11)
rows << row("cover",     M.cover(5), (1..3))
rows << row("empty",     M.be_empty, [1])
rows << row("all",       M.all(M.be > 2), [3,1,4])
print JSON.generate(rows)
`
	out, err := exec.Command(bin, "-e", script).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby oracle error: %v\n%s", err, out)
	}
	var rows []oracleRow
	if err := json.Unmarshal(out, &rows); err != nil {
		t.Fatalf("decoding oracle output: %v\n%s", err, out)
	}
	m := map[string]oracleRow{}
	for _, r := range rows {
		m[r.Name] = r
	}
	return m
}

// goCase mirrors a Ruby oracle row: a matcher, its actual, and the case name.
type goCase struct {
	name   string
	m      Matcher
	actual any
}

func TestOracleMatcherParity(t *testing.T) {
	bin := rubyGate(t)
	oracle := rubyOracle(t, bin)

	cases := []goCase{
		{"eq_fail", Eq(1), 2},
		{"eq_pass", Eq(1), 1},
		{"eql_fail", Eql(1), 1.0},
		{"be_truthy", BeTruthy(), nil},
		{"be_nil", BeNil(), 1},
		{"be_gt", BeGreaterThan(5), 3},
		{"be_ge", BeGreaterOrEqual(5), 3},
		{"be_a", BeKindOf("String"), 1},
		{"be_inst", BeInstanceOf("Integer"), 1.0},
		{"match", Match(&Regexp{Source: "foo"}), "bar"},
		{"include", Include(3), []any{1, 2}},
		{"start", StartWith("foo"), "bar"},
		{"endw", EndWith("foo"), "bar"},
		{"contain", ContainExactly(1, 2), []any{1, 2, 3}},
		{"respond", RespondTo(Symbol("foo")), "x"},
		{"within", BeWithin(0.5).Of(10), 11},
		{"cover", Cover(5), &Range{Begin: 1, End: 3}},
		{"empty", BePredicate("empty"), []any{1}},
		{"all", All(BeGreaterThan(2)), []any{3, 1, 4}},
	}

	for _, c := range cases {
		want, ok := oracle[c.name]
		if !ok {
			t.Errorf("%s: missing from oracle", c.name)
			continue
		}
		matched := c.m.Matches(c.actual)
		if matched != want.Matched {
			t.Errorf("%s: matched=%v, oracle=%v", c.name, matched, want.Matched)
		}
		if !matched {
			if got := c.m.FailureMessage(); got != want.Msg {
				t.Errorf("%s failure_message:\n got: %q\nwant: %q", c.name, got, want.Msg)
			}
		} else if want.MsgN != "" {
			if got := c.m.FailureMessageNegated(); got != want.MsgN {
				t.Errorf("%s failure_message_when_negated:\n got: %q\nwant: %q", c.name, got, want.MsgN)
			}
		}
	}
}

// TestOracleTotalsLine checks the formatter's totals-line grammar against
// rspec-core's SummaryNotification#totals_line for a range of result counts.
func TestOracleTotalsLine(t *testing.T) {
	bin := rubyGate(t)
	script := `
require "rspec/core"
require "json"
N = RSpec::Core::Notifications::SummaryNotification
def totals(examples, failures, pending, errors)
  args = [1.0, Array.new(examples), Array.new(failures), Array.new(pending), 1.0, errors]
  N.new(*args[0, N.members.size]).totals_line
end
combos = [[1,0,0,0],[2,0,0,0],[4,1,1,0],[4,2,0,0],[1,1,0,0],[3,0,2,0],[0,0,0,0],[5,2,1,1]]
print JSON.generate(combos.map { |c| totals(*c) })
`
	out, err := exec.Command(bin, "-e", script).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby totals error: %v\n%s", err, out)
	}
	var want []string
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	combos := [][4]int{{1, 0, 0, 0}, {2, 0, 0, 0}, {4, 1, 1, 0}, {4, 2, 0, 0}, {1, 1, 0, 0}, {3, 0, 2, 0}, {0, 0, 0, 0}, {5, 2, 1, 1}}
	for i, c := range combos {
		if got := TotalsLine(c[0], c[1], c[2], c[3]); got != want[i] {
			t.Errorf("totals %v:\n got: %q\nwant: %q", c, got, want[i])
		}
	}
}
