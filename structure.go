// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"sort"
	"strings"
)

// The rspec-core structure model: the describe/context/it registration tree,
// example metadata, hook attachment (before/after/let/subject), filtering
// (tags, :focus) and ordering (defined / random by seed). Evaluating the bodies
// of examples and hooks is the host's job; this package builds and traverses the
// tree, decides which examples run and in what order, and hands the host a flat
// ordered list of runnable examples. Feeding results back drives the formatter.

// Status is an example's outcome, set by the host after it runs the body.
type Status int

const (
	// StatusUnknown means the example has not been run.
	StatusUnknown Status = iota
	// Passed means the example met all expectations.
	Passed
	// Failed means an expectation failed or the body raised.
	Failed
	// Pending means the example is pending (expected to fail) — or was skipped.
	Pending
)

// Example is one `it`/`specify` example.
type Example struct {
	Description string
	Group       *ExampleGroup
	Focused     bool
	Skip        bool     // `xit` / skip: metadata — never runs
	PendingMsg  string   // set when declared pending
	Tags        []string // metadata tag names (e.g. "slow", "focus")
	Location    string   // "path:line" for the failed-examples rerun list
	DefinedAt   int      // monotonic definition index for stable/seeded ordering

	// Result fields, filled by the host after running the body.
	Result Result

	// Ordinals assigned by the formatter when numbering the failures/pending
	// sections (drives the documentation "(FAILED - N)" marker).
	failureOrdinal int
	pendingOrdinal int
}

// Result is the host-reported outcome of running an example body.
type Result struct {
	Status Status
	// FailureMessage is the matcher's message (or exception text) on failure.
	FailureMessage string
	// FailureExpression is the source line RSpec prints after "Failure/Error:".
	FailureExpression string
	// PendingReason is shown for pending examples in the pending section.
	PendingReason string
	// ExceptionClass / backtrace are the host seam for errors; the formatter uses
	// them when present but they are environment-specific, so tests focus on the
	// deterministic message body.
	ExceptionClass string
	Backtrace      []string
}

// Hook is a before/after/around hook or a let/subject definition attached to a
// group. The body is the host's; this model records its kind and scope so the
// host runs them in RSpec's order.
type Hook struct {
	Kind  HookKind
	Scope HookScope
	Name  string // for let/subject: the memoized name
}

// HookKind distinguishes before/after/around and let/subject.
type HookKind int

const (
	// BeforeHook runs before an example (or the group for :context scope).
	BeforeHook HookKind = iota
	// AfterHook runs after.
	AfterHook
	// AroundHook wraps the example.
	AroundHook
	// LetDef is a lazy memoized helper (`let(:name) { … }`).
	LetDef
	// SubjectDef is the group's subject (`subject { … }` / `subject(:name)`).
	SubjectDef
)

// HookScope is :example (default) or :context (once per group).
type HookScope int

const (
	// ExampleScope runs the hook around each example.
	ExampleScope HookScope = iota
	// ContextScope runs the hook once for the whole group.
	ContextScope
)

// ExampleGroup is a describe/context node in the tree.
type ExampleGroup struct {
	Description string
	Parent      *ExampleGroup
	Children    []*ExampleGroup
	Examples    []*Example
	Hooks       []*Hook
	Focused     bool
	Tags        []string
}

// NewRootGroup starts a top-level `RSpec.describe(desc)` group.
func NewRootGroup(desc string) *ExampleGroup { return &ExampleGroup{Description: desc} }

// Describe adds a nested describe/context group.
func (g *ExampleGroup) Describe(desc string) *ExampleGroup {
	child := &ExampleGroup{Description: desc, Parent: g}
	g.Children = append(g.Children, child)
	return child
}

// It adds an example to the group and returns it for further metadata.
func (g *ExampleGroup) It(desc string) *Example {
	e := &Example{Description: desc, Group: g}
	g.Examples = append(g.Examples, e)
	return e
}

// AddHook attaches a hook/let/subject to the group.
func (g *ExampleGroup) AddHook(h *Hook) { g.Hooks = append(g.Hooks, h) }

// Focus marks the group focused (`fdescribe` / `:focus`).
func (g *ExampleGroup) Focus() *ExampleGroup { g.Focused = true; return g }

// Focus marks the example focused (`fit` / `:focus`).
func (e *Example) Focus() *Example { e.Focused = true; return e }

// Tag adds metadata tags to the example (e.g. "slow").
func (e *Example) Tag(names ...string) *Example { e.Tags = append(e.Tags, names...); return e }

