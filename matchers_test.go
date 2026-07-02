// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"math"
	"math/big"
	"testing"
)

// fmsg / nmsg drive a matcher and return its positive / negated failure message.
func fmsg(m Matcher, a any) string { m.Matches(a); return m.FailureMessage() }
func nmsg(m Matcher, a any) string { m.Matches(a); return m.FailureMessageNegated() }

// bigOf parses a decimal into *big.Int for the tests.
func bigOf(s string) *big.Int { b, _ := new(big.Int).SetString(s, 10); return b }

// hashOf builds an ordered Hash from key/value pairs.
func hashOf(kv ...any) *Hash {
	h := NewHash()
	for i := 0; i+1 < len(kv); i += 2 {
		h.Set(kv[i], kv[i+1])
	}
	return h
}

func TestEqMatcher(t *testing.T) {
	if !Eq(1).Matches(1) {
		t.Fatal("1 eq 1")
	}
	if Eq(1).Matches(2) {
		t.Fatal("1 !eq 2")
	}
	// Cross-type numeric equality: 1 == 1.0.
	if !Eq(1).Matches(1.0) {
		t.Fatal("1 == 1.0")
	}
	want := "\nexpected: 1\n     got: 2\n\n(compared using ==)\n"
	if g := fmsg(Eq(1), 2); g != want {
		t.Errorf("got %q", g)
	}
	wantN := "\nexpected: value != 1\n     got: 1\n\n(compared using ==)\n"
	if g := nmsg(Eq(1), 1); g != wantN {
		t.Errorf("neg got %q", g)
	}
	if d := Eq(1).(Describer).Description(); d != "eq 1" {
		t.Errorf("desc %q", d)
	}
}

func TestEqlMatcher(t *testing.T) {
	if !Eql(1).Matches(1) {
		t.Fatal("1 eql 1")
	}
	if Eql(1).Matches(1.0) {
		t.Fatal("1 !eql 1.0")
	}
	if !Eql(1.5).Matches(1.5) {
		t.Fatal("1.5 eql 1.5")
	}
	if Eql(1.5).Matches(2.5) {
		t.Fatal("floats differ")
	}
	if Eql(1).Matches("x") {
		t.Fatal("int vs string")
	}
	if !Eql("a").Matches("a") {
		t.Fatal("string eql")
	}
	want := "\nexpected: 1\n     got: 1.0\n\n(compared using eql?)\n"
	if g := fmsg(Eql(1), 1.0); g != want {
		t.Errorf("got %q", g)
	}
	if g := nmsg(Eql(1), 1); g != "\nexpected: value != 1\n     got: 1\n\n(compared using eql?)\n" {
		t.Errorf("neg %q", g)
	}
}

func TestEqualMatcher(t *testing.T) {
	// Immutable identity: symbols/ints are equal? when equal.
	if !Equal(Symbol("a")).Matches(Symbol("a")) {
		t.Fatal("sym identity")
	}
	if !Equal(5).Matches(5) {
		t.Fatal("int identity")
	}
	// Two equal-but-distinct strings are NOT equal?.
	if Equal("a").Matches("a") {
		t.Fatal("distinct strings")
	}
	// Same *Object identity by ID.
	o1 := &Object{Class: "Foo", ID: 7}
	o2 := &Object{Class: "Foo", ID: 7}
	if !Equal(o1).Matches(o2) {
		t.Fatal("same id")
	}
	if Equal(o1).Matches(&Object{Class: "Foo", ID: 8}) {
		t.Fatal("diff id")
	}
	// Immutable failure message form.
	if g := fmsg(Equal(1), 2); g != "\nexpected: 1\n     got: 2\n\n(compared using equal?)\n" {
		t.Errorf("imm %q", g)
	}
	// Reference failure message form.
	gm := fmsg(Equal("a"), "b")
	if gm == "" || gm[:10] != "\nexpected " {
		t.Errorf("ref %q", gm)
	}
	_ = nmsg(Equal("a"), "a")
	_ = objectIDRef([]any{1})
	_ = objectIDRef(hashOf())
	_ = objectIDRef(o1)
	_ = objectIDRef(1.5)
}

