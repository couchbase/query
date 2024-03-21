//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// MurmurHash3 implementation adapted from C++ implementation
// github.com/aappleby/smhasher/blob/master/src/MurmurHash3.cpp

package util

const (
	c1_128 = 0x87c37b91114253d5
	c2_128 = 0x4cf5ad432745937f
)

func MurmurHashSum128(data []byte) (h1, h2 uint64) {
	var k1, k2 uint64
	length := len(data)

	for len(data) >= 16 {
		k1 = uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24 | uint64(data[4])<<32 |
			uint64(data[5])<<40 | uint64(data[6])<<48 | uint64(data[7])<<56
		k2 = uint64(data[8]) | uint64(data[9])<<8 | uint64(data[10])<<16 | uint64(data[11])<<24 | uint64(data[12])<<32 |
			uint64(data[13])<<40 | uint64(data[14])<<48 | uint64(data[15])<<56
		data = data[16:]

		k1 *= c1_128
		k1 = (k1 << 31) | (k1 >> 33)
		k1 *= c2_128
		h1 ^= k1

		h1 = (h1 << 27) | (h1 >> 37)
		h1 += h2
		h1 = h1*5 + 0x52dce729

		k2 *= c2_128
		k2 = (k2 << 33) | (k2 >> 31)
		k2 *= c1_128
		h2 ^= k2

		h2 = (h2 << 31) | (h2 >> 33)
		h2 += h1
		h2 = h2*5 + 0x38495ab5
	}

	// tail
	k1 = 0
	k2 = 0
	switch len(data) {
	case 15:
		k2 ^= uint64(data[14]) << 48
		fallthrough
	case 14:
		k2 ^= uint64(data[13]) << 40
		fallthrough
	case 13:
		k2 ^= uint64(data[12]) << 32
		fallthrough
	case 12:
		k2 ^= uint64(data[11]) << 24
		fallthrough
	case 11:
		k2 ^= uint64(data[10]) << 16
		fallthrough
	case 10:
		k2 ^= uint64(data[9]) << 8
		fallthrough
	case 9:
		k2 ^= uint64(data[8]) << 0
		k2 *= c2_128
		k2 = (k2 << 33) | (k2 >> 31)
		k2 *= c1_128
		h2 ^= k2
		fallthrough
	case 8:
		k1 ^= uint64(data[7]) << 56
		fallthrough
	case 7:
		k1 ^= uint64(data[6]) << 48
		fallthrough
	case 6:
		k1 ^= uint64(data[5]) << 40
		fallthrough
	case 5:
		k1 ^= uint64(data[4]) << 32
		fallthrough
	case 4:
		k1 ^= uint64(data[3]) << 24
		fallthrough
	case 3:
		k1 ^= uint64(data[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint64(data[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint64(data[0]) << 0
		k1 *= c1_128
		k1 = (k1 << 31) | (k1 >> 33)
		k1 *= c2_128
		h1 ^= k1
	}

	h1 ^= uint64(length)
	h2 ^= uint64(length)
	h1 += h2
	h2 += h1
	h1, h2 = fmix(h1), fmix(h2)
	h1 += h2
	h2 += h1
	return
}

func fmix(k uint64) uint64 {
	k ^= k >> 33
	k *= 0xff51afd7ed558ccd
	k ^= k >> 33
	k *= 0xc4ceb9fe1a85ec53
	k ^= k >> 33
	return k
}

func MurmurHashSum64(data []byte) uint64 {
	h1, _ := MurmurHashSum128(data)
	return h1
}
