//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
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
	MIN_HASH_TABLE_SIZE = 1024
	MAX_HASH_TABLE_SIZE = 16777216
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
	hashVal   value.Value   // values used for hashing
	inputVals []value.Value // payload
}

func newHashEntry(hashKey uint64, hashVal, inputVal value.Value) *hashEntry {
	inputVals := []value.Value{inputVal}
	return &hashEntry{hashKey, hashVal, inputVals}
}

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
	entries []*hashEntry // the hash table
	count   int          // number of entries in hash table so far
	mode    int          // hash table mode
	bucket  int          // when iterate or get, bucket position
	vector  int          // when iterate or get, vector position
}

func NewHashTable() *HashTable {
	rv := &HashTable{
		mode:   HASH_TABLE_INIT,
		bucket: -1,
		vector: -1,
	}
	rv.entries = make([]*hashEntry, MIN_HASH_TABLE_SIZE)
	return rv
}

// given a hash value (hashVal), put the inputVal into hash table
func (this *HashTable) Put(hashVal, inputVal value.Value) error {
	this.mode = HASH_TABLE_PUT

	if this.loadFactor() >= HTLoadThreshold {
		err := this.Grow()
		if err != nil {
			return err
		}
	}

	// we don't expect count to overflow since we would have max out on hash table size
	this.count++

	hashKey, err := this.getHashKey(hashVal)
	if err != nil {
		return err
	}

	hashEntry := newHashEntry(hashKey, hashVal, inputVal)

	return this.putEntry(hashEntry)
}

func (this *HashTable) putEntry(entry *hashEntry) error {
	// use quadratic probing to find available slot in hash table
	// since the hash table size is power of 2, the mod operation can be
	// achieved by bitwise and
	size_minus_one := uint64(len(this.entries) - 1)
	idx := int(entry.hashKey & size_minus_one)
	found := false
	for i := 0; i < len(this.entries); i++ {
		e := this.entries[idx]
		if e != nil {
			if e.hashKey == entry.hashKey && e.hashVal.Equals(entry.hashVal).Truth() {
				// should not come here if hash table is doubling,
				// since the entire vector is inherited previously
				if this.mode == HASH_TABLE_GROW {
					return errors.NewExecutionInternalError("HashTable.putEntry: unexpected state")
				}
				e.inputVals = append(e.inputVals, entry.inputVals...)
				found = true
				break
			} else {
				idx = int(uint64(idx+i+1) & size_minus_one)
			}
		} else {
			this.entries[idx] = entry
			found = true
			break
		}
	}

	if !found {
		return errors.NewExecutionInternalError("HashTable.putEntry: did not find slot in hash table")
	}

	return nil
}

// give a hash value (hashVal), get the first output value associated with that hash value
func (this *HashTable) Get(hashVal value.Value) (value.Value, error) {
	this.mode = HASH_TABLE_GET

	hashKey, err := this.getHashKey(hashVal)
	if err != nil {
		return nil, err
	}

	size_minus_one := uint64(len(this.entries) - 1)
	idx := int(hashKey & size_minus_one)
	for i := 0; i < len(this.entries); i++ {
		e := this.entries[idx]
		if e != nil {
			if e.hashKey == hashKey && e.hashVal.Equals(hashVal).Truth() {
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
	return nil, errors.NewExecutionInternalError("HashTable.Get: unexpected traversal of hash table")
}

// after initial Get() call, return any additional values associated with the same hash value
func (this *HashTable) GetNext() (value.Value, error) {
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

	return nil, errors.NewExecutionInternalError("HashTable.GetNext: not following a Get call")
}

func (this *HashTable) getHashKey(hashVal value.Value) (uint64, error) {
	bytes, err := hashVal.MarshalJSON()
	if err != nil {
		return 0, err
	}

	hashKey := util.SeaHashSum64(bytes)

	return hashKey, nil
}

func (this *HashTable) Iterate() value.Value {
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
	return float64(this.count) / float64(len(this.entries))
}

func (this *HashTable) Grow() error {
	prevMode := this.mode
	defer func() { this.mode = prevMode }()
	this.mode = HASH_TABLE_GROW

	newSize := len(this.entries) * 2
	if newSize > MAX_HASH_TABLE_SIZE {
		return errors.NewHashTableMaxSizeExceeded()
	}

	oldEntries := this.entries
	this.entries = make([]*hashEntry, newSize)

	for _, entry := range oldEntries {
		if entry != nil {
			err := this.putEntry(entry)
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

func (this *HashTable) Drop() {
	this.mode = HASH_TABLE_DROP
	for i := 0; i < len(this.entries); i++ {
		if this.entries[i] != nil {
			this.entries[i].inputVals = nil
			this.entries[i] = nil
		}
	}
	this.count = 0
}