func TestBeTruthyFalseyNil(t *testing.T) {
	if !BeTruthy().Matches(1) || BeTruthy().Matches(nil) || BeTruthy().Matches(false) {
		t.Fatal("truthy")
	}
	if !BeFalsey().Matches(nil) || !BeFalsey().Matches(false) || BeFalsey().Matches(1) {
		t.Fatal("falsey")
	}
	if !BeNil().Matches(nil) || BeNil().Matches(0) {
		t.Fatal("nil")
	}
	if g := fmsg(BeTruthy(), nil); g != "expected: truthy value\n     got: nil" {
		t.Errorf("%q", g)
	}
	if g := nmsg(BeTruthy(), 1); g != "expected: falsey value\n     got: 1" {
		t.Errorf("%q", g)
	}
	if g := fmsg(BeFalsey(), 1); g != "expected: falsey value\n     got: 1" {
		t.Errorf("%q", g)
	}
	if g := nmsg(BeFalsey(), nil); g != "expected: truthy value\n     got: nil" {
		t.Errorf("%q", g)
	}
	if g := fmsg(BeNil(), 1); g != "expected: nil\n     got: 1" {
		t.Errorf("%q", g)
	}
	if g := nmsg(BeNil(), nil); g != "expected: not nil\n     got: nil" {
		t.Errorf("%q", g)
	}
}

func TestBeComparison(t *testing.T) {
	if !BeGreaterThan(5).Matches(7) || BeGreaterThan(5).Matches(3) {
		t.Fatal(">")
	}
	if !BeGreaterOrEqual(5).Matches(5) || !BeLessThan(5).Matches(3) || !BeLessOrEqual(5).Matches(5) {
		t.Fatal("cmp")
	}
	if !BeEqualOp(5).Matches(5) || !BeNotEqualOp(5).Matches(6) {
		t.Fatal("eq/ne")
	}
	// Incomparable operands never match.
	if BeGreaterThan(5).Matches("x") {
		t.Fatal("incomparable")
	}
	if g := fmsg(BeGreaterThan(5), 3); g != "expected: > 5\n     got:   3" {
		t.Errorf("%q", g)
	}
	if g := fmsg(BeGreaterOrEqual(5), 3); g != "expected: >= 5\n     got:    3" {
		t.Errorf("%q", g)
	}
	if g := fmsg(BeLessThan(5), 7); g != "expected: < 5\n     got:   7" {
		t.Errorf("%q", g)
	}
	if g := fmsg(BeEqualOp(5), 7); g != "expected: == 5\n     got:    7" {
		t.Errorf("%q", g)
	}
	if g := nmsg(BeEqualOp(5), 5); g != "`expect(5).not_to be == 5`" {
		t.Errorf("neg %q", g)
	}
	if g := nmsg(BeGreaterThan(5), 7); g != "`expect(7).not_to be > 5` not only FAILED, it is a bit confusing." {
		t.Errorf("neg gt %q", g)
	}
	if d := BeGreaterThan(2).(Describer).Description(); d != "be > 2" {
		t.Errorf("desc %q", d)
	}
	// string comparison path
	if !BeLessThan("m").Matches("a") {
		t.Fatal("string <")
	}
}

func TestBeKindInstanceOf(t *testing.T) {
	if !BeKindOf("Numeric").Matches(1) || BeKindOf("String").Matches(1) {
		t.Fatal("kind_of")
	}
	if !BeInstanceOf("Integer").Matches(1) || BeInstanceOf("Integer").Matches(1.0) {
		t.Fatal("instance_of")
	}
	if g := fmsg(BeKindOf("String"), 1); g != "expected 1 to be a kind of String" {
		t.Errorf("%q", g)
	}
	if g := nmsg(BeKindOf("Integer"), 5); g != "expected 5 not to be a kind of Integer" {
		t.Errorf("%q", g)
	}
	if g := fmsg(BeInstanceOf("Integer"), 1.0); g != "expected 1.0 to be an instance of Integer" {
		t.Errorf("%q", g)
	}
	if g := nmsg(BeInstanceOf("Integer"), 1); g != "expected 1 not to be an instance of Integer" {
		t.Errorf("%q", g)
	}
	if d := BeKindOf("Integer").(Describer).Description(); d != "be a kind of Integer" {
		t.Errorf("desc %q", d)
	}
}

