// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

// The Ruby value model shared with the sibling go-ruby-* libraries. RSpec's
// matcher and formatter messages embed the Ruby `#inspect` of the values under
// test, so this package reproduces MRI's `inspect` byte-for-byte for the value
// shapes a host maps onto — the deterministic, interpreter-independent core.
//
// A host hands the matchers plain Go values (nil, bool, int/int64/*big.Int,
// float64, string) and the explicit wrapper types below for the Ruby shapes Go
// has no direct analogue for (Symbol, Array via []any, Hash via *Hash, Object,
// Range, Class/Module, Regexp). Running the actual example / hook blocks that
// produce these values is the host's job (the rbgo eval seam).

// Symbol is a Ruby Symbol (`:name`). Its inspect is the leading-colon form.
type Symbol string

// Class is a Ruby Class reference; inspect is the bare class name.
type Class string

// Module is a Ruby Module reference; inspect is the bare module name.
type Module string

// Regexp is a Ruby Regexp literal; inspect is `/source/flags`.
type Regexp struct {
	Source string
	Flags  string
}

// Range is a Ruby Range; inspect is `begin..end` (or `begin...end` when
// Exclusive), matching MRI.
type Range struct {
	Begin     any
	End       any
	Exclusive bool
}

// Object is an opaque Ruby object of the given Class. RSpecClass reports the
// class name matchers such as be_a / be_instance_of test against; IVars and the
// object identity (ID) let the host reproduce `#<Class:0x…>` style inspects and
// have_attributes / respond_to reflection. RespondsTo lists the message names
// the object answers to (for respond_to and predicate matchers).
type Object struct {
	Class      string
	IVars      map[string]any
	Order      []string
	ID         uint64 // object_id for equal? identity; 0 means "use pointer identity"
	RespondsTo []string
}

// Pair is one insertion-ordered Hash entry.
type Pair struct {
	Key, Val any
}

// Hash is an insertion-ordered Ruby Hash. Go maps have no stable order, so the
// matchers and inspect use this to reproduce MRI's key order faithfully.
type Hash struct {
	pairs []Pair
	index map[any]int
}

// NewHash returns an empty ordered Hash.
func NewHash() *Hash { return &Hash{index: map[any]int{}} }

// Set inserts or updates key→val, preserving first-insertion order.
func (h *Hash) Set(key, val any) {
	if h.index == nil {
		h.index = map[any]int{}
	}
	if i, ok := h.index[key]; ok {
		h.pairs[i].Val = val
		return
	}
	h.index[key] = len(h.pairs)
	h.pairs = append(h.pairs, Pair{key, val})
}

// Get returns the value stored for key.
func (h *Hash) Get(key any) (any, bool) {
	i, ok := h.index[key]
	if !ok {
		return nil, false
	}
	return h.pairs[i].Val, true
}

// Pairs returns the entries in insertion order.
func (h *Hash) Pairs() []Pair { return h.pairs }

// Len reports the number of entries.
func (h *Hash) Len() int { return len(h.pairs) }

// Inspect returns the Ruby `#inspect` of v, byte-faithful to MRI 4.0 for the
// value model this package supports. It is the string RSpec embeds in matcher
// and formatter messages.
func Inspect(v any) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case bool:
		if x {
			return "true"
		}
		return "false"
	case string:
		return inspectString(x)
	case Symbol:
		return ":" + string(x)
	case int:
		return strconv.Itoa(x)
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint:
		return strconv.FormatUint(uint64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case *big.Int:
		return x.String()
	case float32:
		return inspectFloat(float64(x))
	case float64:
		return inspectFloat(x)
	case Class:
		return string(x)
	case Module:
		return string(x)
	case *Regexp:
		return "/" + x.Source + "/" + x.Flags
	case *Range:
		sep := ".."
		if x.Exclusive {
			sep = "..."
		}
		return Inspect(x.Begin) + sep + Inspect(x.End)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = Inspect(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *Hash:
		return inspectHash(x.pairs)
	case map[string]any:
		// Deterministic: Ruby preserves insertion order, but a Go map has none,
		// so sort by key for a stable inspect (the host uses *Hash when order
		// matters).
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]Pair, len(keys))
		for i, k := range keys {
			pairs[i] = Pair{k, x[k]}
		}
		return inspectHash(pairs)
	case *Object:
		return inspectObject(x)
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func inspectHash(pairs []Pair) string {
	if len(pairs) == 0 {
		return "{}"
	}
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		// MRI 3.4+ prints Symbol keys as `key: val` and everything else as
		// `key => val`.
		if sym, ok := p.Key.(Symbol); ok {
			parts[i] = string(sym) + ": " + Inspect(p.Val)
		} else {
			parts[i] = Inspect(p.Key) + " => " + Inspect(p.Val)
		}
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func inspectObject(o *Object) string {
	id := o.ID
	if len(o.IVars) == 0 {
		return fmt.Sprintf("#<%s:0x%016x>", o.Class, id)
	}
	keys := o.Order
	if len(keys) == 0 {
		keys = make([]string, 0, len(o.IVars))
		for k := range o.IVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	}
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + Inspect(o.IVars[k])
	}
	return fmt.Sprintf("#<%s:0x%016x %s>", o.Class, id, strings.Join(parts, ", "))
}

// inspectFloat reproduces Ruby Float#inspect: infinities, NaN, and the shortest
// round-tripping decimal with a mandatory fractional part.
func inspectFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	s := strconv.FormatFloat(f, 'g', -1, 64)
	// Ruby always shows a decimal point (1.0, not 1) and uses e-notation with a
	// sign and at least two exponent digits differently; strconv's 'g' already
	// round-trips, we just ensure a fractional part for plain integers.
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

// inspectString reproduces Ruby String#inspect: double-quoted with the standard
// escapes RSpec messages rely on.
func inspectString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '\x00':
			b.WriteString(`\0`)
		case '\a':
			b.WriteString(`\a`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\v':
			b.WriteString(`\v`)
		case '\x1b':
			b.WriteString(`\e`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\x%02X`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
