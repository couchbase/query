//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"encoding/base64"
	"encoding/binary"
	"math"
	"math/rand"
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
		{"decode omitted vs default false, 'dense'",
			NewDecodeVector(str),
			NewDecodeVector(str, NewConstant(false), NewConstant("dense")), true},
		{"decode omitted vs default false, 'DENSE' (case-insensitive)",
			NewDecodeVector(str),
			NewDecodeVector(str, NewConstant(false), NewConstant("DENSE")), true},
		{"decode 'dense' vs 'Dense'",
			NewDecodeVector(str, NewConstant(false), NewConstant("dense")),
			NewDecodeVector(str, NewConstant(false), NewConstant("Dense")), true},
		{"decode false vs false, 'dense'",
			NewDecodeVector(str, NewConstant(false)),
			NewDecodeVector(str, NewConstant(false), NewConstant("dense")), true},
		{"decode omitted vs 'sparse'",
			NewDecodeVector(str),
			NewDecodeVector(str, NewConstant(false), NewConstant("sparse")), false},
		{"decode 'dense' vs 'sparse'",
			NewDecodeVector(str, NewConstant(false), NewConstant("dense")),
			NewDecodeVector(str, NewConstant(false), NewConstant("sparse")), false},
		{"decode omitted vs true, 'dense'",
			NewDecodeVector(str),
			NewDecodeVector(str, NewConstant(true), NewConstant("dense")), false},
		{"decode omitted vs non-constant vector type",
			NewDecodeVector(str),
			NewDecodeVector(str, NewConstant(false), other), false},
		{"encode omitted vs default false, 'Dense'",
			NewEncodeVector(vec),
			NewEncodeVector(vec, NewConstant(false), NewConstant("Dense")), true},
		{"encode omitted vs 'sparse'",
			NewEncodeVector(vec),
			NewEncodeVector(vec, NewConstant(false), NewConstant("sparse")), false},
		{"nested: all defaults written out",
			NewDecodeVector(NewEncodeVector(vec)),
			NewDecodeVector(NewEncodeVector(vec, NewConstant(false), NewConstant("dense")),
				NewConstant(false), NewConstant("DENSE")), true},
		{"decode omitted vs default 'FP32'",
			NewDecodeVector(str),
			NewDecodeVector(str, NewConstant(false), NewConstant("dense"), NewConstant("FP32")), true},
		{"decode omitted vs 'fp16'",
			NewDecodeVector(str),
			NewDecodeVector(str, NewConstant(false), NewConstant("dense"), NewConstant("fp16")), false},
		{"decode 'fp16' vs 'bf16'",
			NewDecodeVector(str, NewConstant(false), NewConstant("dense"), NewConstant("fp16")),
			NewDecodeVector(str, NewConstant(false), NewConstant("dense"), NewConstant("bf16")), false},
		{"encode omitted vs default 'fp32'",
			NewEncodeVector(vec),
			NewEncodeVector(vec, NewConstant(false), NewConstant("dense"), NewConstant("fp32")), true},
		{"encode omitted vs 'bf16'",
			NewEncodeVector(vec),
			NewEncodeVector(vec, NewConstant(false), NewConstant("dense"), NewConstant("bf16")), false},
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

