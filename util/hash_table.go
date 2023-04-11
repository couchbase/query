//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"fmt"
	"math"
)

// an implementation of hash table loosely based on google's densehash
// assumptions:
//   - hash table has only insertion operation and no deletion operation
//   - hash table uses quadratic probing
//   - hash table doubles in size when threshold is met
//   - no synchronization is provided, it assumes hash table is in insertion phase first,
//     then probing phase
//   - hash table is in memory only (currently)

// min and max size of a hash table
// use power of 2 as hash table sizes
// the max size (2 ^ 24) is somewhat arbitrary
const (
	MIN_HASH_TABLE_SIZE_INLIST    = 32
	MIN_HASH_TABLE_SIZE_HASH_JOIN = 1024
	MAX_HASH_TABLE_SIZE           = 16777216
)

// this is the load threadhold that triggers an enlargement of the hash table
// from google dense hash table:
// NUMBER OF PROBES / LOOKUP       Successful            Unsuccessful
// Quadratic collision resolution   1 - ln(1-L) - L/2    1/(1-L) - L - ln(1-L)
// Linear collision resolution     [1+1/(1-L)]/2         [1+1/(1-L)2]/2
//
// -- enlarge_factor --           0.10  0.50  0.60  0.75  0.80  0.90  0.99
// QUADRATIC COLLISION RES.
//    probes/successful lookup    1.05  1.44  1.62  2.01  2.21  2.85  5.11
//    probes/unsuccessful lookup  1.11  2.19  2.82  4.64  5.81  11.4  103.6
// LINEAR COLLISION RES.
//    probes/successful lookup    1.06  1.5   1.75  2.5   3.0   5.5   50.5
//    probes/unsuccessful lookup  1.12  2.5   3.6   8.5   13.0  50.0  5000.0
//
// we use quadratic collision resolution

var HTLoadThreshold = 0.75

type hashEntry struct {
	hashKey   uint64        // generated hash code
	hashVal   interface{}   // values used for hashing
	inputVals []interface{} // payload
}

func newHashEntry(hashKey uint64, hashVal, inputVal interface{}) *hashEntry {
	inputVals := _ARR_POOL_1.Get()
	inputVals = append(inputVals, inputVal)
	return &hashEntry{hashKey, hashVal, inputVals}
}

func newArrayHashEntry(hashKey uint64, hashVal []interface{}, inputVal interface{}) *hashEntry {
	inputVals := _ARR_POOL_1.Get()
	inputVals = append(inputVals, inputVal)
	hashVals := _ARR_POOL_4.GetCapped(len(hashVal))
	hashVals = append(hashVals, hashVal...)
	return &hashEntry{hashKey, hashVals, inputVals}
}

// hash table purpose
const (
	HASH_TABLE_FOR_HASH_JOIN = iota
	HASH_TABLE_FOR_INLIST
)

// hash table mode
const (
	HASH_TABLE_INIT = iota
	HASH_TABLE_PUT
	HASH_TABLE_GET
	HASH_TABLE_GROW
	HASH_TABLE_ITERATE
	HASH_TABLE_DROP
)

type HashTable struct {
	entries  []*hashEntry // the hash table
	count    int          // number of entries in hash table so far
	distinct int          // number of distinct values in hash table so far
	mode     int          // hash table mode
	bucket   int          // when iterate or get, bucket position
	vector   int          // when iterate or get, vector position
	size     uint64       // size in byte of the hash table
	array    bool         // using value array?
}

func NewHashTable(purpose int, cardinality float64, arrLen int) *HashTable {
	rv := &HashTable{
		mode:   HASH_TABLE_INIT,
		bucket: -1,
		vector: -1,
		array:  arrLen > 1,
	}
	defSize := MIN_HASH_TABLE_SIZE_HASH_JOIN
	if purpose == HASH_TABLE_FOR_INLIST {
		defSize = MIN_HASH_TABLE_SIZE_INLIST
	}
	size := int(math.Ceil(cardinality / HTLoadThreshold))
	if size <= defSize {
		// this includes the case where size is not valid (i.e. -1)
		size = defSize
	} else {
		size = 1 << int(math.Ceil(math.Log2(float64(size))))
		if size > MAX_HASH_TABLE_SIZE {
			size = MAX_HASH_TABLE_SIZE
		}
	}
	rv.entries = make([]*hashEntry, size)
	return rv
}

