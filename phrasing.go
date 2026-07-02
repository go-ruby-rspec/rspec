// Copyright (c) the go-ruby-rspec/rspec authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rspec

import "strings"

// andList joins inspected items with RSpec's EnglishPhrasing.list rules:
//
//	[]           -> ""
//	[a]          -> "a"
//	[a, b]       -> "a and b"
//	[a, b, c, …] -> "a, b, and c"  (Oxford comma)
func andList(items []any) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = Inspect(it)
	}
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}
}
