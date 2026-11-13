//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"testing"

	"github.com/couchbase/query/value"
)

func TestVectorFuncEquivalent(t *testing.T) {
	vec := NewIdentifier("vec")
	str := NewIdentifier("s")
	other := NewIdentifier("b")

	tests := []struct {
		name string
		a, b Expression
		want bool
	}{
		{"decode same", NewDecodeVector(str), NewDecodeVector(str), true},
		{"decode omitted vs default false", NewDecodeVector(str), NewDecodeVector(str, NewConstant(false)), true},
		{"decode omitted vs true", NewDecodeVector(str), NewDecodeVector(str, NewConstant(true)), false},
		{"decode omitted vs non-constant", NewDecodeVector(str), NewDecodeVector(str, other), false},
		{"decode omitted vs null", NewDecodeVector(str), NewDecodeVector(str, NewConstant(nil)), false},
		{"decode false vs true", NewDecodeVector(str, NewConstant(false)), NewDecodeVector(str, NewConstant(true)), false},
		{"decode different first operand", NewDecodeVector(str), NewDecodeVector(vec, NewConstant(false)), false},
		{"decode first operand case-sensitive", NewDecodeVector(NewConstant("AAAAAA==")), NewDecodeVector(NewConstant("aaaaaa==")), false},
		{"encode omitted vs default false", NewEncodeVector(vec), NewEncodeVector(vec, NewConstant(false)), true},
		{"encode omitted vs true", NewEncodeVector(vec), NewEncodeVector(vec, NewConstant(true)), false},
		{"nested: inner default written out",
			NewDecodeVector(NewEncodeVector(vec)),
			NewDecodeVector(NewEncodeVector(vec, NewConstant(false))), true},
		{"nested: both defaults written out",
			NewDecodeVector(NewEncodeVector(vec)),
			NewDecodeVector(NewEncodeVector(vec, NewConstant(false)), NewConstant(false)), true},
		{"nested: inner true",
			NewDecodeVector(NewEncodeVector(vec)),
			NewDecodeVector(NewEncodeVector(vec, NewConstant(true)), NewConstant(false)), false},
		{"decode vs encode", NewDecodeVector(vec), NewEncodeVector(vec), false},
		{"isvector same",
			NewIsVector(Expressions{vec, NewConstant(128), NewConstant("float32")}),
			NewIsVector(Expressions{vec, NewConstant(128), NewConstant("float32")}), true},
		{"isvector type case-insensitive",
			NewIsVector(Expressions{vec, NewConstant(128), NewConstant("FLOAT32")}),
			NewIsVector(Expressions{vec, NewConstant(128), NewConstant("float32")}), true},
		{"isvector different dimension",
			NewIsVector(Expressions{vec, NewConstant(128), NewConstant("FLOAT32")}),
			NewIsVector(Expressions{vec, NewConstant(64), NewConstant("float32")}), false},
		{"isvector different field",
			NewIsVector(Expressions{vec, NewConstant(128), NewConstant("FLOAT32")}),
			NewIsVector(Expressions{other, NewConstant(128), NewConstant("float32")}), false},
		{"isvector different type",
			NewIsVector(Expressions{vec, NewConstant(128), NewConstant("float32")}),
			NewIsVector(Expressions{vec, NewConstant(128), NewConstant("float16")}), false},
		{"isvector non-constant type",
			NewIsVector(Expressions{vec, NewConstant(128), NewConstant("float32")}),
			NewIsVector(Expressions{vec, NewConstant(128), other}), false},
	}

	for _, tt := range tests {
		if got := tt.a.EquivalentTo(tt.b); got != tt.want {
			t.Errorf("%s: %s EquivalentTo %s = %v, want %v", tt.name, tt.a, tt.b, got, tt.want)
		}
		if got := tt.b.EquivalentTo(tt.a); got != tt.want {
			t.Errorf("%s (reversed): %s EquivalentTo %s = %v, want %v", tt.name, tt.b, tt.a, got, tt.want)
		}
	}
}

// Guard: each vector function with optional arguments must have one entry in its
// defaults table per argument (nil for required ones, the default value for
// optional ones). Otherwise a call that spells out a default, e.g.
// decode_vector(s, false, 'dense') on master, silently stops matching an index
// on decode_vector(s). If an optional argument is added, or this is merged into
// a branch where the function takes more arguments, extend the table.
func TestVectorFuncDefaultsComplete(t *testing.T) {
	x := NewIdentifier("x")

	tables := []struct {
		name     string
		fn       Function
		defaults value.Values
	}{
		{"_DECODE_VECTOR_DEFAULTS", NewDecodeVector(x), _DECODE_VECTOR_DEFAULTS},
		{"_ENCODE_VECTOR_DEFAULTS", NewEncodeVector(x), _ENCODE_VECTOR_DEFAULTS},
	}

	for _, tt := range tables {
		if len(tt.defaults) != tt.fn.MaxArgs() {
			t.Errorf("%s has %d entries, but %s() takes up to %d arguments; add a default for each optional argument",
				tt.name, len(tt.defaults), tt.fn.Name(), tt.fn.MaxArgs())
			continue
		}
		for i, d := range tt.defaults {
			if required := i < tt.fn.MinArgs(); required != (d == nil) {
				t.Errorf("%s[%d] = %v: entry must be nil exactly for the %d required argument(s) of %s()",
					tt.name, i, d, tt.fn.MinArgs(), tt.fn.Name())
			}
		}
	}
}