// given a hash value (hashVal), put the inputVal into hash table
func (this *HashTable) Put(hashVal, inputVal interface{}, marshal func(interface{}) ([]byte, error),
	equal func(val1, val2 interface{}) bool, size uint64) error {

	this.mode = HASH_TABLE_PUT

	if this.loadFactor() >= HTLoadThreshold {
		err := this.Grow(equal)
		if err != nil {
			return err
		}
	}

	hashKey, err := this.getHashKey(hashVal, marshal)
	if err != nil {
		return err
	}

	var entry *hashEntry
	if this.array {
		if arr, ok := hashVal.([]interface{}); ok {
			entry = newArrayHashEntry(hashKey, arr, inputVal)
		} else {
			return fmt.Errorf("HashTable.Put: expecting an array, not %T", hashVal)
		}
	} else {
		entry = newHashEntry(hashKey, hashVal, inputVal)
	}

	err = this.putEntry(entry, equal)
	if err == nil {
		this.size += size
	}
	return err
}

func (this *HashTable) putEntry(entry *hashEntry, equal func(val1, val2 interface{}) bool) error {
	// use quadratic probing to find available slot in hash table
	// since the hash table size is power of 2, the mod operation can be
	// achieved by bitwise and
	size_minus_one := uint64(len(this.entries) - 1)
	idx := int(entry.hashKey & size_minus_one)
	this.count += len(entry.inputVals)
	found := false
	for i := 0; i < len(this.entries); i++ {
		e := this.entries[idx]
		if e != nil {
			if e.hashKey == entry.hashKey && equal(e.hashVal, entry.hashVal) {
				// should not come here if hash table is doubling,
				// since the entire vector is inherited previously
				if this.mode == HASH_TABLE_GROW {
					return fmt.Errorf("HashTable.putEntry: unexpected state")
				}
				e.inputVals = append(e.inputVals, entry.inputVals...)
				found = true
				break
			} else {
				idx = int(uint64(idx+i+1) & size_minus_one)
			}
		} else {
			this.entries[idx] = entry
			this.distinct++
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("HashTable.putEntry: did not find slot in hash table")
	}

	return nil
}

// give a hash value (hashVal), get the first output value associated with that hash value
func (this *HashTable) Get(hashVal interface{}, marshal func(interface{}) ([]byte, error),
	equal func(val1, val2 interface{}) bool) (interface{}, error) {

	this.mode = HASH_TABLE_GET

	hashKey, err := this.getHashKey(hashVal, marshal)
	if err != nil {
		return nil, err
	}

	size_minus_one := uint64(len(this.entries) - 1)
	idx := int(hashKey & size_minus_one)
	for i := 0; i < len(this.entries); i++ {
		e := this.entries[idx]
		if e != nil {
			if e.hashKey == hashKey && equal(e.hashVal, hashVal) {
				if len(e.inputVals) > 1 {
					this.bucket = idx
					this.vector = 1
				}
				return e.inputVals[0], nil
			} else {
				idx = int(uint64(idx+i+1) & size_minus_one)
			}
		} else {
			return nil, nil
		}
	}

	// should have either found the entry or stopped looking (finding nil)
	return nil, fmt.Errorf("HashTable.Get: unexpected traversal of hash table")
}

// after initial Get() call, return any additional values associated with the same hash value
func (this *HashTable) GetNext() (interface{}, error) {
	if this.mode == HASH_TABLE_GET {
		if this.bucket >= 0 && this.vector >= 0 {
			// if previous get call left a position in the hash table, use that position
			e := this.entries[this.bucket]
			v := e.inputVals[this.vector]
			this.vector++
			if this.vector >= len(e.inputVals) {
				this.bucket = -1
				this.vector = -1
			}
			return v, nil
		} else {
			// no more duplicated hash values
			return nil, nil
		}
	}

	return nil, fmt.Errorf("HashTable.GetNext: not following a Get call")
}

func (this *HashTable) getHashKey(hashVal interface{}, marshal func(interface{}) ([]byte, error)) (uint64, error) {
	bytes, err := marshal(hashVal)
	if err != nil {
		return 0, err
	}

	hashKey := SeaHashSum64(bytes)

	return hashKey, nil
}

func (this *HashTable) Iterate() interface{} {
	if this.mode != HASH_TABLE_ITERATE {
		this.mode = HASH_TABLE_ITERATE
		this.bucket = 0
		this.vector = 0
	}

	for this.bucket < len(this.entries) {
		e := this.entries[this.bucket]
		if e == nil {
			this.bucket++
			continue
		}
		v := e.inputVals[this.vector]
		this.vector++
		if this.vector >= len(e.inputVals) {
			this.vector = 0
			this.bucket++
		}

		return v
	}

	return nil
}

func (this *HashTable) loadFactor() float64 {
	return float64(this.distinct) / float64(len(this.entries))
}

func (this *HashTable) Grow(equal func(val1, val2 interface{}) bool) error {
	prevMode := this.mode
	defer func() { this.mode = prevMode }()
	this.mode = HASH_TABLE_GROW

	newSize := len(this.entries) * 2
	if newSize > MAX_HASH_TABLE_SIZE {
		return fmt.Errorf(fmt.Sprintf("Maximum hash table size %d exceeded", MAX_HASH_TABLE_SIZE))
	}

	this.count = 0
	this.distinct = 0
	oldEntries := this.entries
	this.entries = make([]*hashEntry, newSize)

	for _, entry := range oldEntries {
		if entry != nil {
			err := this.putEntry(entry, equal)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// size of hash table (number of buckets)
func (this *HashTable) NumBuckets() int {
	return len(this.entries)
}

// number of entries in the hash table
func (this *HashTable) Count() int {
	return this.count
}

// cumulative size of the items in the hash table
func (this *HashTable) Size() uint64 {
	return this.size
}

func (this *HashTable) Drop() {
	this.mode = HASH_TABLE_DROP
	for i := 0; i < len(this.entries); i++ {
		if this.entries[i] != nil {
			hashVal := this.entries[i].hashVal
			if this.array {
				_ARR_POOL_4.Put(hashVal.([]interface{}))
			}
			this.entries[i].hashVal = nil
			_ARR_POOL_1.Put(this.entries[i].inputVals)
			this.entries[i].inputVals = nil
			this.entries[i] = nil
		}
	}
	this.count = 0
	this.distinct = 0
	this.size = 0
}

const (
	NUMBER_NOT_AVAIL = -1.0
	_ERR_MARGIN      = 0.0000000000001
)

// calculate number of memcopies due to doubling of hash table
func GetNumMemCopy(purpose int, size float64) float64 {
	if size <= 0.0 {
		return NUMBER_NOT_AVAIL
	}

	var minN, maxN int
	var total float64

	switch purpose {
	case HASH_TABLE_FOR_HASH_JOIN:
		minN = 10 // 2 ^^ 10 = 1024
	case HASH_TABLE_FOR_INLIST:
		minN = 5 // 2 ^^ 5 = 32
	default:
		return NUMBER_NOT_AVAIL
	}

	tSize := size / HTLoadThreshold
	maxN = int(math.Log2(tSize))
	if (math.Pow(float64(maxN), 2) - tSize) < _ERR_MARGIN {
		maxN -= 1
	}

	if maxN > minN {
		total = math.Pow(float64(maxN), 2) - math.Pow(float64(minN), 2)
	}

	return total * HTLoadThreshold
}

var _ARR_POOL_1 = NewInterfacePool(1)
var _ARR_POOL_4 = NewInterfacePool(4)