// FullDescription is the space-joined chain of group descriptions plus the
// example description — RSpec's `full_description`, used by the documentation
// formatter's rerun/location line and doc labels.
func (e *Example) FullDescription() string {
	return strings.TrimSpace(groupPrefix(e.Group) + e.Description)
}

func groupPrefix(g *ExampleGroup) string {
	if g == nil {
		return ""
	}
	var chain []string
	for cur := g; cur != nil; cur = cur.Parent {
		chain = append([]string{cur.Description}, chain...)
	}
	return strings.Join(chain, " ") + " "
}

// BeforeHooks returns the group's before hooks for the given scope, outermost
// group first (RSpec runs outer before hooks before inner ones).
func (g *ExampleGroup) beforeChain(scope HookScope) []*Hook {
	var chain []*Hook
	var groups []*ExampleGroup
	for cur := g; cur != nil; cur = cur.Parent {
		groups = append([]*ExampleGroup{cur}, groups...)
	}
	for _, gr := range groups {
		for _, h := range gr.Hooks {
			if h.Kind == BeforeHook && h.Scope == scope {
				chain = append(chain, h)
			}
		}
	}
	return chain
}

// afterChain returns after hooks innermost group first (reverse of before).
func (g *ExampleGroup) afterChain(scope HookScope) []*Hook {
	var chain []*Hook
	for cur := g; cur != nil; cur = cur.Parent {
		for _, h := range cur.Hooks {
			if h.Kind == AfterHook && h.Scope == scope {
				chain = append(chain, h)
			}
		}
	}
	return chain
}

// Filter selects which examples run given a filter. It applies :focus
// (when any example/group is focused, only focused ones run), inclusion tags,
// and exclusion tags, then flattens the tree in definition order. The returned
// examples carry the before/after chains the host must run around each.
type Filter struct {
	// IncludeTags, when non-empty, restricts to examples carrying any of them.
	IncludeTags []string
	// ExcludeTags drops examples carrying any of them.
	ExcludeTags []string
	// RunFocused, when true and any example is focused, restricts to focused
	// examples (RSpec's default :focus filter). Detected automatically by
	// SelectExamples when unset via anyFocused.
	RunFocused bool
}

// SelectExamples flattens the tree into the examples that should run under the
// filter, in definition order.
func (g *ExampleGroup) SelectExamples(f Filter) []*Example {
	all := g.flatten()
	focusActive := f.RunFocused || anyFocused(all)
	var out []*Example
	for _, e := range all {
		if e.Skip {
			continue
		}
		if focusActive && !e.Focused && !groupFocused(e.Group) {
			continue
		}
		if len(f.IncludeTags) > 0 && !hasAnyTag(e, f.IncludeTags) {
			continue
		}
		if hasAnyTag(e, f.ExcludeTags) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (g *ExampleGroup) flatten() []*Example {
	var out []*Example
	out = append(out, g.Examples...)
	for _, c := range g.Children {
		out = append(out, c.flatten()...)
	}
	return out
}

func anyFocused(examples []*Example) bool {
	for _, e := range examples {
		if e.Focused || groupFocused(e.Group) {
			return true
		}
	}
	return false
}

func groupFocused(g *ExampleGroup) bool {
	for cur := g; cur != nil; cur = cur.Parent {
		if cur.Focused {
			return true
		}
	}
	return false
}

func hasAnyTag(e *Example, tags []string) bool {
	for _, want := range tags {
		for _, have := range e.Tags {
			if have == want {
				return true
			}
		}
	}
	return false
}

// OrderDefined returns the examples in definition order (RSpec's default).
func OrderDefined(examples []*Example) []*Example {
	out := append([]*Example(nil), examples...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].DefinedAt < out[j].DefinedAt })
	return out
}

// OrderRandom returns the examples shuffled by RSpec's seeded ordering. RSpec
// uses `Ordering::Random`, which shuffles with a Ruby MT-seeded RNG; here we
// use a deterministic Fisher-Yates driven by the same 32-bit seed so a given
// seed yields a stable, reproducible permutation (the ordering is deterministic,
// which is what the golden tests pin; exact parity with MRI's Mersenne Twister
// would require the host RNG).
func OrderRandom(examples []*Example, seed uint32) []*Example {
	out := append([]*Example(nil), examples...)
	r := newLCG(seed)
	for i := len(out) - 1; i > 0; i-- {
		j := int(r.next() % uint32(i+1))
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// lcg is a small deterministic PRNG (numerical-recipes LCG) for seeded ordering.
type lcg struct{ state uint32 }

func newLCG(seed uint32) *lcg {
	if seed == 0 {
		seed = 1
	}
	return &lcg{state: seed}
}

func (l *lcg) next() uint32 {
	l.state = l.state*1664525 + 1013904223
	return l.state
}
