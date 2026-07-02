<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-rspec/brand/main/social/go-ruby-rspec-rspec.png" alt="go-ruby-rspec/rspec" width="720"></p>

# rspec — go-ruby-rspec

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-rspec.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the deterministic core of Ruby's
[RSpec](https://rspec.info/)** — the `rspec-expectations` matchers and the
`rspec-core` example-group structure model and formatters. It answers the parts
of a spec run that are pure functions of values and results: *does a matcher
match*, *what failure message does it produce*, *which examples run and in what
order*, and *how does the formatter render a set of results* — all byte-faithful
to RSpec 3.13, and **without any Ruby runtime**.

It is the RSpec backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module with no dependency on the Ruby runtime — a
sibling of [go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) (the Onigmo
engine) and [go-ruby-erb](https://github.com/go-ruby-erb/erb) (the ERB compiler).

> **What it is — and isn't.** A matcher's match logic, its failure-message
> strings, the describe/context/it tree, filtering, ordering, and the formatter
> output are all deterministic and need **no interpreter**, so they live here as
> pure Go. **Running the body of an `it` example or a `before`/`let` hook is the
> host's job** (rbgo evaluates the Ruby): the block-form matchers (`change`,
> `raise_error`) consume the host's *observation* of what a block did, and the
> reporter is driven by host-supplied results.

## Features

### rspec-expectations matchers

Every matcher carries `Matches(actual)` plus `FailureMessage()` /
`FailureMessageNegated()`, rendered byte-for-byte against the gem:

- **Equality** — `eq` (`==`), `eql` (`eql?`), `equal` (`equal?` identity), with
  Ruby's cross-type numeric `==` and the distinctive `equal?` object-identity
  message.
- **Truthiness / nil** — `be_truthy`, `be_falsey`, `be_nil`.
- **Comparison** — `be > / >= / < / <= / == / !=` (including RSpec's "a bit
  confusing" negated form for ordering operators).
- **Type** — `be_a` / `be_kind_of` (ancestor walk) and `be_instance_of`.
- **Strings & collections** — `match(/re/)`, `include`, `start_with` /
  `end_with`, `contain_exactly` / `match_array` (multiset diff with the
  missing/extra element diagnostics), `all(matcher)`.
- **Reflection** — `have_attributes`, `respond_to(:m).with(n).arguments`,
  predicate matchers (`be_empty` → `empty?`).
- **Numeric / range** — `be_within(delta).of(x)`, `cover`.
- **Blocks** — `change { … }.from/to/by/by_at_least/by_at_most` (the diff logic)
  and `raise_error(Class, message|/re/)`, driven by host observations.
- **Custom & composed** — `satisfy`, plus `.and` / `.or` composition (with the
  multi-line `...or:` message) and `not_to`.

### rspec-core structure model & formatters

- The **`describe`/`context`/`it` registration tree** with example metadata, the
  `before`/`after`/`around` + `let`/`subject` hooks (outer-before / inner-after
  ordering), **filtering** (`:focus`, include/exclude tags), and **ordering**
  (defined order or a seeded random permutation).
- The **progress** (`.` / `F` / `*`) and **documentation** formatters, plus the
  shared summary block: the pending section, the failures section, the totals
  line (`"N examples, M failures, P pending"` with RSpec's pluralisation), and
  the `Failed examples:` rerun list — all byte-faithful to a real `rspec` run.

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x) and three operating systems.

## Install

```sh
go get github.com/go-ruby-rspec/rspec
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-rspec/rspec"
)

func main() {
	// A matcher: match logic + byte-faithful RSpec failure message.
	m := rspec.Eq(4)
	ok, msg := rspec.Expect(3, m, true) // expect(3).to eq(4)
	fmt.Println(ok)                     // false
	fmt.Print(msg)
	// expected: 4
	//      got: 3
	// (compared using ==)

	// A structure tree + formatter, driven by host-supplied results.
	root := rspec.NewRootGroup("Calc")
	a := root.It("adds")
	a.Result = rspec.Result{Status: rspec.Passed}
	b := root.It("subtracts")
	b.Location = "./calc_spec.rb:5"
	b.Result = rspec.Result{
		Status:            rspec.Failed,
		FailureExpression: "expect(5 - 2).to eq(4)",
		FailureMessage:    msg,
	}

	r := rspec.NewReporter()
	r.Record(a)
	r.Record(b)
	fmt.Print(r.Render(rspec.Timing{Run: "0.01 seconds", Load: "0.03 seconds"}))
	// .F
	// …
	// 2 examples, 1 failure
}
```

## Ruby value model

Matcher messages embed the Ruby `#inspect` of the values under test, so this
package reproduces MRI's `inspect` for a small, fixed value model a host maps its
object graph onto:

| Ruby             | Go                                          |
| ---------------- | ------------------------------------------- |
| `nil`            | `nil`                                       |
| `true` / `false` | `bool`                                       |
| `Integer`        | `int…`, `uint…`, `*big.Int`                 |
| `Float`          | `float64`, `float32`                        |
| `String`         | `string`                                    |
| `Symbol`         | `rspec.Symbol`                              |
| `Array`          | `[]any`                                      |
| `Hash`           | `*rspec.Hash` (insertion-ordered)           |
| `Range`          | `*rspec.Range`                              |
| `Regexp`         | `*rspec.Regexp`                             |
| `Class`/`Module` | `rspec.Class` / `rspec.Module`              |
| object           | `*rspec.Object` (class, ivars, `object_id`) |

The reflection matchers (`have_attributes`, `respond_to`, predicates) resolve
against the built-in model directly; for host objects they route through the
overridable `AttrReader` / `Responder` / `Predicate` hooks so rbgo can dispatch
the call through the interpreter.

## The host seam

Deterministic pieces live here; **evaluating Ruby is the host's job.** The
block-form matchers take an *observation*:

```go
// expect { a[0] += 3 }.to change { a[0] }.by(5)
obs := rspec.Change{ExprName: "a[0]", Before: 0, After: 3} // host runs the block + probe
ok, msg := rspec.Expect(nil, rspec.ChangeObserved(obs).By(5), true)
// ok == false, msg == "expected `a[0]` to have changed by 5, but was changed by 3"
```

Likewise `raise_error` consumes a `rspec.RaisedError{Class, Message}` the host
reports after running the example block, and the `Reporter` is fed `Result`s the
host produces by running each `it` body.

## Tests & coverage

The suite pairs deterministic, ruby-free golden-vector tests (which alone hold
coverage at 100%, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential RSpec oracle**: each matcher is run on both sides — here and the
real `rspec-expectations` gem — on pass and fail actuals, and the `matches?`
result plus both failure-message strings are compared byte-for-byte; the totals
line is checked against `rspec-core`'s `SummaryNotification`. The oracle skips
itself where `ruby` / the rspec gems are absent and gates on `RUBY_VERSION >=
"4.0"`.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-rspec/rspec authors.
