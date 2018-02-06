//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