// MB-75348: a sparse vector is N uint32 indices followed by N float32 values,
// so its length must be a multiple of 8 bytes. Any other length gives NULL
// instead of a truncated vector.
func TestDecodeSparseVectorLength(t *testing.T) {
	tests := []struct {
		name string
		b64  string
		want value.Value
	}{
		{"16 bytes: indices [1, 2], values [0.5, 1.5]", "AQAAAAIAAAAAAAA/AADAPw==",
			value.NewValue([]any{[]any{1, 2}, []any{0.5, 1.5}})},
		{"empty", "", value.NewValue([]any{[]any{}, []any{}})},
		{"12 bytes", "AQAAAAIAAAAAAAA/", value.NULL_VALUE},
		{"4 bytes", "BwAAAA==", value.NULL_VALUE},
	}

	for _, tt := range tests {
		got, err := NewDecodeVector(NewConstant(tt.b64), NewConstant(false), NewConstant(_SPARSE)).Evaluate(nil, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error %v", tt.name, err)
		}
		if got.Type() != tt.want.Type() || got.Collate(tt.want) != 0 {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

var vectorElemTypes = []string{"fp32", "fp16", "bf16"}

var unitRoundoff = map[string]float64{
	"fp32": math.Ldexp(1, -24),
	"fp16": math.Ldexp(1, -11),
	"bf16": math.Ldexp(1, -8),
}

var vectorCodecCases = []struct {
	name      string
	vec       any
	bigEndian bool
	vType     string
	elemType  string
	b64       string
}{
	{"dense fp32 LE", []any{1.0, -2.5}, false, _DENSE, "fp32", "AACAPwAAIMA="},
	{"dense fp16 LE", []any{1.0, -2.5}, false, _DENSE, "fp16", "ADwAwQ=="},
	{"dense bf16 LE", []any{1.0, -2.5}, false, _DENSE, "bf16", "gD8gwA=="},
	{"dense fp32 BE", []any{1.0, -2.5}, true, _DENSE, "fp32", "P4AAAMAgAAA="},
	{"dense fp16 BE", []any{1.0, -2.5}, true, _DENSE, "fp16", "PADBAA=="},
	{"dense bf16 BE", []any{1.0, -2.5}, true, _DENSE, "bf16", "P4DAIA=="},
	{"sparse fp32 LE", []any{[]any{3, 7}, []any{1.0, -2.5}}, false, _SPARSE, "fp32", "AwAAAAcAAAAAAIA/AAAgwA=="},
	{"sparse fp16 LE", []any{[]any{3, 7}, []any{1.0, -2.5}}, false, _SPARSE, "fp16", "AwAAAAcAAAAAPADB"},
	{"sparse bf16 LE", []any{[]any{3, 7}, []any{1.0, -2.5}}, false, _SPARSE, "bf16", "AwAAAAcAAACAPyDA"},
	{"sparse fp16 BE", []any{[]any{3, 7}, []any{1.0, -2.5}}, true, _SPARSE, "fp16", "AAAAAwAAAAc8AMEA"},
	{"sparse fp16 12 bytes", []any{[]any{1, 0}, []any{0.0, 0.0}}, false, _SPARSE, "fp16", "AQAAAAAAAAAAAAAA"},
	{"sparse bf16 12 bytes", []any{[]any{1, 0}, []any{0.0, 0.0}}, false, _SPARSE, "bf16", "AQAAAAAAAAAAAAAA"},
}

func evalVectorFunc(t *testing.T, f Function) value.Value {
	t.Helper()
	rv, err := f.Evaluate(nil, nil)
	if err != nil {
		t.Fatalf("%s: unexpected error %v", f.Name(), err)
	}
	return rv
}

func constants(args []any) []Expression {
	exprs := make([]Expression, len(args))
	for i, a := range args {
		exprs[i] = NewConstant(a)
	}
	return exprs
}

func decodeVector(t *testing.T, args ...any) value.Value {
	t.Helper()
	return evalVectorFunc(t, NewDecodeVector(constants(args)...))
}

func encodeVector(t *testing.T, args ...any) value.Value {
	t.Helper()
	return evalVectorFunc(t, NewEncodeVector(constants(args)...))
}

func toFloats(t *testing.T, v value.Value) []float64 {
	t.Helper()
	if v.Type() != value.ARRAY {
		t.Fatalf("expected an array, got %v", v)
	}
	var rv []float64
	for i := 0; ; i++ {
		e, ok := v.Index(i)
		if !ok {
			return rv
		}
		rv = append(rv, value.AsNumberValue(e).Float64())
	}
}

// roundTrip encodes src to base64 and then decodes it back.
func roundTrip(t *testing.T, src []float64, elemType string) []float64 {
	t.Helper()
	vec := make([]any, len(src))
	for i, f := range src {
		vec[i] = f
	}
	b64 := encodeVector(t, vec, false, _DENSE, elemType)
	if b64.Type() != value.STRING {
		t.Fatalf("%s: ENCODE_VECTOR returned %v", elemType, b64)
	}
	dec := toFloats(t, decodeVector(t, b64.Actual(), false, _DENSE, elemType))
	if len(dec) != len(src) {
		t.Fatalf("%s: %d values in, %d values out", elemType, len(src), len(dec))
	}
	return dec
}

func randomVector(n int) []float64 {
	r := rand.New(rand.NewSource(75111))
	rv := make([]float64, n)
	for i := range rv {
		rv[i] = float64(float32(math.Exp2(-14 + r.Float64()*(math.Log2(65504)+14))))
		if r.Intn(2) == 0 {
			rv[i] = -rv[i]
		}
	}
	return rv
}

// decoding known base64 strings gives the expected vectors
func TestDecodeVectorElemTypes(t *testing.T) {
	for _, c := range vectorCodecCases {
		got := decodeVector(t, c.b64, c.bigEndian, c.vType, c.elemType)
		if want := value.NewValue(c.vec); got.Collate(want) != 0 {
			t.Errorf("%s: got %v, want %v", c.name, got, want)
		}
	}

	want := value.NewValue([]any{1.0, -2.5})
	if got := decodeVector(t, "AACAPwAAIMA="); got.Collate(want) != 0 {
		t.Errorf("default element type: got %v, want fp32 %v", got, want)
	}
	if got := decodeVector(t, "ADwAwQ==", false, _DENSE, "FP16"); got.Collate(want) != 0 {
		t.Errorf("upper-case element type: got %v, want %v", got, want)
	}
}

// bad input to DECODE_VECTOR gives NULL
func TestDecodeVectorInvalidInputIsNull(t *testing.T) {
	cases := []struct{ name, b64, vType, elemType string }{
		{"unknown element type", "ADwAwQ==", _DENSE, "fp64"},
		{"fp32 length not a multiple of 4", "ADwAwQA=", _DENSE, "fp32"},
		{"fp16 length not a multiple of 2", "AA==", _DENSE, "fp16"},
		{"bf16 length not a multiple of 2", "AAAA", _DENSE, "bf16"},
		{"sparse fp32 length not a multiple of 8", "AwAAAAcAAAAAAIA/", _SPARSE, "fp32"},
		{"sparse fp32 12 bytes", "AQAAAAAAAAAAAAAA", _SPARSE, "fp32"},
		{"sparse fp16 length not a multiple of 6", "AACAPwAAIMA=", _SPARSE, "fp16"},
		{"sparse bf16 length not a multiple of 6", "AACAPwAAIMA=", _SPARSE, "bf16"},
		{"not base64", "!!!!", _DENSE, "fp16"},
	}
	for _, c := range cases {
		if got := decodeVector(t, c.b64, false, c.vType, c.elemType); got.Type() != value.NULL {
			t.Errorf("%s: got %v, want NULL", c.name, got)
		}
	}
}

// fp16 loses more precision than fp32, and bf16 loses more than fp16
func TestVectorLossByElemType(t *testing.T) {
	// 0.1 can't be stored exactly, so each type stores a slightly different value.
	want := map[string]float64{
		"fp32": float64(float32(0.1)),
		"fp16": 0.0999755859375,
		"bf16": 0.10009765625,
	}
	for _, et := range vectorElemTypes {
		if got := roundTrip(t, []float64{0.1}, et)[0]; got != want[et] {
			t.Errorf("%s: 0.1 came back as %v, want %v", et, got, want[et])
		}
	}

	// For every value, the error must be fp32 <= fp16 <= bf16.
	src := randomVector(10000)
	dec32 := roundTrip(t, src, "fp32")
	dec16 := roundTrip(t, src, "fp16")
	decbf := roundTrip(t, src, "bf16")
	var sum32, sum16, sumbf float64
	for i, x := range src {
		e32, e16, ebf := math.Abs(dec32[i]-x), math.Abs(dec16[i]-x), math.Abs(decbf[i]-x)
		if e32 > e16 || e16 > ebf {
			t.Fatalf("%v: errors fp32 %g, fp16 %g, bf16 %g are not in order", x, e32, e16, ebf)
		}
		sum32 += e32 / math.Abs(x)
		sum16 += e16 / math.Abs(x)
		sumbf += ebf / math.Abs(x)
	}
	if !(sum32 < sum16 && sum16 < sumbf) {
		t.Errorf("total relative error fp32 %g, fp16 %g, bf16 %g: want fp32 < fp16 < bf16",
			sum32, sum16, sumbf)
	}
}

// encoding and then decoding a vector gives values close to the original
func TestDecodeVectorCloseToSource(t *testing.T) {
	closeEnough := map[string]float64{"fp32": 1e-6, "fp16": 1e-3, "bf16": 1e-2}
	src := []float64{0.1, -1.0 / 3, math.Pi, 1e-3, -12345.678, 0.7071067811865476}
	srcAny := make([]any, len(src))
	for i, f := range src {
		srcAny[i] = f
	}
	indices := []any{0, 5, 9, 100, 4096, 70000}

	for _, et := range vectorElemTypes {
		dense := toFloats(t, decodeVector(t, encodeVector(t, srcAny, false, _DENSE, et).Actual(), false, _DENSE, et))
		sparseB64 := encodeVector(t, []any{indices, srcAny}, true, _SPARSE, et).Actual()
		sparse := decodeVector(t, sparseB64, true, _SPARSE, et)
		gotIndices, _ := sparse.Index(0)
		if gotIndices.Collate(value.NewValue(indices)) != 0 {
			t.Errorf("%s: sparse indices %v, want %v", et, gotIndices, indices)
		}
		sparseValues, _ := sparse.Index(1)

		for name, dec := range map[string][]float64{"dense": dense, "sparse": toFloats(t, sparseValues)} {
			if len(dec) != len(src) {
				t.Fatalf("%s %s: %d values, want %d", et, name, len(dec), len(src))
			}
			for i, x := range src {
				if math.Abs(dec[i]-x) > closeEnough[et]*math.Abs(x) {
					t.Errorf("%s %s: %v decoded as %v", et, name, x, dec[i])
				}
			}
		}
	}
}

// the error after encoding and decoding stays within each type's limit
func TestVectorLossWithinTheoreticalBound(t *testing.T) {
	src := randomVector(10000)
	for _, et := range vectorElemTypes {
		dec := roundTrip(t, src, et)
		for i, x := range src {
			if math.Abs(dec[i]-x) > unitRoundoff[et]*math.Abs(x) {
				t.Errorf("%s: |%v - %v| is above the bound %g×|x|", et, dec[i], x, unitRoundoff[et])
			}
		}
	}

	// Tiny fp16 values (below 2^-14) use a fixed step, so allow half a step.
	r := rand.New(rand.NewSource(1))
	sub := make([]float64, 1000)
	for i := range sub {
		sub[i] = float64(float32(r.Float64() * math.Ldexp(1, -14)))
	}
	dec := roundTrip(t, sub, "fp16")
	for i, x := range sub {
		if math.Abs(dec[i]-x) > math.Ldexp(1, -25) {
			t.Errorf("fp16 subnormal: |%v - %v| is above 2^-25", dec[i], x)
		}
	}
}

// FP64ToFP16 and FP64ToBF16 pick the closest value; on a tie, the even one
func TestFP64To16BitRoundsToNearestEven(t *testing.T) {
	formats := []struct {
		name      string
		encode    func(float64) (uint16, bool)
		decode    func(uint16) float32
		maxFinite uint16
	}{
		{"fp16", FP64ToFP16, FP16ToFP32, 0x7bff},
		{"bf16", FP64ToBF16, BF16ToFP32, 0x7f7f},
	}

	r := rand.New(rand.NewSource(3))
	inputs := []float64{
		math.NaN(), math.Inf(1), math.Inf(-1),
		math.MaxFloat64, math.Copysign(0, -1), math.SmallestNonzeroFloat64,
		65519.999, 65520, // float32 would turn 65519.999 into 65520
		math.Ldexp(1, -25), math.Nextafter(math.Ldexp(1, -25), 1), // 2^-25, and just above it
	}
	for i := 0; i < 1000000; i++ {
		inputs = append(inputs, math.Float64frombits(r.Uint64()))
		inputs = append(inputs, math.Ldexp(r.Float64(), r.Intn(44)-28))
		// halfway between two bf16 values
		inputs = append(inputs, float64(math.Float32frombits(uint32(r.Intn(0xffff))<<16|0x8000)))
	}
	// halfway between each pair of fp16 values, and the values just next to it
	for h := uint16(0); h < 0x7bff; h++ {
		mid := (float64(FP16ToFP32(h)) + float64(FP16ToFP32(h+1))) / 2
		for _, x := range []float64{mid, math.Nextafter(mid, 0), math.Nextafter(mid, math.Inf(1))} {
			inputs = append(inputs, x, -x)
		}
	}

	for _, fm := range formats {
		maxV := float64(fm.decode(fm.maxFinite))
		overflowAt := maxV + (maxV-float64(fm.decode(fm.maxFinite-1)))/2
		for _, x := range inputs {
			h, ok := fm.encode(x)
			// A value at or above overflowAt is too big and must fail.
			if wantOK := math.Abs(x) < overflowAt; ok != wantOK {
				t.Fatalf("%s %v: ok=%v, want %v", fm.name, x, ok, wantOK)
			}
			if !ok {
				continue
			}
			if (h&0x8000 != 0) != math.Signbit(x) {
				t.Fatalf("%s %v: %#04x has the wrong sign", fm.name, x, h)
			}
			d := math.Abs(x - float64(fm.decode(h)))
			mag := h & 0x7fff
			for _, n := range []uint16{mag - 1, mag + 1} {
				if (n == mag-1 && mag == 0) || n > fm.maxFinite {
					continue
				}
				dn := math.Abs(x - float64(fm.decode(h&0x8000|n)))
				if dn < d || (dn == d && h&1 == 1) {
					t.Fatalf("%s %v: got %#04x, but %#04x is nearer or even",
						fm.name, x, h, h&0x8000|n)
				}
			}
		}
	}
}

// encoding known vectors gives the expected base64 strings
func TestEncodeVectorElemTypes(t *testing.T) {
	for _, c := range vectorCodecCases {
		got := encodeVector(t, c.vec, c.bigEndian, c.vType, c.elemType)
		if got.Actual() != c.b64 {
			t.Errorf("%s: got %v, want %q", c.name, got, c.b64)
		}
	}
	if got := encodeVector(t, []any{1.0, -2.5}); got.Actual() != "AACAPwAAIMA=" {
		t.Errorf("default element type: got %v, want fp32 encoding", got)
	}
	// Values below 65520 round down to 65504, the biggest fp16 value.
	for _, f := range []float64{65519, 65519.999} {
		if got := encodeVector(t, []any{f}, false, _DENSE, "fp16"); got.Actual() != "/3s=" {
			t.Errorf("fp16 %v: got %v, want 65504 (\"/3s=\")", f, got)
		}
	}
}

// bad input to ENCODE_VECTOR gives NULL
func TestEncodeVectorInvalidInputIsNull(t *testing.T) {
	cases := []struct {
		name     string
		vec      []any
		elemType string
	}{
		{"unknown element type", []any{1.0}, "fp64"},
		{"non-number element", []any{1.0, "x"}, "fp16"},
		{"fp32 overflow", []any{1e39}, "fp32"},
		{"fp16 overflow", []any{70000}, "fp16"},
		{"fp16 rounds up to Inf", []any{65520}, "fp16"},
		{"bf16 rounds up to Inf", []any{3.4e38}, "bf16"},
	}
	for _, c := range cases {
		if got := encodeVector(t, c.vec, false, _DENSE, c.elemType); got.Type() != value.NULL {
			t.Errorf("%s: got %v, want NULL", c.name, got)
		}
	}
	if got := encodeVector(t, []any{70000}, false, _DENSE, "bf16"); got.Type() != value.STRING {
		t.Errorf("bf16 70000 is in range: got %v", got)
	}
}

// decoding and then encoding a base64 string gives back the same string
func TestEncodeVectorOfDecodeMatchesSource(t *testing.T) {
	for _, c := range vectorCodecCases {
		vec := decodeVector(t, c.b64, c.bigEndian, c.vType, c.elemType)
		if got := encodeVector(t, vec, c.bigEndian, c.vType, c.elemType); got.Actual() != c.b64 {
			t.Errorf("%s: got %v, want %q", c.name, got, c.b64)
		}
	}

	// Try every 16-bit value except NaN, Inf and -0 (decoding turns -0 into 0).
	finite16 := map[string]func(uint16) bool{
		"fp16": func(h uint16) bool { return h&0x7c00 != 0x7c00 && h != 0x8000 },
		"bf16": func(h uint16) bool { return h&0x7f80 != 0x7f80 && h != 0x8000 },
	}
	for et, keep := range finite16 {
		for _, bigEndian := range []bool{false, true} {
			order := binary.AppendByteOrder(binary.LittleEndian)
			if bigEndian {
				order = binary.BigEndian
			}
			var raw []byte
			for h := 0; h <= math.MaxUint16; h++ {
				if keep(uint16(h)) {
					raw = order.AppendUint16(raw, uint16(h))
				}
			}
			assertEncodeOfDecode(t, et, bigEndian, raw)
		}
	}

	r := rand.New(rand.NewSource(2))
	var raw []byte
	for len(raw) < 4*100000 {
		b := r.Uint32()
		if b&0x7f800000 != 0x7f800000 && b != 0x80000000 {
			raw = binary.LittleEndian.AppendUint32(raw, b)
		}
	}
	assertEncodeOfDecode(t, "fp32", false, raw)
}

func assertEncodeOfDecode(t *testing.T, elemType string, bigEndian bool, raw []byte) {
	t.Helper()
	src := base64.StdEncoding.EncodeToString(raw)
	vec := decodeVector(t, src, bigEndian, _DENSE, elemType)
	got := encodeVector(t, vec, bigEndian, _DENSE, elemType)
	if got.Actual() == src {
		return
	}
	gotRaw, _ := base64.StdEncoding.DecodeString(got.Actual().(string))
	for i := range raw {
		if i >= len(gotRaw) || gotRaw[i] != raw[i] {
			t.Errorf("%s bigEndian=%v: first difference at byte %d", elemType, bigEndian, i)
			return
		}
	}
	t.Errorf("%s bigEndian=%v: %d bytes in, %d bytes out", elemType, bigEndian, len(raw), len(gotRaw))
}