func TestMatchMatcher(t *testing.T) {
	if !Match(&Regexp{Source: "fo+"}).Matches("foobar") {
		t.Fatal("regex match")
	}
	if Match(&Regexp{Source: "z"}).Matches("foobar") {
		t.Fatal("no match")
	}
	if Match(&Regexp{Source: "("}).Matches("x") {
		t.Fatal("bad regex")
	}
	if Match(&Regexp{Source: "x"}).Matches(123) {
		t.Fatal("non-string")
	}
	if !Match("a.c").Matches("abc") {
		t.Fatal("string pattern")
	}
	if !Match("(").Matches("(") { // invalid regex falls back to literal ==
		t.Fatal("literal fallback")
	}
	if Match(42).Matches("x") {
		t.Fatal("unsupported pattern")
	}
	if g := fmsg(Match(&Regexp{Source: "foo"}), "bar"); g != "expected \"bar\" to match /foo/" {
		t.Errorf("%q", g)
	}
	if g := nmsg(Match(&Regexp{Source: "x"}), "y"); g != "expected \"y\" not to match /x/" {
		t.Errorf("%q", g)
	}
	if d := Match(&Regexp{Source: "x"}).(Describer).Description(); d != "match /x/" {
		t.Errorf("desc %q", d)
	}
}

func TestIncludeMatcher(t *testing.T) {
	if !Include(2).Matches([]any{1, 2, 3}) || Include(9).Matches([]any{1}) {
		t.Fatal("array include")
	}
	if !Include("b").Matches("abc") || Include("z").Matches("abc") {
		t.Fatal("string include")
	}
	if Include(1).Matches("abc") { // non-string want on string
		t.Fatal("string include non-string")
	}
	if !Include(Symbol("a")).Matches(hashOf(Symbol("a"), 1)) {
		t.Fatal("hash key include")
	}
	if !Include(hashOf(Symbol("a"), 1)).Matches(hashOf(Symbol("a"), 1)) {
		t.Fatal("hash kv include")
	}
	if Include(hashOf(Symbol("a"), 2)).Matches(hashOf(Symbol("a"), 1)) {
		t.Fatal("hash kv mismatch")
	}
	if Include(1).Matches(42) { // unsupported actual
		t.Fatal("unsupported actual")
	}
	if g := fmsg(Include(3), []any{1, 2}); g != "expected [1, 2] to include 3" {
		t.Errorf("%q", g)
	}
	if g := fmsg(Include(1, 5), []any{1, 2}); g != "expected [1, 2] to include 5" {
		t.Errorf("%q", g)
	}
	if g := nmsg(Include(1), []any{1, 2}); g != "expected [1, 2] not to include 1" {
		t.Errorf("%q", g)
	}
}

func TestStartEndWith(t *testing.T) {
	if !StartWith("fo").Matches("foobar") || StartWith("x").Matches("foobar") {
		t.Fatal("start string")
	}
	if !EndWith("ar").Matches("foobar") || EndWith("x").Matches("foobar") {
		t.Fatal("end string")
	}
	if StartWith("a", "b").Matches("str") { // multi-arg on string invalid
		t.Fatal("multi string")
	}
	if StartWith(1).Matches("str") { // non-string arg on string
		t.Fatal("nonstring arg")
	}
	if !StartWith(1, 2).Matches([]any{1, 2, 3}) || StartWith(2).Matches([]any{1, 2}) {
		t.Fatal("start array")
	}
	if !EndWith(2, 3).Matches([]any{1, 2, 3}) || EndWith(9).Matches([]any{1, 2}) {
		t.Fatal("end array")
	}
	if StartWith(1, 2, 3, 4).Matches([]any{1}) { // too many
		t.Fatal("too many")
	}
	if StartWith(1).Matches(42) { // unsupported actual
		t.Fatal("unsupported")
	}
	if g := fmsg(StartWith("foo"), "bar"); g != "expected \"bar\" to start with \"foo\"" {
		t.Errorf("%q", g)
	}
	if g := fmsg(EndWith("foo"), "bar"); g != "expected \"bar\" to end with \"foo\"" {
		t.Errorf("%q", g)
	}
	if g := nmsg(StartWith(1), []any{1, 2}); g != "expected [1, 2] not to start with 1" {
		t.Errorf("%q", g)
	}
}

