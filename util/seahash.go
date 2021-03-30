//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

// based on seahash, see https://ticki.github.io/blog/seahash-explained

const (
	seed1  = 0x16f11fe89b0d677c
	seed2  = 0xb480a793d8e6c86c
	seed3  = 0x6fe2e5aaf078ebc9
	seed4  = 0x14f994a4c5259381
	pcgRNG = 0x6eed0e9da4d94a4f
)

// Given a byte slice, compute a 64 bit checksum (hash value)
func SeaHashSum64(b []byte) uint64 {
	s := NewSeaHash()
	s.write(b)
	return diffuse(s.a ^ s.b ^ s.c ^ s.d ^ uint64(s.inputLen))
}

type seaHash struct {
	a uint64
	b uint64
	c uint64
	d uint64

	inputLen uint64
}

func NewSeaHash() *seaHash {
	return &seaHash{
		a: seed1,
		b: seed2,
		c: seed3,
		d: seed4,
	}
}

func (this *seaHash) write(p []byte) {
	var i int
	// handle chucks of 8
	for ; i < len(p)-7; i += 8 {
		this.update(readInt64(p[i : i+8]))
	}
	// handle any remaining bytes
	if i < len(p) {
		this.update(readInt64(p[i:]))
	}

	this.inputLen += uint64(len(p))
	return
}

func (this *seaHash) update(x uint64) {
	a := diffuse(this.a ^ x)

	this.a = this.b
	this.b = this.c
	this.c = this.d
	this.d = a
}

func diffuse(x uint64) uint64 {
	x *= pcgRNG
	a, b := x>>32, x>>60
	x ^= a >> b
	x *= pcgRNG
	return x
}

// caller to ensure len(b) <= 8
func readInt64(b []uint8) uint64 {
	var x uint64

	for i := len(b) - 1; i >= 0; i-- {
		x <<= 8
		x |= uint64(b[i])
	}

	return x
}
