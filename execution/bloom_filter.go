//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"math"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
)

// basic bloom filter implementation

const _WORD_SIZE = uint(64)
const _WORD_SIZE_MINUS_ONE = uint(63)
const _NUM_LOCATIONS = 5
const _LARGE_FILTER_SIZE = 1000000

type BloomFilter struct {
	elements  []uint64
	size      uint
	locations int
}

// A full bloom filter with a false-positives rate of 1/p uses roughly
// 1.44log2(p) bits per element
func newBloomFilter(n int) *BloomFilter {
	if n > math.MaxInt32 {
		return nil
	}

	var size uint
	var length, minLen uint32
	var locations int
	if n > _LARGE_FILTER_SIZE {
		minLen = uint32(0.127*float64(n) + 0.5)
		locations = _NUM_LOCATIONS + 1
	} else {
		minLen = uint32(0.150*float64(n) + 0.5)
		locations = _NUM_LOCATIONS
	}

	// length is power of 2
	length = 1
	for length < minLen {
		length *= 2
	}
	size = uint(length) * _WORD_SIZE

	return &BloomFilter{
		elements:  make([]uint64, length),
		size:      size,
		locations: locations,
	}
}

func (this *BloomFilter) Add(data []byte) {
	sizeMinusOne := this.size - 1
	h1, h2 := util.MurmurHashSum128(data)
	for i := 0; i < this.locations; i++ {
		h1 += h2
		pos := uint(h1) & sizeMinusOne
		this.elements[pos/_WORD_SIZE] |= 1 << (pos & _WORD_SIZE_MINUS_ONE)
	}
}

func (this *BloomFilter) Test(data []byte) bool {
	sizeMinusOne := this.size - 1
	h1, h2 := util.MurmurHashSum128(data)
	for i := 0; i < this.locations; i++ {
		h1 += h2
		pos := uint(h1) & sizeMinusOne
		if (this.elements[pos/_WORD_SIZE] & (1 << (pos & _WORD_SIZE_MINUS_ONE))) == 0 {
			return false
		}
	}
	return true
}

// in-place merge
func (this *BloomFilter) Merge(other *BloomFilter) errors.Error {
	if len(this.elements) != len(other.elements) || this.size != other.size {
		return errors.NewExecutionInternalError("BloomFilter.Merge: incompatible bloom filters")
	}

	for i, _ := range this.elements {
		this.elements[i] |= other.elements[i]
	}

	return nil
}
