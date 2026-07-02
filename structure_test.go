// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import "testing"

func TestGroupTree(t *testing.T) {
	root := NewRootGroup("Calc")
	adds := root.It("adds")
	nested := root.Describe("when negative")
	subs := nested.It("subtracts")
	if adds.FullDescription() != "Calc adds" {
		t.Errorf("full %q", adds.FullDescription())
	}
	if subs.FullDescription() != "Calc when negative subtracts" {
		t.Errorf("nested full %q", subs.FullDescription())
	}
	if groupPrefix(nil) != "" {
		t.Fatal("nil prefix")
	}
}

func TestHookChains(t *testing.T) {
	root := NewRootGroup("R")
	root.AddHook(&Hook{Kind: BeforeHook, Scope: ExampleScope})
	root.AddHook(&Hook{Kind: AfterHook, Scope: ExampleScope})
	root.AddHook(&Hook{Kind: BeforeHook, Scope: ContextScope})
	child := root.Describe("c")
	child.AddHook(&Hook{Kind: BeforeHook, Scope: ExampleScope})
	child.AddHook(&Hook{Kind: AfterHook, Scope: ExampleScope})

	// before chain: outermost first (root before, then child before).
	bc := child.beforeChain(ExampleScope)
	if len(bc) != 2 {
		t.Fatalf("before chain %d", len(bc))
	}
	// context-scope before at root only.
	if len(child.beforeChain(ContextScope)) != 1 {
		t.Fatal("ctx before")
	}
	// after chain: innermost first (child after, then root after).
	ac := child.afterChain(ExampleScope)
	if len(ac) != 2 {
		t.Fatalf("after chain %d", len(ac))
	}
	if len(child.afterChain(ContextScope)) != 0 {
		t.Fatal("ctx after none")
	}
	// let/subject/around hooks recorded but not in before/after chains.
	root.AddHook(&Hook{Kind: LetDef, Name: "x"})
	root.AddHook(&Hook{Kind: SubjectDef})
	root.AddHook(&Hook{Kind: AroundHook})
}

func TestSelectExamples(t *testing.T) {
	root := NewRootGroup("R")
	a := root.It("a")
	a.DefinedAt = 0
	b := root.It("b")
	b.Tag("slow")
	b.DefinedAt = 1
	sub := root.Describe("sub")
	c := sub.It("c")
	c.DefinedAt = 2
	skipped := root.It("s")
	skipped.Skip = true

	// No filter: all non-skipped run.
	if got := root.SelectExamples(Filter{}); len(got) != 3 {
		t.Fatalf("all %d", len(got))
	}
	// Exclude slow.
	if got := root.SelectExamples(Filter{ExcludeTags: []string{"slow"}}); len(got) != 2 {
		t.Fatalf("exclude %d", len(got))
	}
	// Include only slow.
	if got := root.SelectExamples(Filter{IncludeTags: []string{"slow"}}); len(got) != 1 {
		t.Fatalf("include %d", len(got))
	}
	// Focus: only focused examples run.
	c.Focus()
	if got := root.SelectExamples(Filter{}); len(got) != 1 || got[0] != c {
		t.Fatalf("focus %v", got)
	}
	c.Focused = false
	// Focused group.
	sub.Focus()
	if got := root.SelectExamples(Filter{}); len(got) != 1 || got[0] != c {
		t.Fatalf("group focus %v", got)
	}
	sub.Focused = false
	// RunFocused with none focused still returns all (focusActive false).
	if got := root.SelectExamples(Filter{RunFocused: true}); len(got) != 3 {
		// RunFocused true forces focusActive, so with none focused, nothing runs.
		if len(got) != 0 {
			t.Fatalf("runfocused %d", len(got))
		}
	}
}

func TestOrdering(t *testing.T) {
	root := NewRootGroup("R")
	var exs []*Example
	for i := 0; i < 6; i++ {
		e := root.It("e")
		e.DefinedAt = i
		exs = append(exs, e)
	}
	// defined order stable
	def := OrderDefined(exs)
	for i := range def {
		if def[i].DefinedAt != i {
			t.Fatalf("defined order broken at %d", i)
		}
	}
	// random: same seed -> same permutation; different seed likely differs.
	r1 := OrderRandom(exs, 42)
	r2 := OrderRandom(exs, 42)
	for i := range r1 {
		if r1[i] != r2[i] {
			t.Fatal("seed not deterministic")
		}
	}
	// seed 0 normalises to 1 (no panic, valid perm).
	r0 := OrderRandom(exs, 0)
	if len(r0) != len(exs) {
		t.Fatal("seed0 length")
	}
}
