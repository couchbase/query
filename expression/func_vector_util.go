/*
Copyright 2026-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

package expression

import (
	"math"
)

// widens an fp16 value to float32 without loss.
func FP16ToFP32(h uint16) float32 {
	sign := uint32(h>>15) << 31 // 1 bit
	exp := uint32(h>>10) & 0x1f // 5 bits
	mant := uint32(h) & 0x3ff   // 10 bits
	switch {
	case exp == 0x1f: // exp is invalid
		return math.Float32frombits(sign | 0xff<<23 | mant<<13)
	case exp != 0: // rebias exp from 15 to 127
		return math.Float32frombits(sign | (exp+112)<<23 | mant<<13)
	}
	f := float32(mant) / (1 << 24)
	if sign != 0 {
		f = -f
	}
	return f
}

// widens a bf16 value to float32 without loss.
func BF16ToFP32(h uint16) float32 {
	return math.Float32frombits(uint32(h) << 16)
}

// rounds f to a 16-bit float with expBits exponent bits (ties to even); false for NaN, Inf or overflow.
func fp64To16(f float64, expBits uint) (uint16, bool) {
	u := math.Float64bits(f)
	mantBits := 15 - expBits
	maxExp := 1<<expBits - 1
	sign := uint16(u>>48) & 0x8000
	mant := u & (1<<52 - 1)
	exp := int(u>>52&0x7ff) - 1023 + maxExp/2 // re-bias
	switch {
	case u>>52&0x7ff == 0x7ff:
		return 0, false // NaN or Inf
	case exp >= maxExp:
		return 0, false // too large
	case exp >= 1:
		h := uint64(exp)<<mantBits + roundShiftEven(mant, 52-mantBits)
		if h >= uint64(maxExp)<<mantBits {
			return 0, false // rounded up to Inf
		}
		return sign | uint16(h), true
	case exp >= -int(mantBits):
		return sign | uint16(roundShiftEven(mant|1<<52, 53-mantBits-uint(exp))), true // subnormal
	default:
		return sign, true // rounds to zero
	}
}

func FP64ToFP16(f float64) (uint16, bool) { return fp64To16(f, 5) }
func FP64ToBF16(f float64) (uint16, bool) { return fp64To16(f, 8) }

// returns x >> s, rounded to nearest with ties to even.
func roundShiftEven(x uint64, s uint) uint64 {
	m := x >> s
	rem := x & (uint64(1)<<s - 1)
	half := uint64(1) << (s - 1)
	if rem > half || (rem == half && m&1 == 1) {
		m++
	}
	return m
}