func TestContainExactly(t *testing.T) {
	if !ContainExactly(1, 2, 3).Matches([]any{3, 2, 1}) {
		t.Fatal("reorder")
	}
	if ContainExactly(1, 2).Matches([]any{1, 2, 3}) {
		t.Fatal("extra")
	}
	if ContainExactly(1, 2, 3).Matches([]any{1, 2}) {
		t.Fatal("missing")
	}
	if ContainExactly(1).Matches("notarray") {
		t.Fatal("non-array")
	}
	if !MatchArray([]any{1, 2}).Matches([]any{2, 1}) {
		t.Fatal("match_array")
	}
	if g := fmsg(ContainExactly(1, 2), []any{1, 2, 3}); g != "expected collection contained:  [1, 2]\nactual collection contained:    [1, 2, 3]\nthe extra elements were:        [3]\n" {
		t.Errorf("%q", g)
	}
	if g := fmsg(ContainExactly(1, 2, 3), []any{1, 2}); g != "expected collection contained:  [1, 2, 3]\nactual collection contained:    [1, 2]\nthe missing elements were:      [3]\n" {
		t.Errorf("%q", g)
	}
	if g := fmsg(MatchArray([]any{1, 2, 4}), []any{1, 2, 3}); g != "expected collection contained:  [1, 2, 4]\nactual collection contained:    [1, 2, 3]\nthe missing elements were:      [4]\nthe extra elements were:        [3]\n" {
		t.Errorf("%q", g)
	}
	if g := nmsg(ContainExactly(1, 2, 3), []any{1, 2, 3}); g != "expected [1, 2, 3] not to contain exactly 1, 2, and 3" {
		t.Errorf("%q", g)
	}
	if g := nmsg(ContainExactly(1), []any{1}); g != "expected [1] not to contain exactly 1" {
		t.Errorf("%q", g)
	}
}

func TestHaveAttributes(t *testing.T) {
	if !HaveAttributes(hashOf(Symbol("size"), 3)).Matches("abc") {
		t.Fatal("size match")
	}
	if HaveAttributes(hashOf(Symbol("size"), 3)).Matches("ab") {
		t.Fatal("size mismatch")
	}
	// Missing attribute (no method).
	if HaveAttributes(hashOf(Symbol("nope"), 1)).Matches("ab") {
		t.Fatal("missing attr")
	}
	if g := fmsg(HaveAttributes(hashOf(Symbol("size"), 3)), "ab"); g != "expected \"ab\" to have attributes {size: 3} but had attributes {size: 2}" {
		t.Errorf("%q", g)
	}
	// Sorted multi-key.
	m := HaveAttributes(hashOf(Symbol("size"), 3, Symbol("length"), 4))
	if g := fmsg(m, "ab"); g != "expected \"ab\" to have attributes {length: 4, size: 3} but had attributes {length: 2, size: 2}" {
		t.Errorf("%q", g)
	}
	if g := nmsg(HaveAttributes(hashOf(Symbol("size"), 2)), "ab"); g != "expected \"ab\" not to have attributes {size: 2}" {
		t.Errorf("%q", g)
	}
	// String key and non-symbol/string key in attrKey.
	_ = attrKey("plain")
	_ = attrKey(5)
}

func TestAllMatcher(t *testing.T) {
	if !All(BeGreaterThan(0)).Matches([]any{1, 2, 3}) {
		t.Fatal("all pass")
	}
	if All(BeGreaterThan(2)).Matches([]any{3, 1, 4}) {
		t.Fatal("all fail")
	}
	if All(BeGreaterThan(0)).Matches("notarray") {
		t.Fatal("non-array")
	}
	want := "expected [3, 1, 4] to all be > 2\n\n   object at index 1 failed to match:\n      expected: > 2\n           got:   1"
	if g := fmsg(All(BeGreaterThan(2)), []any{3, 1, 4}); g != want {
		t.Errorf("%q", g)
	}
	if g := nmsg(All(BeGreaterThan(2)), []any{3, 1}); g != "expected [3, 1] not to all be > 2" {
		t.Errorf("%q", g)
	}
	// inner without Describer falls back to "match".
	_ = innerDescription(All(BeGreaterThan(0)))
}

