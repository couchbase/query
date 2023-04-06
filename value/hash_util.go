//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"fmt"
)

/*
 * A couple of helper functions for using hash table for values
 */

// when we know val is a single value
func MarshalValue(val interface{}) ([]byte, error) {
	hashVal := NewValue(val)
	return hashVal.MarshalJSON()
}

// when we know val is an array ([]interface{})
func MarshalArray(val interface{}) ([]byte, error) {
	if arr, ok := val.([]interface{}); ok {
		hashVal := NewValue(arr)
		return hashVal.MarshalJSON()
	}
	return nil, fmt.Errorf("MarshalArray: expecting array, not %T", val)
}

func EqualValue(val1, val2 interface{}) bool {
	value1 := NewValue(val1)
	value2 := NewValue(val2)
	return value1.Equals(value2).Truth()
}

func EqualArray(vals1, vals2 interface{}) bool {
	array1 := vals1.([]interface{})
	array2 := vals2.([]interface{})
	if len(array1) != len(array2) {
		return false
	}
	for i := 0; i < len(array1); i++ {
		value1 := NewValue(array1[i])
		value2 := NewValue(array2[i])
		if !value1.Equals(value2).Truth() {
			return false
		}
	}
	return true
}

func EqualValueMissingNull(val1, val2 interface{}) bool {
	value1 := NewValue(val1)
	value2 := NewValue(val2)
	type1 := value1.Type()
	type2 := value2.Type()
	if type1 == MISSING {
		return type2 == MISSING
	} else if type1 == NULL {
		return type2 == NULL
	}
	return value1.Equals(value2).Truth()
}

func EqualArrayMissingNull(vals1, vals2 interface{}) bool {
	array1 := vals1.([]interface{})
	array2 := vals2.([]interface{})
	if len(array1) != len(array2) {
		return false
	}
	for i := 0; i < len(array1); i++ {
		value1 := NewValue(array1[i])
		value2 := NewValue(array2[i])
		type1 := value1.Type()
		type2 := value2.Type()
		if type1 == MISSING {
			if type2 != MISSING {
				return false
			}
		} else if type1 == NULL {
			if type2 != NULL {
				return false
			}
		} else if !value1.Equals(value2).Truth() {
			return false
		}
	}
	return true
}
