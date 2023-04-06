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
	"strconv"
	"testing"
)

func getBytesInt(val interface{}) ([]byte, error) {
	i := val.(int)
	s := strconv.FormatInt(int64(i), 10)
	return []byte(s), nil
}

func getBytesStr(val interface{}) ([]byte, error) {
	s := val.(string)
	return []byte(s), nil
}

func equalInt(val1, val2 interface{}) bool {
	v1 := val1.(int)
	v2 := val2.(int)
	return v1 == v2
}

func equalStr(val1, val2 interface{}) bool {
	v1 := val1.(string)
	v2 := val2.(string)
	return v1 == v2
}

func TestHashTable(t *testing.T) {

	var count, dup int
	var e error
	var intVal int
	var strVal, inputVal1, inputVal2 string
	var outputVal interface{}

	// create a hash table
	htab := NewHashTable(HASH_TABLE_FOR_HASH_JOIN, 1)

	// insert values into hash table
	for i := 0; i < 4096; i++ {
		if (i & 0xfff) == 0 {
			dup = 25
		} else if (i & 0xff) == 0 {
			dup = 5
		} else {
			dup = 1
		}

		intVal = i
		strVal = fmt.Sprintf("this is string %d", i)
		for j := 0; j < dup; j++ {
			inputVal1 = fmt.Sprintf("this is payload value for int hash value i = %d j = %d", i, j)
			inputVal2 = fmt.Sprintf("this is payload value for string hash value i = %d j = %d", i, j)

			e = htab.Put(intVal, inputVal1, getBytesInt, equalInt, 0)
			if e != nil {
				t.Errorf("PUT of int value failed, i = %d j = %d", i, j)
			}

			e = htab.Put(strVal, inputVal2, getBytesStr, equalStr, 0)
			if e != nil {
				t.Errorf("PUT of string value failed, i = %d j = %d", i, j)
			}
		}
	}

	// retrieve values from hash table
	for i := -2; i < 4100; i++ {
		if i < 0 || i >= 4096 {
			dup = 0
		} else if (i & 0xfff) == 0 {
			dup = 25
		} else if (i & 0xff) == 0 {
			dup = 5
		} else {
			dup = 1
		}

		intVal = i
		strVal = fmt.Sprintf("this is string %d", i)

		count = 0
		outputVal, e = htab.Get(intVal, getBytesInt, equalInt)
		if e != nil {
			t.Errorf("GET of int value failed, i = %d j = %d", i, count)
		}
		if outputVal != nil {
			count++
			for {
				outputVal, e = htab.GetNext()
				if e != nil {
					t.Errorf("GET of int value failed, i = %d j = %d", i, count)
				}

				if outputVal == nil {
					break
				}
				count++
			}
		}
		if count != dup {
			t.Errorf("Unexpected number of results for int value, expect %d get %d", dup, count)
		}

		count = 0
		outputVal, e = htab.Get(strVal, getBytesStr, equalStr)
		if e != nil {
			t.Errorf("GET of string value failed, i = %d j = %d", i, count)
		}
		if outputVal != nil {
			count++
			for {
				outputVal, e = htab.GetNext()
				if e != nil {
					t.Errorf("GET of string value failed, i = %d j = %d", i, count)
				}

				if outputVal == nil {
					break
				}
				count++
			}
		}
		if count != dup {
			t.Errorf("Unexpected number of results for string value, expect %d get %d", dup, count)
		}
	}

	// iterate through hash table
	count = 0
	for {
		outputVal = htab.Iterate()
		if outputVal == nil {
			break
		}

		count++
	}
	// should have 4180 intVal and 4180 strVal
	if count != 8360 {
		t.Errorf("Incorrect number of entries from Iterate(), expect 8360, get %d", count)
	}

	if htab.NumBuckets() != 16384 {
		t.Errorf("Incorrect number of buckets in hash table, expect 16384, get %d", htab.NumBuckets())
	}

	// drop the hash table
	htab.Drop()
}