func TestRespondTo(t *testing.T) {
	if !RespondTo(Symbol("size")).Matches("x") || RespondTo(Symbol("foo")).Matches("x") {
		t.Fatal("respond string")
	}
	if !RespondTo(Symbol("size"), Symbol("length")).Matches([]any{1}) {
		t.Fatal("array methods")
	}
	if !RespondTo(Symbol("keys")).Matches(hashOf()) {
		t.Fatal("hash methods")
	}
	o := &Object{Class: "Foo", RespondsTo: []string{"bar"}}
	if !RespondTo(Symbol("bar")).Matches(o) || RespondTo(Symbol("baz")).Matches(o) {
		t.Fatal("object respond")
	}
	oi := &Object{Class: "Foo", IVars: map[string]any{"@x": 1}}
	if !RespondTo(Symbol("x")).Matches(oi) {
		t.Fatal("ivar responder")
	}
	if g := fmsg(RespondTo(Symbol("foo")), "x"); g != "expected \"x\" to respond to :foo" {
		t.Errorf("%q", g)
	}
	if g := nmsg(RespondTo(Symbol("size")), "x"); g != "expected \"x\" not to respond to :size" {
		t.Errorf("%q", g)
	}
	// with(n).arguments suffix
	m := RespondTo(Symbol("foo")).With(2)
	if g := fmsg(m, "x"); g != "expected \"x\" to respond to :foo with 2 arguments" {
		t.Errorf("%q", g)
	}
	m1 := RespondTo(Symbol("foo")).With(1)
	if g := fmsg(m1, "x"); g != "expected \"x\" to respond to :foo with 1 argument" {
		t.Errorf("%q", g)
	}
}

func TestSatisfy(t *testing.T) {
	if !Satisfy("", "x > 1", func(a any) bool { return true }).Matches(2) {
		t.Fatal("pass")
	}
	if g := fmsg(Satisfy("", "x > 5", func(a any) bool { return false }), 3); g != "expected 3 to satisfy expression `x > 5`" {
		t.Errorf("%q", g)
	}
	if g := fmsg(Satisfy("be prime", "", func(a any) bool { return false }), 4); g != "expected 4 to be prime" {
		t.Errorf("%q", g)
	}
	if g := nmsg(Satisfy("be prime", "", func(a any) bool { return true }), 5); g != "expected 5 not to be prime" {
		t.Errorf("%q", g)
	}
}

func TestBeWithin(t *testing.T) {
	if !BeWithin(0.5).Of(10).Matches(10.3) || BeWithin(0.5).Of(10).Matches(11) {
		t.Fatal("within")
	}
	if BeWithin(0.5).Of(10).Matches("x") {
		t.Fatal("non-numeric")
	}
	if g := fmsg(BeWithin(0.5).Of(10), 11); g != "expected 11 to be within 0.5 of 10" {
		t.Errorf("%q", g)
	}
	if g := nmsg(BeWithin(5.0).Of(10), 11); g != "expected 11 not to be within 5.0 of 10" {
		t.Errorf("%q", g)
	}
	// non-integral centre renders as float
	if g := fmsg(BeWithin(0.1).Of(2.5), 3.0); g != "expected 3.0 to be within 0.1 of 2.5" {
		t.Errorf("%q", g)
	}
	if numFromFloat(math.Inf(1)) == nil {
		t.Fatal("inf")
	}
}

func TestCover(t *testing.T) {
	r := &Range{Begin: 1, End: 5}
	if !Cover(3).Matches(r) || Cover(9).Matches(r) {
		t.Fatal("cover")
	}
	if !Cover(5).Matches(r) { // inclusive end
		t.Fatal("inclusive")
	}
	ex := &Range{Begin: 1, End: 5, Exclusive: true}
	if Cover(5).Matches(ex) {
		t.Fatal("exclusive end")
	}
	if Cover(3).Matches("notrange") {
		t.Fatal("non-range")
	}
	if Cover("x").Matches(r) { // incomparable
		t.Fatal("incomparable value")
	}
	// begin incomparable
	if Cover(0).Matches(&Range{Begin: "a", End: "z"}) {
		t.Fatal("begin incomparable numeric")
	}
	if g := fmsg(Cover(5), &Range{Begin: 1, End: 3}); g != "expected 1..3 to cover 5" {
		t.Errorf("%q", g)
	}
	if g := fmsg(Cover(2, 5, 9), &Range{Begin: 1, End: 3}); g != "expected 1..3 to cover 2, 5, and 9" {
		t.Errorf("%q", g)
	}
	if g := nmsg(Cover(2), &Range{Begin: 1, End: 3}); g != "expected 1..3 not to cover 2" {
		t.Errorf("%q", g)
	}
}

func TestPredicate(t *testing.T) {
	if !BePredicate("empty").Matches([]any{}) || BePredicate("empty").Matches([]any{1}) {
		t.Fatal("array empty")
	}
	if !BePredicate("empty").Matches("") || !BePredicate("empty").Matches(hashOf()) {
		t.Fatal("empty variants")
	}
	// Object with stored predicate.
	o := &Object{Class: "F", IVars: map[string]any{"?valid": true}}
	if !BePredicate("valid").Matches(o) {
		t.Fatal("object predicate")
	}
	// Object declaring the predicate method but no value -> matches? false, responds true.
	od := &Object{Class: "F", RespondsTo: []string{"ready?"}}
	m := BePredicate("ready")
	m.Matches(od)
	// Unknown predicate -> "respond to" message.
	if g := fmsg(BePredicate("valid"), &Object{Class: "Obj"}); g != "expected #<Obj:0x0000000000000000> to respond to `valid?`" {
		t.Errorf("%q", g)
	}
	if g := fmsg(BePredicate("empty"), []any{1}); g != "expected `[1].empty?` to be truthy, got false" {
		t.Errorf("%q", g)
	}
	if g := nmsg(BePredicate("empty"), []any{}); g != "expected `[].empty?` to be falsey, got true" {
		t.Errorf("%q", g)
	}
}

func TestCompose(t *testing.T) {
	if !And(BeKindOf("Integer"), BeGreaterThan(0)).Matches(5) {
		t.Fatal("and pass")
	}
	if And(BeKindOf("Integer"), BeGreaterThan(0)).Matches(-1) {
		t.Fatal("and right fail")
	}
	if And(BeKindOf("String"), BeGreaterThan(0)).Matches(5) {
		t.Fatal("and left fail")
	}
	if !Or(Eq(1), Eq(2)).Matches(2) || Or(Eq(1), Eq(2)).Matches(3) {
		t.Fatal("or")
	}
	// AND failure message = the failing sub-matcher's.
	if g := fmsg(And(BeKindOf("Integer"), BeGreaterThan(5)), 3); g != "expected: > 5\n     got:   3" {
		t.Errorf("and %q", g)
	}
	// AND left failure path.
	af := And(BeKindOf("String"), BeGreaterThan(0))
	af.Matches(5)
	_ = af.FailureMessage()
	// AND without prior Matches uses left.
	if And(Eq(1), Eq(2)).FailureMessage() == "" {
		t.Fatal("and default msg")
	}
	if d := And(Eq(1), BeLessThan(5)).(Describer).Description(); d != "eq 1 and be < 5" {
		t.Errorf("and desc %q", d)
	}
	orWant := "\n   expected: 1\n        got: 3\n\n   (compared using ==)\n\n...or:\n\n   expected: 2\n        got: 3\n\n   (compared using ==)\n"
	if g := fmsg(Or(Eq(1), Eq(2)), 3); g != orWant {
		t.Errorf("or %q", g)
	}
	if d := Or(Eq(1), Eq(2)).(Describer).Description(); d != "eq 1 or eq 2" {
		t.Errorf("or desc %q", d)
	}
	_ = And(Eq(1), Eq(2)).FailureMessageNegated()
	_ = Or(Eq(1), Eq(2)).FailureMessageNegated()
}

func TestExpect(t *testing.T) {
	if ok, msg := Expect(1, Eq(1), true); !ok || msg != "" {
		t.Fatal("pass to")
	}
	if ok, _ := Expect(1, Eq(2), true); ok {
		t.Fatal("fail to")
	}
	if ok, _ := Expect(1, Eq(2), false); !ok {
		t.Fatal("pass not_to")
	}
	if ok, _ := Expect(1, Eq(1), false); ok {
		t.Fatal("fail not_to")
	}
}

func TestBigAndInspectInMatchers(t *testing.T) {
	if !Eq(bigOf("123456789012345678901234567890")).Matches(bigOf("123456789012345678901234567890")) {
		t.Fatal("big eq")
	}
}
